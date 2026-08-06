package management_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"monitra/internal/foundation/management"
)

func TestManagementDistinguishesLiveFromReadyDuringStartup(t *testing.T) {
	state := management.NewState()
	server, err := management.Start("127.0.0.1:0", state, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("start management listener: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Errorf("shutdown management listener: %v", err)
		}
	})

	assertStatus(t, "http://"+server.Address()+"/livez", http.StatusOK, `{"status":"live"}`)
	assertStatus(t, "http://"+server.Address()+"/readyz", http.StatusServiceUnavailable, `{"status":"not_ready"}`)

	state.MarkReady()
	assertStatus(t, "http://"+server.Address()+"/readyz", http.StatusOK, `{"status":"ready"}`)
}

func assertStatus(t *testing.T, url string, wantStatus int, wantBody string) {
	t.Helper()
	response, err := http.Get(url) //nolint:gosec // Test calls its loopback-only server.
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s response: %v", url, err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("GET %s status = %d, want %d", url, response.StatusCode, wantStatus)
	}
	if string(body) != wantBody+"\n" {
		t.Fatalf("GET %s body = %q, want %q", url, body, wantBody+"\n")
	}
}
