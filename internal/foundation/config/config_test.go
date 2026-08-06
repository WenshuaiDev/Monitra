package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"monitra/internal/foundation/config"
)

func TestLoadReturnsTypedStartupConfigurationWithSecretFileReference(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "postgres-password")
	if err := os.WriteFile(secretFile, []byte("not-logged\n"), 0o600); err != nil {
		t.Fatalf("write secret file: %v", err)
	}

	t.Setenv("MONITRA_RELEASE_IDENTITY", "2026.08.06-test")
	t.Setenv("MONITRA_MANAGEMENT_ADDRESS", "127.0.0.1:19090")
	t.Setenv("MONITRA_POSTGRES_HOST", "127.0.0.1")
	t.Setenv("MONITRA_POSTGRES_PORT", "55432")
	t.Setenv("MONITRA_POSTGRES_DATABASE", "monitra_test")
	t.Setenv("MONITRA_POSTGRES_USER", "monitra")
	t.Setenv("MONITRA_POSTGRES_PASSWORD_FILE", secretFile)
	t.Setenv("MONITRA_POSTGRES_SSL_MODE", "disable")
	t.Setenv("MONITRA_POSTGRES_MAX_CONNECTIONS", "7")
	t.Setenv("MONITRA_POSTGRES_STARTUP_TIMEOUT", "3s")

	got, err := config.Load()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	if got.ReleaseIdentity != "2026.08.06-test" {
		t.Fatalf("release identity = %q", got.ReleaseIdentity)
	}
	if got.ManagementAddress != "127.0.0.1:19090" {
		t.Fatalf("management address = %q", got.ManagementAddress)
	}
	if got.PostgreSQL.Host != "127.0.0.1" || got.PostgreSQL.Port != 55432 {
		t.Fatalf("postgres address = %s:%d", got.PostgreSQL.Host, got.PostgreSQL.Port)
	}
	if got.PostgreSQL.Database != "monitra_test" || got.PostgreSQL.User != "monitra" {
		t.Fatalf("postgres identity = %s/%s", got.PostgreSQL.Database, got.PostgreSQL.User)
	}
	if got.PostgreSQL.PasswordFile != secretFile {
		t.Fatalf("password file = %q", got.PostgreSQL.PasswordFile)
	}
	if got.PostgreSQL.SSLMode != "disable" {
		t.Fatalf("ssl mode = %q", got.PostgreSQL.SSLMode)
	}
	if got.PostgreSQL.MaxConnections != 7 {
		t.Fatalf("max connections = %d", got.PostgreSQL.MaxConnections)
	}
	if got.PostgreSQL.StartupTimeout != 3*time.Second {
		t.Fatalf("startup timeout = %s", got.PostgreSQL.StartupTimeout)
	}
}

func TestLoadRejectsUnreadablePostgreSQLSecretReference(t *testing.T) {
	t.Setenv("MONITRA_RELEASE_IDENTITY", "2026.08.06-test")
	t.Setenv("MONITRA_POSTGRES_PASSWORD_FILE", filepath.Join(t.TempDir(), "missing"))

	if _, err := config.Load(); err == nil {
		t.Fatal("load configuration succeeded with a missing secret file")
	}
}
