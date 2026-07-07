// Package backends defines the interface that all transcription backends must
// implement, plus the factory type used to open per-connection sessions.
package backends

import "context"

// Backend is the interface a transcription backend must implement to be used by
// a ubustt client.
type Backend interface {
	// WaitReady blocks until the backend is ready to receive audio.
	WaitReady(ctx context.Context) error
	// Done returns a channel closed when the backend session has ended.
	Done() <-chan struct{}
	// Close tears down the backend session.
	Close() error
	// SendPCM16 forwards a chunk of little-endian signed 16-bit mono PCM audio.
	SendPCM16(pcm []byte) error
	// Finalize signals the end of audio input and waits for the backend to flush.
	Finalize(ctx context.Context) error
}

// Factory opens a new Backend session. The onDelta and onCommit callbacks are
// wired by the factory so the backend can emit transcription events without
// knowing about the UbuSTT protocol.
type Factory func(ctx context.Context, onDelta func(string), onCommit func(string)) (Backend, error)

// SessionConfig holds the session metadata advertised to the user via
// session.created.
type SessionConfig struct {
	Model      string
	Lang       string
	SampleRate int
}
