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

	factory backends.Factory
	backend backends.Backend
	ctx     context.Context // root context for the session lifetime

	writeMu sync.Mutex // serializes websocket writes to the user

	closeOnce sync.Once

	mu                   sync.Mutex
	modelLoaded          bool // true if the backend has loaded a model
	sessionStarted       bool // true if the user has sent any audio chunks
	audioBufferFinalized bool // true if the user has sent InputAudioBufferCommit
}

// NewClient creates a Client bound to a user websocket connection.
func NewClient(conn *websocket.Conn, factory backends.Factory) *Client {
	return &Client{Connection: conn, factory: factory}
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

	created := messages.NewSessionCreated(c.backend.GetConfig())

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
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			fmt.Sprintf("decoding message: %s", err),
		)
	}

	switch m := msg.(type) {
	case *messages.SessionUpdate:
		return c.handleSessionUpdate(m)
	case *messages.InputAudioBufferAppend:
		return c.handleInputAudioBufferAppend(m)
	case *messages.InputAudioBufferCommit:
		return c.handleInputAudioBufferCommit(m)
	default:
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			fmt.Sprintf("unexpected message type: %T", m),
		)
	}
}

func getMergedConfiguration(original *backends.SessionConfig, update *messages.SessionUpdate) backends.SessionConfig {
	cfg := *original // Shallow copy

	if update.Session.Type != nil {
		cfg.Type = *update.Session.Type
	}
	if update.Session.Instructions != nil {
		cfg.Instructions = update.Session.Instructions
	}
	if update.Session.Prompt != nil {
		cfg.Prompt = update.Session.Prompt
	}
	if update.Session.Include != nil {
		cfg.Include = update.Session.Include
	}
	if update.Session.Audio != nil && update.Session.Audio.Input != nil {
		if update.Session.Audio.Input.Format != nil {
			cfg.SampleRate = update.Session.Audio.Input.Format.Rate
		}
		if update.Session.Audio.Input.Transcription != nil {
			if update.Session.Audio.Input.Transcription.Model != nil {
				cfg.Model = *update.Session.Audio.Input.Transcription.Model
			}
			if update.Session.Audio.Input.Transcription.Language != nil {
				cfg.Lang = *update.Session.Audio.Input.Transcription.Language
			}
		}
	}

	return cfg
}

///////////////func (c *Client) validateSessionUpdateAgainstBackend(m *messages.SessionUpdate) error {
///////////////	// Validate audio format supported by the backend
///////////////	if m.Session.Audio != nil && m.Session.Audio.Input != nil && m.Session.Audio.Input.Format != nil {
///////////////		backendSupport, err := c.backend.IsAudioFormatSupported(m.Session.Audio.Input.Format.Rate)
///////////////		if err != nil {
///////////////			return c.SendError(
///////////////				messages.ErrorTypeServer,
///////////////				messages.ErrorCodeServerError,
///////////////				fmt.Sprintf("checking audio format support: %s", err),
///////////////			)
///////////////		}
///////////////		if !backendSupport {
///////////////			return c.SendError(
///////////////				messages.ErrorTypeInvalidRequest,
///////////////				messages.ErrorCodeInvalidParameter,
///////////////				"unsupported audio format",
///////////////			)
///////////////		}
///////////////	}
///////////////
///////////////	// Validate model and language availability
///////////////	if m.Session.Audio != nil && m.Session.Audio.Input != nil && m.Session.Audio.Input.Transcription != nil {
///////////////		if m.Session.Audio.Input.Transcription.Model != nil {
///////////////			modelAvailable, err := c.backend.IsModelAvailable(*m.Session.Audio.Input.Transcription.Model)
///////////////			if err != nil {
///////////////				return c.SendError(
///////////////					messages.ErrorTypeServer,
///////////////					messages.ErrorCodeServerError,
///////////////					fmt.Sprintf("checking model availability: %s", err),
///////////////				)
///////////////			}
///////////////			if !modelAvailable {
///////////////				return c.SendError(
///////////////					messages.ErrorTypeInvalidRequest,
///////////////					messages.ErrorCodeInvalidParameter,
///////////////					"unsupported model",
///////////////				)
///////////////			}
///////////////		}
///////////////
///////////////		if m.Session.Audio.Input.Transcription.Language != nil {
///////////////			languageAvailable, err := c.backend.IsLanguageAvailable(*m.Session.Audio.Input.Transcription.Language)
///////////////			if err != nil {
///////////////				return c.SendError(
///////////////					messages.ErrorTypeServer,
///////////////					messages.ErrorCodeServerError,
///////////////					fmt.Sprintf("checking language support: %s", err),
///////////////				)
///////////////			}
///////////////			if !languageAvailable {
///////////////				return c.SendError(
///////////////					messages.ErrorTypeInvalidRequest,
///////////////					messages.ErrorCodeInvalidParameter,
///////////////					"unsupported language",
///////////////				)
///////////////			}
///////////////		}
///////////////	}
///////////////	return nil
///////////////}

// handleSessionUpdate merges the requested session patch and acknowledges it
// with a session.updated event.
func (c *Client) handleSessionUpdate(m *messages.SessionUpdate) error {
	// Validate request
	if err := m.Validate(); err != nil {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			fmt.Sprintf("invalid session update: %s", err),
		)
	}

	// Merge configurations
	mergedConfig := getMergedConfiguration(c.backend.GetConfig(), m)

	// Validate merged config against the backend
	ok, err := c.backend.ValidateSessionConfig(mergedConfig)
	if err != nil {
		return c.SendError(
			messages.ErrorTypeServer,
			messages.ErrorCodeServerError,
			fmt.Sprintf("validating session configuration: %s", err),
		)
	} else if !ok {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			fmt.Sprintf("invalid session configuration: %s", err),
		)
	}

	// Reject changes if currently transcribing
	if c.sessionStarted {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			"session updates are not allowed after audio has been sent",
		)
	}

	// Apply the merged configuration to the backend
	err = c.backend.SetConfig(mergedConfig)

	if err != nil {
		return c.SendError(
			messages.ErrorTypeServer,
			messages.ErrorCodeServerError,
			fmt.Sprintf("setting backend config: %s", err),
		)
	}

	return c.send(messages.NewSessionUpdated(c.backend.GetConfig()))
}

// Sends an error message on the user connection.
// The error is also returned to the caller for convenience.
func (c *Client) SendError(errorType string, errorCode string, message string) error {
	c.send(messages.NewError(errorType, errorCode, message))
	return fmt.Errorf("%s", message)
}

// handleInputAudioBufferAppend decodes a base64 PCM16 chunk and forwards it to
// the WhisperLive backend.
func (c *Client) handleInputAudioBufferAppend(m *messages.InputAudioBufferAppend) error {
	pcm, err := base64.StdEncoding.DecodeString(m.Audio)
	if err != nil {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeInvalidParameter,
			"audio is not valid base64",
		)
	}
	if len(pcm) == 0 {
		return nil
	}
	if c.audioBufferFinalized {
		return c.SendError(
			messages.ErrorTypeServer,
			messages.ErrorCodeServerError,
			"appending to a finalized audio buffer is not supported",
		)
	}
	if c.backend == nil {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"backend session not started",
		)
	}
	if !c.modelLoaded {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"no model loaded",
		)
	}

	c.mu.Lock()
	c.sessionStarted = true
	c.mu.Unlock()

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
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"backend session not started",
		)
	}
	if c.audioBufferFinalized {
		return c.SendError(
			messages.ErrorTypeServer,
			messages.ErrorCodeServerError,
			"committing a finalized audio buffer is not supported",
		)
	}
	if !c.modelLoaded {
		return c.SendError(
			messages.ErrorTypeInvalidRequest,
			messages.ErrorCodeNoModelError,
			"no model loaded",
		)
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
	if err := c.send(messages.NewTranscriptionDelta(delta)); err != nil {
		log.Printf("[WARN]: sending transcription delta: %v", err)
	}
}

// onCommit is called by the backend when a segment is finalized. An empty text
// signals a partial reset (the backend revised the in-progress text).
func (c *Client) onCommit(text string) {
	if err := c.send(messages.NewTranscriptionCompleted(text)); err != nil {
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
		Type:         &m.Session.Type,
		Instructions: m.Session.Instructions,
		Prompt:       m.Session.Prompt,
		Audio: &messages.SessionUpdateAudio{
			Input: &messages.SessionUpdateAudioInput{
				Format: &messages.SessionUpdateFormat{Rate: m.Session.Audio.Input.Format.Rate},
				Transcription: &messages.SessionUpdateTranscription{
					Model:    &m.Session.Audio.Input.Transcription.Model,
					Language: &m.Session.Audio.Input.Transcription.Language,
				},
			},
		},
		Include: m.Session.Include,
	}
}
