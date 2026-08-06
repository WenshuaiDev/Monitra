// Package runtime composes and supervises the core process's Foundation-owned
// resources.
package runtime

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"monitra/internal/foundation/config"
	"monitra/internal/foundation/management"
	"monitra/internal/foundation/postgresql"
)

const cleanupTimeout = 2 * time.Second

var (
	ErrManagementListener = errors.New("management listener failed")
	ErrDependencyStartup  = errors.New("required dependency failed during startup")
)

type connectionResult struct {
	pool *postgresql.Pool
	err  error
}

// Run starts and supervises the Foundation runtime until ctx is cancelled or a
// required startup dependency fails.
func Run(ctx context.Context, configuration config.Configuration, logger *slog.Logger) error {
	health := management.NewState()
	managementServer, err := management.Start(configuration.ManagementAddress, health, logger)
	if err != nil {
		logger.Error("management listener failed", "reason", "bind_failed")
		return ErrManagementListener
	}

	logger.Info(
		"waiting for required dependency",
		"dependency", "postgresql",
		"startup_timeout", configuration.PostgreSQL.StartupTimeout.String(),
		"max_connections", configuration.PostgreSQL.MaxConnections,
	)

	startupCtx, cancelStartup := context.WithTimeout(ctx, configuration.PostgreSQL.StartupTimeout)
	connection := make(chan connectionResult, 1)
	go func() {
		pool, connectErr := postgresql.Connect(startupCtx, configuration.PostgreSQL)
		connection <- connectionResult{pool: pool, err: connectErr}
	}()

	var pool *postgresql.Pool
	select {
	case result := <-connection:
		cancelStartup()
		if result.err != nil {
			if ctx.Err() != nil {
				shutdownManagement(managementServer, logger)
				return nil
			}
			logger.Error(
				"required dependency startup failed",
				"dependency", "postgresql",
				"reason", startupFailureReason(result.err),
			)
			shutdownManagement(managementServer, logger)
			return ErrDependencyStartup
		}
		pool = result.pool
	case <-ctx.Done():
		cancelStartup()
		closeConnectionResult(<-connection)
		shutdownManagement(managementServer, logger)
		return nil
	case <-managementServer.Errors():
		cancelStartup()
		closeConnectionResult(<-connection)
		shutdownManagement(managementServer, logger)
		logger.Error("management listener failed", "reason", "serve_failed")
		return ErrManagementListener
	}

	health.MarkReady()
	logger.Info("core process ready", "dependency", "postgresql", "release_identity", configuration.ReleaseIdentity)

	var runErr error
	select {
	case <-ctx.Done():
	case <-managementServer.Errors():
		logger.Error("management listener failed", "reason", "serve_failed")
		runErr = ErrManagementListener
	}

	health.MarkNotReady()
	shutdownManagement(managementServer, logger)
	pool.Close()
	logger.Info("core process shutdown complete")
	return runErr
}

func startupFailureReason(err error) string {
	switch {
	case errors.Is(err, postgresql.ErrStartupDeadline):
		return "startup_deadline_exceeded"
	case errors.Is(err, postgresql.ErrSecretUnavailable):
		return "secret_unavailable"
	default:
		return "configuration_invalid"
	}
}

func closeConnectionResult(result connectionResult) {
	if result.pool != nil {
		result.pool.Close()
	}
}

func shutdownManagement(server *management.Server, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("management listener cleanup failed", "reason", "shutdown_timeout")
	}
}
