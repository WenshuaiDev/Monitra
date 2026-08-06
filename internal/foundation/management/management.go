// Package management owns the private operational HTTP listener for the core process.
package management

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"

	"monitra/internal/foundation/httpserver"
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

// Server is a running Management Listener.
type Server = httpserver.Server

// Start binds the management listener before returning and supervises serving in
// a goroutine. The listener is live immediately; readiness is controlled by state.
func Start(address string, state *State, logger *slog.Logger) (*Server, error) {
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

	server, err := httpserver.Start(address, mux)
	if err != nil {
		return nil, err
	}
	logger.Info("management listener started", "address", server.Address())
	return server, nil
}

func writeStatus(response http.ResponseWriter, status int, value string) {
	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(struct {
		Status string `json:"status"`
	}{Status: value})
}
