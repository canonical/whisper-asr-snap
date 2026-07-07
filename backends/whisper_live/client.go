package whisperlive

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const endOfAudioMarker = "END_OF_AUDIO"

type TranscriptionCallback func(text string, segments []Segment)
type TranslationCallback func(text string, segments []Segment)

type Segment struct {
	ID        any
	Start     float64
	End       float64
	Text      string
	Completed bool
	Raw       map[string]any
}

type Client struct {
	cfg ClientConfig

	mu                   sync.RWMutex
	recording            bool
	waiting              bool
	serverError          bool
	errorMessage         string
	language             string
	task                 string
	uid                  string
	serverBackend        string
	lastSegment          *Segment
	lastReceivedSegment  string
	lastResponseReceived time.Time
	transcript           []Segment
	translatedTranscript []Segment
	retryCount           int
	closedByUser         bool
	conn                 *websocket.Conn
	connected            chan struct{}
	connectedOnce        sync.Once
	runDone              chan struct{}
	writeCh              chan outboundMessage
}

type outboundMessage struct {
	messageType int
	payload     []byte
}

func NewClient(cfg ClientConfig) (*Client, error) {
	defaults := DefaultClientConfig()
	cfg = mergeConfig(defaults, cfg)

	if cfg.Host == nil || strings.TrimSpace(*cfg.Host) == "" {
		return nil, fmt.Errorf("host and port are required")
	}
	if cfg.Port == nil || *cfg.Port <= 0 {
		return nil, fmt.Errorf("host and port are required")
	}

	uid := strings.TrimSpace(valueOrZero(cfg.UID))
	if uid == "" {
		uid = newUID()
	}

	task := "transcribe"
	if valueOrZero(cfg.Translate) {
		task = "translate"
	}

	c := &Client{
		cfg:       cfg,
		language:  valueOrZero(cfg.Lang),
		task:      task,
		uid:       uid,
		connected: make(chan struct{}),
		runDone:   make(chan struct{}),
		writeCh:   make(chan outboundMessage, 256),
	}

	go c.run()
	return c, nil
}

func (c *Client) run() {
	defer close(c.runDone)

	for {
		if c.isClosedByUser() {
			return
		}

		if err := c.connectAndServe(); err != nil {
			if c.isClosedByUser() {
				return
			}
			c.setServerError(err.Error())
			return
		}

		if c.isClosedByUser() {
			return
		}

		if !c.shouldRetry() {
			return
		}

		c.mu.Lock()
		c.retryCount++
		currentRetry := c.retryCount
		maxRetries := valueOrZero(c.cfg.MaxRetries)
		retryDelay := valueOrZero(c.cfg.RetryDelay)
		c.mu.Unlock()

		if retryDelay > 0 {
			log.Printf("[INFO]: Reconnecting (%d/%d) in %s...", currentRetry, maxRetries, retryDelay)
			time.Sleep(retryDelay)
		}
	}
}

func (c *Client) connectAndServe() error {
	socketProtocol := "ws"
	if valueOrZero(c.cfg.UseWSS) {
		socketProtocol = "wss"
	}
	u := url.URL{Scheme: socketProtocol, Host: fmt.Sprintf("%s:%d", valueOrZero(c.cfg.Host), valueOrZero(c.cfg.Port))}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.waiting = false
	c.serverError = false
	c.errorMessage = ""
	c.mu.Unlock()

	log.Printf("[INFO]: Opened connection")

	if err := c.sendOpenConfig(); err != nil {
		_ = conn.Close()
		return fmt.Errorf("send initial config: %w", err)
	}

	writerStop := make(chan struct{})
	writerDone := make(chan struct{})
	go c.writeLoop(conn, writerStop, writerDone)
	defer func() {
		close(writerStop)
		_ = conn.Close()
		<-writerDone
		c.mu.Lock()
		if c.conn == conn {
			c.conn = nil
		}
		c.recording = false
		c.waiting = false
		c.mu.Unlock()
	}()

	for {
		messageType, payload, readErr := conn.ReadMessage()
		if readErr != nil {
			log.Printf("[INFO]: Websocket connection closed: %v", readErr)
			return nil
		}
		if messageType != websocket.TextMessage {
			continue
		}
		if err := c.handleMessage(payload); err != nil {
			log.Printf("[WARN]: message handling error: %v", err)
		}
	}
}

func (c *Client) writeLoop(conn *websocket.Conn, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	for {
		select {
		case msg := <-c.writeCh:
			if err := conn.WriteMessage(msg.messageType, msg.payload); err != nil {
				return
			}
		case <-stop:
			return
		case <-c.runDone:
			return
		}
	}
}

func (c *Client) sendOpenConfig() error {
	c.mu.RLock()
	payload := map[string]any{
		"uid":                   c.uid,
		"language":              emptyStringAsNil(c.language),
		"task":                  c.task,
		"model":                 valueOrZero(c.cfg.Model),
		"use_vad":               valueOrZero(c.cfg.UseVAD),
		"send_last_n_segments":  valueOrZero(c.cfg.SendLastNSegments),
		"no_speech_thresh":      valueOrZero(c.cfg.NoSpeechThresh),
		"clip_audio":            valueOrZero(c.cfg.ClipAudio),
		"same_output_threshold": valueOrZero(c.cfg.SameOutputThreshold),
		"enable_translation":    valueOrZero(c.cfg.EnableTranslation),
		"target_language":       valueOrZero(c.cfg.TargetLanguage),
		"hotwords":              emptyStringAsNil(valueOrZero(c.cfg.Hotwords)),
		"enable_diarization":    valueOrZero(c.cfg.EnableDiarization),
		"max_speakers":          valueOrZero(c.cfg.MaxSpeakers),
		"word_timestamps":       valueOrZero(c.cfg.WordTimestamps),
		"initial_prompt":        emptyStringAsNil(valueOrZero(c.cfg.InitialPrompt)),
		"vad_parameters":        c.cfg.VADParameters,
		"audio_format":          valueOrZero(c.cfg.AudioFormat),
	}
	c.mu.RUnlock()

	msg, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.enqueue(websocket.TextMessage, msg)
}

func (c *Client) handleMessage(raw []byte) error {
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	uid, _ := msg["uid"].(string)
	if uid != c.uid {
		log.Printf("[ERROR]: invalid client uid")
		return nil
	}

	if statusValue, ok := msg["status"]; ok {
		status, _ := statusValue.(string)
		c.handleStatus(status, msg)
		return nil
	}

	if message, _ := msg["message"].(string); message == "DISCONNECT" {
		log.Printf("[INFO]: Server disconnected due to overtime.")
		c.mu.Lock()
		c.recording = false
		c.mu.Unlock()
	}

	if message, _ := msg["message"].(string); message == "SERVER_READY" {
		backend, _ := msg["backend"].(string)
		c.mu.Lock()
		c.recording = true
		c.serverBackend = backend
		c.lastResponseReceived = time.Now()
		c.mu.Unlock()
		c.connectedOnce.Do(func() { close(c.connected) })
		log.Printf("[INFO]: Server Running with backend %s", backend)
		return nil
	}

	if langValue, ok := msg["language"]; ok {
		lang, _ := langValue.(string)
		langProb, _ := asFloat64(msg["language_prob"])
		c.mu.Lock()
		c.language = lang
		c.mu.Unlock()
		log.Printf("[INFO]: Server detected language %s with probability %.4f", lang, langProb)
		return nil
	}

	if rawSegments, ok := msg["segments"]; ok {
		segments := parseSegments(rawSegments)
		if len(segments) > 0 {
			c.processSegments(segments, false)
		}
	}

	if rawSegments, ok := msg["translated_segments"]; ok {
		segments := parseSegments(rawSegments)
		if len(segments) > 0 {
			c.processSegments(segments, true)
		}
	}

	return nil
}

func (c *Client) handleStatus(status string, messageData map[string]any) {
	switch status {
	case "WAIT":
		waitMinutes, _ := asFloat64(messageData["message"])
		c.mu.Lock()
		c.waiting = true
		c.mu.Unlock()
		log.Printf("[INFO]: Server is full. Estimated wait time %.0f minutes.", math.Round(waitMinutes))
	case "ERROR":
		msg, _ := messageData["message"].(string)
		c.setServerError(msg)
		log.Printf("Message from Server: %s", msg)
	case "WARNING":
		msg, _ := messageData["message"].(string)
		log.Printf("Message from Server: %s", msg)
	}
}

func (c *Client) processSegments(segments []Segment, translated bool) {
	if len(segments) == 0 {
		return
	}

	text := make([]string, 0, len(segments))

	c.mu.Lock()
	serverBackend := c.serverBackend
	for i, seg := range segments {
		if len(text) == 0 || text[len(text)-1] != seg.Text {
			text = append(text, strings.TrimSpace(seg.Text))
			if i == len(segments)-1 && !seg.Completed {
				copySeg := seg
				c.lastSegment = &copySeg
			} else if serverBackend == "faster_whisper" && seg.Completed {
				if translated {
					if len(c.translatedTranscript) == 0 || seg.Start >= c.translatedTranscript[len(c.translatedTranscript)-1].End {
						c.translatedTranscript = append(c.translatedTranscript, seg)
					}
				} else {
					if len(c.transcript) == 0 || seg.Start >= c.transcript[len(c.transcript)-1].End {
						c.transcript = append(c.transcript, seg)
					}
				}
			}
		}
	}

	if !translated {
		lastText := segments[len(segments)-1].Text
		if c.lastReceivedSegment == "" || c.lastReceivedSegment != lastText {
			c.lastResponseReceived = time.Now()
			c.lastReceivedSegment = lastText
		}
	}

	transcriptionCallback := c.cfg.TranscriptionCallback
	translationCallback := c.cfg.TranslationCallback
	logTranscription := valueOrZero(c.cfg.LogTranscription)
	targetLanguage := valueOrZero(c.cfg.TargetLanguage)
	transcriptCopy := append([]Segment(nil), c.transcript...)
	translatedCopy := append([]Segment(nil), c.translatedTranscript...)
	lastSegment := c.lastSegment
	displaySegments := valueOrZero(c.cfg.DisplaySegments)
	c.mu.Unlock()

	joined := strings.TrimSpace(strings.Join(text, " "))

	if translated {
		if translationCallback != nil {
			translationCallback(joined, segments)
			return
		}
	} else {
		if transcriptionCallback != nil {
			transcriptionCallback(joined, segments)
			return
		}
	}

	if !logTranscription {
		return
	}

	if displaySegments <= 0 {
		displaySegments = 4
	}

	if !translated {
		printable := tailSegmentTexts(transcriptCopy, displaySegments)
		if lastSegment != nil && !containsText(printable, lastSegment.Text) {
			printable = append(printable, lastSegment.Text)
		}
		log.Printf("[TRANSCRIPT] %s", strings.Join(printable, " "))
		if valueOrZero(c.cfg.EnableTranslation) {
			log.Printf("[TRANSLATION -> %s] %s", targetLanguage, strings.Join(tailSegmentTexts(translatedCopy, displaySegments), " "))
		}
	}
}

func (c *Client) SendPacketToServer(message []byte) error {
	if len(message) == 0 {
		return nil
	}
	return c.enqueue(websocket.BinaryMessage, message)
}

func (c *Client) SendFloat32Samples(samples []float32) error {
	if len(samples) == 0 {
		return nil
	}
	buf := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(sample))
	}
	return c.SendPacketToServer(buf)
}

func (c *Client) SendEndOfAudio() error {
	return c.SendPacketToServer([]byte(endOfAudioMarker))
}

func (c *Client) CloseWebSocket() error {
	c.mu.Lock()
	c.closedByUser = true
	conn := c.conn
	c.mu.Unlock()

	if conn != nil {
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(500*time.Millisecond))
		_ = conn.Close()
	}

	select {
	case <-c.runDone:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out waiting for websocket close")
	}
	return nil
}

func (c *Client) WaitUntilReady(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.connected:
		return nil
	}
}

func (c *Client) WaitBeforeDisconnect(ctx context.Context) error {
	c.mu.RLock()
	last := c.lastResponseReceived
	idleFor := valueOrZero(c.cfg.DisconnectAfterIdle)
	c.mu.RUnlock()

	if last.IsZero() {
		return fmt.Errorf("last response timestamp is not available")
	}
	if idleFor <= 0 {
		idleFor = 15 * time.Second
	}

	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.runDone:
			return nil
		case <-ticker.C:
			c.mu.RLock()
			elapsed := time.Since(c.lastResponseReceived)
			recording := c.recording
			c.mu.RUnlock()
			if !recording {
				// The backend ended the session (e.g. closed or disconnected);
				// there is nothing more to wait for.
				return nil
			}
			if elapsed >= idleFor {
				return nil
			}
		}
	}
}

// WaitForServerClose blocks until the backend closes the connection or the grace
// period elapses, whichever comes first. It gives the backend a short window to
// deliver a final flushed segment after END_OF_AUDIO before the client tears the
// connection down.
func (c *Client) WaitForServerClose(ctx context.Context, grace time.Duration) {
	if grace <= 0 {
		grace = 5 * time.Second
	}
	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-c.runDone:
	case <-timer.C:
	case <-ctx.Done():
	}
}

func (c *Client) GetClientSocket() *websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func (c *Client) UID() string {
	return c.uid
}

func (c *Client) Recording() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.recording
}

func (c *Client) Waiting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.waiting
}

func (c *Client) ServerError() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverError
}

func (c *Client) ErrorMessage() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.errorMessage
}

func (c *Client) Language() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.language
}

func (c *Client) ServerBackend() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.serverBackend
}

func (c *Client) Transcript() []Segment {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Segment(nil), c.transcript...)
}

// LastSegment returns a copy of the most recent in-progress (not yet completed)
// segment, or nil if there is none. This is the trailing partial transcript that
// has not been finalized by the backend.
func (c *Client) LastSegment() *Segment {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastSegment == nil {
		return nil
	}
	cp := *c.lastSegment
	return &cp
}

func (c *Client) TranslatedTranscript() []Segment {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]Segment(nil), c.translatedTranscript...)
}

func (c *Client) enqueue(messageType int, payload []byte) error {
	if c.isClosedByUser() {
		return errors.New("client is closed")
	}
	copyPayload := append([]byte(nil), payload...)
	select {
	case c.writeCh <- outboundMessage{messageType: messageType, payload: copyPayload}:
		return nil
	case <-c.runDone:
		return io.EOF
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out writing outbound message")
	}
}

func (c *Client) setServerError(msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.serverError = true
	c.errorMessage = msg
}

func (c *Client) shouldRetry() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	maxRetries := valueOrZero(c.cfg.MaxRetries)
	return maxRetries > 0 && c.retryCount < maxRetries && !c.serverError && !c.closedByUser
}

func (c *Client) isClosedByUser() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closedByUser
}

func parseSegments(raw any) []Segment {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	segments := make([]Segment, 0, len(list))
	for _, item := range list {
		asMap, ok := item.(map[string]any)
		if !ok {
			continue
		}
		start, _ := asFloat64(asMap["start"])
		end, _ := asFloat64(asMap["end"])
		text, _ := asMap["text"].(string)
		completed, _ := asMap["completed"].(bool)
		segments = append(segments, Segment{
			ID:        asMap["id"],
			Start:     start,
			End:       end,
			Text:      text,
			Completed: completed,
			Raw:       asMap,
		})
	}
	return segments
}

func asFloat64(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		n, err := t.Float64()
		if err == nil {
			return n, true
		}
	}
	return 0, false
}

func emptyStringAsNil(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func tailSegmentTexts(segments []Segment, n int) []string {
	if n <= 0 {
		n = 4
	}
	if len(segments) <= n {
		out := make([]string, 0, len(segments))
		for _, seg := range segments {
			out = append(out, seg.Text)
		}
		return out
	}
	start := len(segments) - n
	out := make([]string, 0, n)
	for _, seg := range segments[start:] {
		out = append(out, seg.Text)
	}
	return out
}

func containsText(lines []string, text string) bool {
	for _, line := range lines {
		if line == text {
			return true
		}
	}
	return false
}

func valueOrZero[T any](v *T) T {
	var zero T
	if v == nil {
		return zero
	}
	return *v
}

func newUID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
}
