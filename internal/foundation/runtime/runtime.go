// Package runtime composes and supervises the core process's Foundation-owned
// resources.
package runtime

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"monitra/internal/foundation/application"
	"monitra/internal/foundation/config"
	"monitra/internal/foundation/httpserver"
	"monitra/internal/foundation/management"
	"monitra/internal/foundation/postgresql"
)

const (
	cleanupTimeout          = 2 * time.Second
	dependencyProbeInterval = 100 * time.Millisecond
	dependencyProbeTimeout  = 500 * time.Millisecond
)

var (
	ErrApplicationListener = errors.New("application listener failed")
	ErrManagementListener  = errors.New("management listener failed")
	ErrDependencyStartup   = errors.New("required dependency failed during startup")
	errRuntimeCanceled     = errors.New("runtime canceled")
)

type connectionResult struct {
	pool *postgresql.Pool
	err  error
}

// coreRuntime owns the implementation of the core process lifecycle. Its
// state stays behind Run so callers only need the package's single runtime
// interface.
type coreRuntime struct {
	configuration     config.Configuration
	logger            *slog.Logger
	health            *management.State
	applicationServer *application.Server
	managementServer  *management.Server
	pool              *postgresql.Pool
}

// Run starts and supervises the Foundation runtime until ctx is cancelled or a
// required startup dependency fails.
func Run(ctx context.Context, configuration config.Configuration, logger *slog.Logger) error {
	core := newCoreRuntime(configuration, logger)
	if err := core.startManagement(); err != nil {
		return err
	}
	defer core.shutdown()

	if err := core.connectPostgreSQL(ctx); err != nil {
		if errors.Is(err, errRuntimeCanceled) {
			return nil
		}
		return err
	}
	if err := core.startApplication(); err != nil {
		return err
	}

	core.markReady()
	return core.supervise(ctx)
}

func (core *coreRuntime) startApplication() error {
	server, err := application.Start(
		core.configuration.ApplicationAddress,
		core.configuration.ReleaseIdentity,
		core.logger,
	)
	if err != nil {
		core.logger.Error("application listener failed", "reason", "bind_failed")
		return ErrApplicationListener
	}
	core.applicationServer = server
	return nil
}

func newCoreRuntime(configuration config.Configuration, logger *slog.Logger) *coreRuntime {
	return &coreRuntime{
		configuration: configuration,
		logger:        logger,
		health:        management.NewState(),
	}
}

func (core *coreRuntime) startManagement() error {
	server, err := management.Start(core.configuration.ManagementAddress, core.health, core.logger)
	if err != nil {
		core.logger.Error("management listener failed", "reason", "bind_failed")
		return ErrManagementListener
	}
	core.managementServer = server
	return nil
}

func (core *coreRuntime) connectPostgreSQL(ctx context.Context) error {
	configuration := core.configuration.PostgreSQL
	core.logger.Info(
		"waiting for required dependency",
		"dependency", "postgresql",
		"startup_timeout", configuration.StartupTimeout.String(),
		"max_connections", configuration.MaxConnections,
	)

	startupCtx, cancelStartup := context.WithTimeout(ctx, configuration.StartupTimeout)
	connection := make(chan connectionResult, 1)
	go func() {
		pool, connectErr := postgresql.Connect(startupCtx, configuration)
		connection <- connectionResult{pool: pool, err: connectErr}
	}()

	select {
	case result := <-connection:
		cancelStartup()
		if result.err != nil {
			if ctx.Err() != nil {
				return errRuntimeCanceled
			}
			core.logger.Error(
				"required dependency startup failed",
				"dependency", "postgresql",
				"reason", startupFailureReason(result.err),
			)
			return ErrDependencyStartup
		}
		core.pool = result.pool
		return nil
	case <-ctx.Done():
		cancelStartup()
		closeConnectionResult(<-connection)
		return errRuntimeCanceled
	case <-core.managementServer.Errors():
		cancelStartup()
		closeConnectionResult(<-connection)
		core.logger.Error("management listener failed", "reason", "serve_failed")
		return ErrManagementListener
	}
}

func (core *coreRuntime) markReady() {
	core.logger.Info(
		"postgresql connection pool created",
		"dependency", "postgresql",
		"max_connections", core.configuration.PostgreSQL.MaxConnections,
	)
	core.health.MarkReady()
	core.logger.Info(
		"core process ready",
		"dependency", "postgresql",
		"release_identity", core.configuration.ReleaseIdentity,
	)
}

func (core *coreRuntime) supervise(ctx context.Context) error {
	probeTicker := time.NewTicker(dependencyProbeInterval)
	defer probeTicker.Stop()
	dependencyAvailable := true

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-core.managementServer.Errors():
			core.logger.Error("management listener failed", "reason", "serve_failed")
			return ErrManagementListener
		case <-core.applicationServer.Errors():
			core.logger.Error("application listener failed", "reason", "serve_failed")
			return ErrApplicationListener
		case <-probeTicker.C:
			dependencyAvailable = core.probePostgreSQL(ctx, dependencyAvailable)
		}
	}
}

func (core *coreRuntime) probePostgreSQL(ctx context.Context, wasAvailable bool) bool {
	probeCtx, cancelProbe := context.WithTimeout(ctx, dependencyProbeTimeout)
	available := core.pool.Available(probeCtx)
	cancelProbe()
	if available == wasAvailable {
		return wasAvailable
	}
	if available {
		core.health.MarkReady()
		core.logger.Info(
			"required dependency restored",
			"dependency", "postgresql",
			"connection_pool", "existing",
		)
		return true
	}

	core.health.MarkNotReady()
	core.logger.Warn(
		"required dependency unavailable",
		"dependency", "postgresql",
		"connection_pool", "existing",
	)
	return false
}

func (core *coreRuntime) shutdown() {
	core.health.MarkNotReady()
	if core.applicationServer != nil {
		shutdownListener(core.applicationServer, "application", core.logger)
	}
	shutdownListener(core.managementServer, "management", core.logger)
	if core.pool != nil {
		core.pool.Close()
		core.logger.Info("core process shutdown complete")
	}
}

func shutdownListener(server *httpserver.Server, listener string, logger *slog.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error(listener+" listener cleanup failed", "reason", "shutdown_timeout")
	}
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
