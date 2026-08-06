package application_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"monitra/internal/foundation/application"
)

func TestApplicationListenerServesTheVersionedHandler(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	server, err := application.Start(
		"127.0.0.1:0",
		"2026.08.06-test",
		logger,
	)
	if err != nil {
		t.Fatalf("start application listener: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("shutdown application listener: %v", err)
		}
	})

	response, err := http.Get("http://" + server.Address() + "/api/v1/startup-handshake")
	if err != nil {
		t.Fatalf("call startup handshake: %v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read startup handshake: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.StatusCode, http.StatusOK, body)
	}
}
