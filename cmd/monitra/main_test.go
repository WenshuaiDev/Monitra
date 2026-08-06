package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRuntimeConfigCommandWritesDeploymentDocument(t *testing.T) {
	output := filepath.Join(t.TempDir(), "runtime-config.json")
	t.Setenv("MONITRA_RELEASE_IDENTITY", "2026.08.06-production")

	if exitCode := run([]string{"project-runtime-config", "--output", output}, testLogger()); exitCode != 0 {
		t.Fatalf("project command exit code = %d", exitCode)
	}

	document, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read projected runtime config: %v", err)
	}
	const expected = `{"expected_api_major":1,"expected_release_identity":"2026.08.06-production"}` + "\n"
	if string(document) != expected {
		t.Fatalf("runtime config = %q, want %q", document, expected)
	}
}

func TestProjectRuntimeConfigCommandRejectsMissingReleaseWithoutOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "runtime-config.json")
	t.Setenv("MONITRA_RELEASE_IDENTITY", "")

	if exitCode := run([]string{"project-runtime-config", "--output", output}, testLogger()); exitCode == 0 {
		t.Fatal("project command succeeded without a release identity")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("invalid projection left an output file: %v", err)
	}
}

func TestCheckReadinessCommandUsesPrivateManagementResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/readyz" {
			t.Fatalf("readiness path = %q", request.URL.Path)
		}
		response.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	if exitCode := run([]string{"check-readiness", "--url", server.URL + "/readyz"}, testLogger()); exitCode != 0 {
		t.Fatalf("readiness command exit code = %d", exitCode)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}
