package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"

	"myna-adapter/backends"
	"myna-adapter/openai/events"

	"github.com/gorilla/websocket"
)

// binding describes a single network/address pair the server listens on.
type binding struct {
	network string
	address string
}

func (b binding) displayAddress() string {
	if b.network == "unix" {
		return "unix://" + b.address
	}
	return b.address
}

type WebSocketServer struct {
	bindings []binding

	factory backends.Factory

	upgrader websocket.Upgrader
	httpSrv  *http.Server
	mu       sync.Mutex
	running  bool
}

// NewWebSocketServer creates a server that listens on TCP (host/port) and,
// if unixSocketPath is non-empty, additionally on a Unix domain socket, at
// the same time.
func NewWebSocketServer(host string, port int, unixSocketPath string) *WebSocketServer {
	bindings := []binding{
		{network: "tcp", address: net.JoinHostPort(host, strconv.Itoa(port))},
	}

	if unixSocketPath != "" {
		bindings = append(bindings, binding{network: "unix", address: unixSocketPath})
	}

	return &WebSocketServer{
		bindings: bindings,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// SetBackend configures the session metadata and the factory used to open a
// backend session for each connecting user.
func (s *WebSocketServer) SetBackend(cfg backends.SessionConfig, factory backends.Factory) {
	s.factory = factory
}

// Address returns the address of the first configured listener.
func (s *WebSocketServer) Address() string {
	return s.bindings[0].displayAddress()
}

// Addresses returns the display addresses of every listener the server binds to.
func (s *WebSocketServer) Addresses() []string {
	addrs := make([]string, len(s.bindings))
	for i, b := range s.bindings {
		addrs[i] = b.displayAddress()
	}
	return addrs
}

func (s *WebSocketServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/realtime", s.HandleWebSocket)
	mux.HandleFunc("/", s.handleHealth)

	s.httpSrv = &http.Server{Handler: mux}
	s.running = true
	s.mu.Unlock()

	listeners := make([]net.Listener, 0, len(s.bindings))
	cleanup := func() {
		for _, l := range listeners {
			l.Close()
		}
		s.mu.Lock()
		s.running = false
		s.httpSrv = nil
		s.mu.Unlock()
	}

	for _, b := range s.bindings {
		if b.network == "unix" {
			if err := os.Remove(b.address); err != nil && !errors.Is(err, os.ErrNotExist) {
				cleanup()
				return fmt.Errorf("removing existing unix socket %q: %w", b.address, err)
			}
		}

		listener, err := net.Listen(b.network, b.address)
		if err != nil {
			cleanup()
			return err
		}

		if b.network == "unix" {
			// set file permissions so unprivileged software can connect to the socket
			if err := os.Chmod(b.address, 0777); err != nil {
				listener.Close()
				cleanup()
				return fmt.Errorf("setting socket permissions: %w", err)
			}
		}

		listeners = append(listeners, listener)
	}

	errCh := make(chan error, len(listeners))
	for _, listener := range listeners {
		go func(l net.Listener) {
			err := s.httpSrv.Serve(l)
			if err == http.ErrServerClosed {
				err = nil
			}
			errCh <- err
		}(listener)
	}

	var firstErr error
	for range listeners {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
			// a listener failed on its own (not via Stop); tear down the
			// others so the server doesn't keep running in a degraded state.
			s.httpSrv.Close()
		}
	}

	for _, b := range s.bindings {
		if b.network == "unix" {
			os.Remove(b.address)
		}
	}

	s.mu.Lock()
	s.running = false
	s.httpSrv = nil
	s.mu.Unlock()

	return firstErr
}

func (s *WebSocketServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	srv := s.httpSrv
	running := s.running
	s.mu.Unlock()

	if srv == nil || !running {
		return nil
	}

	return srv.Shutdown(ctx)
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	session := NewSession(conn, s.factory)
	defer session.Close()

	// Open the backend session and advertise session.created before accepting
	// audio from the user.
	if err := session.Start(r.Context()); err != nil {
		fmt.Printf("starting client session: %v\n", err)
		_ = session.SendError(
			events.ErrorTypeServer,
			events.ErrorCodeServerError,
			"failed to start session",
		)
		return
	}

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("reading message: %v\n", err)
			return
		}

		switch messageType {
		case websocket.BinaryMessage:
			// Send error for unsupported binary frames
			_ = session.SendError(
				events.ErrorTypeInvalidRequest,
				events.ErrorCodeInvalidParameter,
				"binary messages are unsupported",
			)

		case websocket.TextMessage:
			if err := session.HandleMessage(payload); err != nil {
				fmt.Fprintf(os.Stderr, "Error handling message: %v\n", err)
			}
		}
	}
}

func (s *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
