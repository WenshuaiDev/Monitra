// Package httpserver owns the shared lifecycle policy for Foundation HTTP listeners.
package httpserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

const maxHeaderBytes = 1 << 20

// Server is a bound and supervised Foundation HTTP listener.
type Server struct {
	listener net.Listener
	server   *http.Server
	errors   chan error
}

// Start binds a listener with Foundation's bounded HTTP policy before returning.
func Start(address string, handler http.Handler) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	server := &Server{
		listener: listener,
		server: &http.Server{
			Handler:           handler,
			ReadHeaderTimeout: 2 * time.Second,
			ReadTimeout:       2 * time.Second,
			WriteTimeout:      2 * time.Second,
			IdleTimeout:       30 * time.Second,
			MaxHeaderBytes:    maxHeaderBytes,
		},
		errors: make(chan error, 1),
	}

	go func() {
		if err := server.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			server.errors <- err
		}
	}()
	return server, nil
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
