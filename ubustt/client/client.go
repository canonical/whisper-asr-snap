package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"

	whisperlive "ubustt-proxy/backends/whisper_live"
	"ubustt-proxy/ubustt/messages"

	"github.com/gorilla/websocket"
)

// Client holds the per-connection state for a single UbuSTT user. Each client
// owns a dedicated WhisperLive backend session: audio frames received from the
// user are forwarded to WhisperLive, and transcription results coming back from
// WhisperLive are translated into UbuSTT events and pushed to the user.
type Client struct {
	Connection *websocket.Conn

	backendCfg whisperlive.Config
	backend    *whisperlive.Client
	ctx        context.Context // root context for the session lifetime

	writeMu sync.Mutex // serializes websocket writes to the user

	closeOnce sync.Once

	mu            sync.Mutex
	session       *messages.SessionUpdateSession
	itemSeq       int     // monotonic counter used to mint item ids
	currentItemID string  // open (in-progress) transcription item, if any
	emitted       string  // text already sent as deltas for the current item
	lastCompleted float64 // end time of the last completed segment we forwarded
}

// NewClient creates a Client bound to a user websocket connection. The backend
// configuration describes the WhisperLive session that will be opened for this
// user by Start.
func NewClient(conn *websocket.Conn, backendCfg whisperlive.Config) *Client {
	return &Client{Connection: conn, backendCfg: backendCfg}
}

// Start opens the WhisperLive backend session, waits until it is ready, and
// advertises the default session configuration to the user via session.created.
func (c *Client) Start(ctx context.Context) error {
	c.ctx = ctx

	cfg := c.backendCfg
	cfg.OnTranscription = c.onTranscription

	backend, err := whisperlive.Dial(ctx, cfg)
	if err != nil {
		return fmt.Errorf("connecting to whisper live backend: %w", err)
	}
	c.backend = backend

	if err := backend.WaitReady(ctx); err != nil {
		return fmt.Errorf("waiting for whisper live backend: %w", err)
	}

	rate := cfg.SampleRate
	if rate <= 0 {
		rate = 16000
	}
	created := messages.NewSessionCreated(cfg.Model, cfg.Lang, rate)

	c.mu.Lock()
	session := sessionFromCreated(created)
	c.session = &session
	c.mu.Unlock()

	if err := c.send(created); err != nil {
		return fmt.Errorf("sending session.created: %w", err)
	}

	// If the backend session ends (WhisperLive closed the connection, errored,
	// or we tore it down), close the user connection too.
	go c.watchBackend()
	return nil
}

// watchBackend blocks until the WhisperLive session ends and then closes the
// user-facing connection, unblocking the server read loop.
func (c *Client) watchBackend() {
	<-c.backend.Done()
	log.Printf("Whisper live backend closed, closing user connection")
	c.Close()
}

// Close tears down the backend session and the user connection. It is safe to
// call multiple times and from multiple goroutines.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		if c.backend != nil {
			_ = c.backend.Close()
		}
		_ = c.Connection.Close()
	})
	return nil
}

// HandleMessage decodes a raw text frame and dispatches it to the handler that
// owns the required client state.
func (c *Client) HandleMessage(payload []byte) error {
	msg, err := messages.FromJson(payload)
	if err != nil {
		return fmt.Errorf("decoding message: %w", err)
	}

	switch m := msg.(type) {
	case *messages.SessionUpdate:
		return c.handleSessionUpdate(m)
	case *messages.InputAudioBufferAppend:
		return c.handleInputAudioBufferAppend(m)
	case *messages.InputAudioBufferCommit:
		return c.handleInputAudioBufferCommit(m)
	default:
		return fmt.Errorf("unexpected inbound message type: %T", m)
	}
}

// handleSessionUpdate merges the requested session patch and acknowledges it
// with a session.updated event.
func (c *Client) handleSessionUpdate(m *messages.SessionUpdate) error {
	c.mu.Lock()
	session := m.Session
	c.session = &session
	c.mu.Unlock()

	// Note: changing model/language mid-stream would require reconnecting the
	// backend; that is out of scope here, we only echo the accepted state.
	return c.send(messages.NewSessionUpdated(session))
}

// handleInputAudioBufferAppend decodes a base64 PCM16 chunk and forwards it to
// the WhisperLive backend.
func (c *Client) handleInputAudioBufferAppend(m *messages.InputAudioBufferAppend) error {
	pcm, err := base64.StdEncoding.DecodeString(m.Audio)
	if err != nil {
		return c.send(messages.NewError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			"audio is not valid base64",
		))
	}
	if len(pcm) == 0 {
		return nil
	}
	if c.backend == nil {
		return fmt.Errorf("backend session not started")
	}
	if err := c.backend.SendPCM16(pcm); err != nil {
		return fmt.Errorf("forwarding audio to backend: %w", err)
	}
	return nil
}

// handleInputAudioBufferCommit signals end of audio input. It launches the
// finalization sequence in a goroutine: wait until the backend is idle (so the
// final segment is not dropped), send END_OF_AUDIO, then wait for the server to
// close — at which point watchBackend tears down the user connection.
func (c *Client) handleInputAudioBufferCommit(_ *messages.InputAudioBufferCommit) error {
	if c.backend == nil {
		return fmt.Errorf("backend session not started")
	}
	go func() {
		if err := c.backend.Finalize(c.ctx); err != nil {
			log.Printf("[WARN]: finalizing backend: %v", err)
		}
	}()
	return nil
}

// onTranscription is invoked on the backend read loop for every transcription
// update. It maps WhisperLive segments onto UbuSTT delta/completed events.
func (c *Client) onTranscription(_ string, segments []whisperlive.Segment) {
	if len(segments) == 0 {
		return
	}

	var outbound []messages.Message

	c.mu.Lock()
	for _, seg := range segments {
		if seg.Completed && seg.Start >= c.lastCompleted {
			itemID := c.currentOrNewItemLocked()
			outbound = append(outbound,
				messages.NewTranscriptionCompleted(itemID, 0, strings.TrimSpace(seg.Text)))
			c.lastCompleted = seg.End
			c.rotateItemLocked()
		}
	}

	// A trailing, not-yet-completed segment is streamed as an incremental delta.
	last := segments[len(segments)-1]
	if !last.Completed {
		itemID := c.currentOrNewItemLocked()
		if delta := suffix(last.Text, c.emitted); delta != "" {
			outbound = append(outbound,
				messages.NewTranscriptionDelta(itemID, 0, delta))
			c.emitted = last.Text
		}
	}
	c.mu.Unlock()

	for _, msg := range outbound {
		if err := c.send(msg); err != nil {
			log.Printf("[WARN]: sending transcription event: %v", err)
			return
		}
	}
}

// currentOrNewItemLocked returns the id of the open transcription item, minting
// a new one if none is open. Callers must hold c.mu.
func (c *Client) currentOrNewItemLocked() string {
	if c.currentItemID == "" {
		c.itemSeq++
		c.currentItemID = fmt.Sprintf("item_%d", c.itemSeq)
		c.emitted = ""
	}
	return c.currentItemID
}

// rotateItemLocked closes the current item so the next partial starts a fresh
// one. Callers must hold c.mu.
func (c *Client) rotateItemLocked() {
	c.currentItemID = ""
	c.emitted = ""
}

// send serializes an outbound message and writes it to the user websocket.
func (c *Client) send(m messages.Message) error {
	data, err := messages.ToJson(m)
	if err != nil {
		return fmt.Errorf("encoding message: %w", err)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.Connection.WriteMessage(websocket.TextMessage, data)
}

// suffix returns the part of text that follows prev, or the whole text when prev
// is not a prefix (the backend revised the partial rather than extending it).
func suffix(text, prev string) string {
	if prev != "" && strings.HasPrefix(text, prev) {
		return text[len(prev):]
	}
	return text
}

// sessionFromCreated seeds local session state from the advertised defaults so
// later session.update patches merge against a known baseline.
func sessionFromCreated(m *messages.SessionCreated) messages.SessionUpdateSession {
	return messages.SessionUpdateSession{
		Type:         m.Session.Type,
		Instructions: m.Session.Instructions,
		Prompt:       m.Session.Prompt,
		Audio: messages.SessionUpdateAudio{
			Input: messages.SessionUpdateAudioInput{
				Format: messages.SessionUpdateFormat{Rate: m.Session.Audio.Input.Format.Rate},
				Transcription: messages.SessionUpdateTranscription{
					Model:    m.Session.Audio.Input.Transcription.Model,
					Language: m.Session.Audio.Input.Transcription.Language,
				},
			},
		},
		Include: m.Session.Include,
	}
}
