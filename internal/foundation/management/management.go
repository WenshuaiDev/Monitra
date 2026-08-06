// Package management owns the private operational HTTP listener for the core process.
package management

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// State is the process readiness state exposed by the management listener.
type State struct {
	ready atomic.Bool
}

func NewState() *State {
	return &State{}
}

func (state *State) MarkReady() {
	state.ready.Store(true)
}

func (state *State) MarkNotReady() {
	state.ready.Store(false)
}

// Server is a running management listener.
type Server struct {
	listener net.Listener
	server   *http.Server
	errors   chan error
}

// Start binds the management listener before returning and supervises serving in
// a goroutine. The listener is live immediately; readiness is controlled by state.
func Start(address string, state *State, logger *slog.Logger) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(response http.ResponseWriter, _ *http.Request) {
		writeStatus(response, http.StatusOK, "live")
	})
	mux.HandleFunc("GET /readyz", func(response http.ResponseWriter, _ *http.Request) {
		if state.ready.Load() {
			writeStatus(response, http.StatusOK, "ready")
			return
		}
		writeStatus(response, http.StatusServiceUnavailable, "not_ready")
	})

	managementServer := &Server{
		listener: listener,
		server: &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 2 * time.Second,
			ReadTimeout:       2 * time.Second,
			WriteTimeout:      2 * time.Second,
			IdleTimeout:       30 * time.Second,
		},
		errors: make(chan error, 1),
	}

	logger.Info("management listener started", "address", listener.Addr().String())
	go func() {
		if err := managementServer.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			managementServer.errors <- err
		}
	}()

	return managementServer, nil
}

func (server *Server) Address() string {
	return server.listener.Addr().String()
}

func (server *Server) Errors() <-chan error {
	return server.errors
}

func (server *Server) Shutdown(ctx context.Context) error {
	return server.server.Shutdown(ctx)
}

func writeStatus(response http.ResponseWriter, status int, value string) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(struct {
		Status string `json:"status"`
	}{Status: value})
}
