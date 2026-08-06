// Package postgresql owns creation and lifecycle of Monitra's relational
// database connection pool.
package postgresql

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitra/internal/foundation/config"
)

const (
	connectionRetryInterval = 100 * time.Millisecond
	connectionAttemptLimit  = 500 * time.Millisecond
)

var (
	ErrSecretUnavailable = errors.New("postgresql secret unavailable")
	ErrInvalidConfig     = errors.New("postgresql configuration invalid")
	ErrStartupDeadline   = errors.New("postgresql startup deadline exceeded")
)

// Pool is the single bounded relational database pool owned by Foundation.
// Its driver-specific implementation does not escape this package.
type Pool struct {
	pool *pgxpool.Pool
}

// Connect creates exactly one pool and returns it only after a real database
// round trip succeeds. It closes the pool before returning any failure.
func Connect(ctx context.Context, configuration config.PostgreSQL) (*Pool, error) {
	password, err := readPassword(configuration.PasswordFile)
	if err != nil {
		return nil, ErrSecretUnavailable
	}

	poolConfiguration, err := pgxpool.ParseConfig(connectionURL(configuration, password))
	password = ""
	if err != nil {
		return nil, ErrInvalidConfig
	}
	poolConfiguration.MaxConns = configuration.MaxConnections
	poolConfiguration.MinConns = 0
	poolConfiguration.ConnConfig.ConnectTimeout = connectionAttemptLimit

	driverPool, err := pgxpool.NewWithConfig(ctx, poolConfiguration)
	if err != nil {
		return nil, ErrInvalidConfig
	}
	pool := &Pool{pool: driverPool}
	if err := awaitConnection(ctx, driverPool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func (pool *Pool) Close() {
	pool.pool.Close()
}

// Available reports whether the existing pool can currently complete a
// database round trip. It never creates a replacement pool or retries a
// submitted business statement.
func (pool *Pool) Available(ctx context.Context) bool {
	return pool.pool.Ping(ctx) == nil
}

func awaitConnection(ctx context.Context, pool *pgxpool.Pool) error {
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, connectionAttemptLimit)
		err := pool.Ping(attemptCtx)
		cancel()
		if err == nil {
			return nil
		}

		timer := time.NewTimer(connectionRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ErrStartupDeadline
		case <-timer.C:
		}
	}
}

func readPassword(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	password := strings.TrimSuffix(string(contents), "\n")
	password = strings.TrimSuffix(password, "\r")
	if password == "" {
		return "", ErrSecretUnavailable
	}
	return password, nil
}

func connectionURL(configuration config.PostgreSQL, password string) string {
	address := &url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(configuration.User, password),
		Host:   net.JoinHostPort(configuration.Host, strconv.Itoa(int(configuration.Port))),
		Path:   configuration.Database,
	}
	query := address.Query()
	query.Set("sslmode", configuration.SSLMode)
	address.RawQuery = query.Encode()
	return address.String()
}
