package client

import (
	"fmt"
	"sync"

	"ubustt-proxy/ubustt/messages"

	"github.com/gorilla/websocket"
)

// Client holds the per-connection state: the websocket connection and any
// session/backend state that message handlers need. Behaviour for each message
// type lives here (next to the state it operates on), keeping the messages
// package a pure set of data types.
type Client struct {
	Connection *websocket.Conn

	mu      sync.Mutex
	session *messages.SessionUpdateSession
}

// NewClient creates a Client bound to a websocket connection.
func NewClient(conn *websocket.Conn) *Client {
	return &Client{Connection: conn}
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

func (c *Client) handleSessionUpdate(m *messages.SessionUpdate) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	session := m.Session
	c.session = &session
	return nil
}

func (c *Client) handleInputAudioBufferAppend(m *messages.InputAudioBufferAppend) error {
	// TODO: decode m.Audio and forward it to the transcription backend.
	return nil
}

func (c *Client) handleInputAudioBufferCommit(m *messages.InputAudioBufferCommit) error {
	// TODO: flush buffered audio to the backend and emit transcription results.
	return nil
}
