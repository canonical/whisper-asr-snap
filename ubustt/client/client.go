package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"sync"

	"ubustt-proxy/backends"
	"ubustt-proxy/ubustt/messages"

	"github.com/gorilla/websocket"
)

// Client holds the per-connection state for a single UbuSTT user. Each client
// owns a dedicated backend session: audio frames received from the user are
// forwarded to the backend, and transcription results are translated into
// UbuSTT events and pushed to the user.
type Client struct {
	Connection *websocket.Conn

	sessionCfg backends.SessionConfig
	factory    backends.Factory
	backend    backends.Backend
	ctx        context.Context // root context for the session lifetime

	writeMu sync.Mutex // serializes websocket writes to the user

	closeOnce sync.Once

	mu                   sync.Mutex
	modelLoaded          bool // true if the backend has loaded a model
	session              *messages.SessionUpdateSession
	itemSeq              int    // monotonic counter used to mint item ids
	currentItemID        string // open (in-progress) transcription item, if any
	audioBufferFinalized bool   // true if the user has sent InputAudioBufferCommit
}

// NewClient creates a Client bound to a user websocket connection.
func NewClient(conn *websocket.Conn, sessionCfg backends.SessionConfig, factory backends.Factory) *Client {
	return &Client{Connection: conn, sessionCfg: sessionCfg, factory: factory}
}

// Start opens the backend session, waits until it is ready, and advertises the
// session configuration to the user via session.created.
func (c *Client) Start(ctx context.Context) error {
	c.ctx = ctx

	callbacks := backends.BackendCallbacks{
		OnDelta:         c.onDelta,
		OnCommit:        c.onCommit,
		OnModelLoaded:   c.onModelLoaded,
		OnModelUnloaded: c.onModelUnloaded,
	}

	backend, err := c.factory(ctx, callbacks)
	if err != nil {
		return fmt.Errorf("opening backend session: %w", err)
	}
	c.backend = backend

	if err := backend.WaitReady(ctx); err != nil {
		return fmt.Errorf("waiting for backend: %w", err)
	}

	rate := c.sessionCfg.SampleRate
	if rate <= 0 {
		rate = 16000
	}
	created := messages.NewSessionCreated(c.sessionCfg.Model, c.sessionCfg.Lang, rate)

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
		c.send(messages.NewError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"backend session not started",
		))
		return fmt.Errorf("backend session not started")
	}
	if !c.modelLoaded {
		c.send(messages.NewError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"no model loaded",
		))
		return fmt.Errorf("no model loaded")
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
		c.send(messages.NewError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"backend session not started",
		))
		return fmt.Errorf("backend session not started")
	}
	if !c.modelLoaded {
		c.send(messages.NewError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"no model loaded",
		))
		return fmt.Errorf("no model loaded")
	}
	c.mu.Lock()
	c.audioBufferFinalized = true
	c.mu.Unlock()
	go func() {
		if err := c.backend.Finalize(c.ctx); err != nil {
			log.Printf("[WARN]: finalizing backend: %v", err)
		}
	}()
	return nil
}

// onDelta is called by the backend whenever a new text fragment is available
// for the current partial segment.
func (c *Client) onDelta(delta string) {
	c.mu.Lock()
	itemID := c.currentOrNewItemLocked()
	c.mu.Unlock()
	if err := c.send(messages.NewTranscriptionDelta(itemID, 0, delta)); err != nil {
		log.Printf("[WARN]: sending transcription delta: %v", err)
	}
}

// onCommit is called by the backend when a segment is finalized. An empty text
// signals a partial reset (the backend revised the in-progress text).
func (c *Client) onCommit(text string) {
	c.mu.Lock()
	itemID := c.currentOrNewItemLocked()
	c.mu.Unlock()
	if err := c.send(messages.NewTranscriptionCompleted(itemID, 0, text)); err != nil {
		log.Printf("[WARN]: sending transcription completed: %v", err)
	}

	if c.audioBufferFinalized {
		log.Printf("Received onCommit after audio buffer finalized.\n")
		// At this point we could close the connection and consider the session done
		// but if we close the connection here, data is not flushed to the client and the
		// transcription completed message we just sent is lost along the way
		// so we let the client close the connection after receiving the transcription completed message
	}
}

// Invoked when the backend has loaded a model.
func (c *Client) onModelLoaded() {
	c.mu.Lock()
	c.modelLoaded = true
	c.mu.Unlock()
	// Send a new ModelLoaded message to the user
	err := c.send(new(messages.ModelLoaded))
	if err != nil {
		log.Printf("[WARN]: sending model loaded: %v", err)
	}
}

// Invoked when the backend has unloaded a model.
func (c *Client) onModelUnloaded() {
	c.mu.Lock()
	c.modelLoaded = false
	c.mu.Unlock()
	// Send a new ModelUnloaded message to the user
	err := c.send(new(messages.ModelUnloaded))
	if err != nil {
		log.Printf("[WARN]: sending model unloaded: %v", err)
	}
}

// currentOrNewItemLocked returns the id of the open transcription item, minting
// a new one if none is open. Callers must hold c.mu.
func (c *Client) currentOrNewItemLocked() string {
	if c.currentItemID == "" {
		c.itemSeq++
		c.currentItemID = fmt.Sprintf("item_%d", c.itemSeq)
	}
	return c.currentItemID
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
