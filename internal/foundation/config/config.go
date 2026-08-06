package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultApplicationAddress = "127.0.0.1:8080"
	defaultManagementAddress  = "127.0.0.1:9090"
	defaultPostgreSQLHost     = "127.0.0.1"
	defaultPostgreSQLPort     = 5432
	defaultPostgreSQLDatabase = "monitra"
	defaultPostgreSQLUser     = "monitra"
	defaultPostgreSQLSSLMode  = "require"
	defaultMaxConnections     = 4
	defaultStartupTimeout     = 10 * time.Second
)

// Configuration contains the minimum typed inputs needed to start the core process.
type Configuration struct {
	ReleaseIdentity    string
	ApplicationAddress string
	ManagementAddress  string
	PostgreSQL         PostgreSQL
}

// PostgreSQL describes the single bounded connection pool owned by Foundation.
// PasswordFile is retained as a reference so secret contents do not enter general
// configuration logging or formatting.
type PostgreSQL struct {
	Host           string
	Port           uint16
	Database       string
	User           string
	PasswordFile   string
	SSLMode        string
	MaxConnections int32
	StartupTimeout time.Duration
}

// Load reads and validates production configuration from the process environment.
func Load() (Configuration, error) {
	releaseIdentity, err := required("MONITRA_RELEASE_IDENTITY")
	if err != nil {
		return Configuration{}, err
	}

	applicationAddress := valueOrDefault("MONITRA_APPLICATION_ADDRESS", defaultApplicationAddress)
	if _, _, err := net.SplitHostPort(applicationAddress); err != nil {
		return Configuration{}, errors.New("MONITRA_APPLICATION_ADDRESS must be a host:port address")
	}

	managementAddress := valueOrDefault("MONITRA_MANAGEMENT_ADDRESS", defaultManagementAddress)
	if _, _, err := net.SplitHostPort(managementAddress); err != nil {
		return Configuration{}, errors.New("MONITRA_MANAGEMENT_ADDRESS must be a host:port address")
	}

	port, err := integer("MONITRA_POSTGRES_PORT", defaultPostgreSQLPort, 1, 65535)
	if err != nil {
		return Configuration{}, err
	}
	maxConnections, err := integer("MONITRA_POSTGRES_MAX_CONNECTIONS", defaultMaxConnections, 1, 100)
	if err != nil {
		return Configuration{}, err
	}
	startupTimeout, err := duration("MONITRA_POSTGRES_STARTUP_TIMEOUT", defaultStartupTimeout)
	if err != nil {
		return Configuration{}, err
	}
	passwordFile, err := required("MONITRA_POSTGRES_PASSWORD_FILE")
	if err != nil {
		return Configuration{}, err
	}
	secret, err := os.ReadFile(passwordFile)
	if err != nil || strings.TrimSpace(string(secret)) == "" {
		return Configuration{}, errors.New("MONITRA_POSTGRES_PASSWORD_FILE must reference a readable non-empty file")
	}

	sslMode := valueOrDefault("MONITRA_POSTGRES_SSL_MODE", defaultPostgreSQLSSLMode)
	if !validSSLMode(sslMode) {
		return Configuration{}, errors.New("MONITRA_POSTGRES_SSL_MODE is unsupported")
	}

	return Configuration{
		ReleaseIdentity:    releaseIdentity,
		ApplicationAddress: applicationAddress,
		ManagementAddress:  managementAddress,
		PostgreSQL: PostgreSQL{
			Host:           valueOrDefault("MONITRA_POSTGRES_HOST", defaultPostgreSQLHost),
			Port:           uint16(port),
			Database:       valueOrDefault("MONITRA_POSTGRES_DATABASE", defaultPostgreSQLDatabase),
			User:           valueOrDefault("MONITRA_POSTGRES_USER", defaultPostgreSQLUser),
			PasswordFile:   passwordFile,
			SSLMode:        sslMode,
			MaxConnections: int32(maxConnections),
			StartupTimeout: startupTimeout,
		},
	}, nil
}

func required(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func integer(name string, fallback, minimum, maximum int) (int, error) {
	text := strings.TrimSpace(os.Getenv(name))
	if text == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("%s must be between %d and %d", name, minimum, maximum)
	}
	return value, nil
}

func duration(name string, fallback time.Duration) (time.Duration, error) {
	text := strings.TrimSpace(os.Getenv(name))
	if text == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(text)
	if err != nil || value <= 0 || value > 5*time.Minute {
		return 0, fmt.Errorf("%s must be greater than zero and no more than 5m", name)
	}
	return value, nil
}

func validSSLMode(value string) bool {
	switch value {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}
