package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	host string
	port int

	upgrader websocket.Upgrader
	httpSrv  *http.Server
	mu       sync.Mutex
	running  bool
}

func NewWebSocketServer(host string, port int) *WebSocketServer {
	return &WebSocketServer{
		host: host,
		port: port,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

func (s *WebSocketServer) Address() string {
	return s.host + ":" + strconv.Itoa(s.port)
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
		Addr:    s.Address(),
		Handler: mux,
	}
	s.running = true

	err := s.httpSrv.ListenAndServe()
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
