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

	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	network string
	address string

	upgrader websocket.Upgrader
	httpSrv  *http.Server
	mu       sync.Mutex
	running  bool
}

func NewWebSocketServer(host string, port int, unixSocketPath string) *WebSocketServer {
	network := "tcp"
	address := net.JoinHostPort(host, strconv.Itoa(port))

	if unixSocketPath != "" {
		network = "unix"
		address = unixSocketPath
	}

	return &WebSocketServer{
		network: network,
		address: address,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *WebSocketServer) Address() string {
	if s.network == "unix" {
		return "unix://" + s.address
	}
	return s.address
}

func (s *WebSocketServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.HandleWebSocket)
	mux.HandleFunc("/", s.handleHealth)

	s.httpSrv = &http.Server{
		Addr:    s.address,
		Handler: mux,
	}
	s.running = true

	if s.network == "unix" {
		if err := os.Remove(s.address); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.running = false
			s.httpSrv = nil
			return fmt.Errorf("removing existing unix socket %q: %w", s.address, err)
		}
	}

	listener, err := net.Listen(s.network, s.address)
	if err != nil {
		s.running = false
		s.httpSrv = nil
		return err
	}

	if s.network == "unix" {
		defer func() {
			_ = os.Remove(s.address)
		}()
	}
	defer listener.Close()

	err = s.httpSrv.Serve(listener)
	if err == http.ErrServerClosed {
		err = nil
	}

	s.running = false
	s.httpSrv = nil
	return err
}

func (s *WebSocketServer) Stop(ctx context.Context) error {
	if s.httpSrv == nil || !s.running {
		return nil
	}

	return s.httpSrv.Shutdown(ctx)
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer conn.Close()

	for {
		messageType, payload, err := conn.ReadMessage()
		if err != nil {
			return
		}

		if err := conn.WriteMessage(messageType, payload); err != nil {
			return
		}
	}
}

func (s *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
