// Package application owns the browser-facing HTTP listener for the core process.
package application

import (
	"log/slog"

	"monitra/internal/foundation/httpinterface"
	"monitra/internal/foundation/httpserver"
)

// Server is a running Application Listener.
type Server = httpserver.Server

// Start composes the browser-facing routes, binds the Application Listener,
// and supervises serving in a goroutine.
func Start(address, releaseIdentity string, logger *slog.Logger) (*Server, error) {
	handler := httpinterface.NewHandler(releaseIdentity, logger)
	server, err := httpserver.Start(address, handler)
	if err != nil {
		return nil, err
	}
	logger.Info("application listener started", "address", server.Address())
	return server, nil
}
