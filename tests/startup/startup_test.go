//go:build integration

package startup_test

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

const testSecret = "startup-secret-never-log"

func TestCoreProcessBecomesReadyWithRealPostgreSQL(t *testing.T) {
	postgresAddress := startPostgreSQL(t)
	binary := buildCoreProcess(t)
	managementAddress := unusedAddress(t)
	process := startCoreProcess(t, binary, managementAddress, postgresAddress, "10s")

	client := &http.Client{Timeout: 100 * time.Millisecond}
	sawLive := false
	sawNotReady := false
	sawReady := false
	deadline := time.NewTimer(12 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

observe:
	for {
		select {
		case waitErr := <-process.waited:
			t.Fatalf("core process exited before becoming ready: %v\n%s", waitErr, process.output.String())
		case <-deadline.C:
			t.Fatalf("core process did not become ready with real PostgreSQL:\n%s", process.output.String())
		case <-ticker.C:
			sawLive = sawLive || statusCode(client, "http://"+managementAddress+"/livez") == http.StatusOK
			readyStatus := statusCode(client, "http://"+managementAddress+"/readyz")
			sawNotReady = sawNotReady || readyStatus == http.StatusServiceUnavailable
			if readyStatus == http.StatusOK {
				sawReady = true
				break observe
			}
		}
	}

	if !sawLive || !sawNotReady || !sawReady {
		t.Fatalf("startup health observations: live=%t not_ready=%t ready=%t; logs:\n%s", sawLive, sawNotReady, sawReady, process.output.String())
	}
	if err := process.command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal core process: %v", err)
	}
	select {
	case err := <-process.waited:
		if err != nil {
			t.Fatalf("core process did not shut down cleanly: %v\n%s", err, process.output.String())
		}
	case <-time.After(3 * time.Second):
		t.Fatal("core process did not complete bounded cleanup after SIGTERM")
	}

	if strings.Contains(process.output.String(), testSecret) {
		t.Fatalf("structured logs leaked the PostgreSQL secret: %s", process.output.String())
	}
	if !strings.Contains(process.output.String(), `"msg":"core process ready"`) || !strings.Contains(process.output.String(), `"dependency":"postgresql"`) {
		t.Fatalf("structured logs do not record successful PostgreSQL startup: %s", process.output.String())
	}
}

func TestCoreProcessExitsNonzeroWithoutEverBecomingReadyWhenPostgreSQLIsUnavailable(t *testing.T) {
	binary := buildCoreProcess(t)
	managementAddress := unusedAddress(t)
	postgresAddress := unusedAddress(t)
	process := startCoreProcess(t, binary, managementAddress, postgresAddress, "900ms")

	startedAt := time.Now()
	client := &http.Client{Timeout: 75 * time.Millisecond}
	sawLive := false
	sawNotReady := false
	sawReady := false
	var waitErr error

	deadline := time.NewTimer(4 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

observe:
	for {
		select {
		case waitErr = <-process.waited:
			break observe
		case <-deadline.C:
			t.Fatal("core process did not exit after its PostgreSQL startup deadline")
		case <-ticker.C:
			sawLive = sawLive || statusCode(client, "http://"+managementAddress+"/livez") == http.StatusOK
			readyStatus := statusCode(client, "http://"+managementAddress+"/readyz")
			sawNotReady = sawNotReady || readyStatus == http.StatusServiceUnavailable
			sawReady = sawReady || readyStatus == http.StatusOK
		}
	}

	var exitError *exec.ExitError
	if !errors.As(waitErr, &exitError) || exitError.ExitCode() == 0 {
		t.Fatalf("core process exit error = %v, want a nonzero exit", waitErr)
	}
	if elapsed := time.Since(startedAt); elapsed > 3*time.Second {
		t.Fatalf("core process exit took %s, want at most 3s", elapsed)
	}
	if !sawLive || !sawNotReady {
		t.Fatalf("startup health observations: live=%t not_ready=%t; logs:\n%s", sawLive, sawNotReady, process.output.String())
	}
	if sawReady {
		t.Fatalf("readiness succeeded while PostgreSQL was unavailable; logs:\n%s", process.output.String())
	}
	if strings.Contains(process.output.String(), testSecret) {
		t.Fatalf("structured logs leaked the PostgreSQL secret: %s", process.output.String())
	}
	if !strings.Contains(process.output.String(), `"dependency":"postgresql"`) || !strings.Contains(process.output.String(), `"reason":"startup_deadline_exceeded"`) {
		t.Fatalf("structured logs do not identify the bounded PostgreSQL startup failure: %s", process.output.String())
	}
}

type coreProcess struct {
	command *exec.Cmd
	output  bytes.Buffer
	waited  chan error
}

func startCoreProcess(t *testing.T, binary, managementAddress, postgresAddress, startupTimeout string) *coreProcess {
	t.Helper()
	postgresHost, postgresPort, err := net.SplitHostPort(postgresAddress)
	if err != nil {
		t.Fatalf("split PostgreSQL address: %v", err)
	}
	process := &coreProcess{
		command: exec.Command(binary),
		waited:  make(chan error, 1),
	}
	process.command.Env = processEnvironment(map[string]string{
		"MONITRA_RELEASE_IDENTITY":         "integration-test",
		"MONITRA_MANAGEMENT_ADDRESS":       managementAddress,
		"MONITRA_POSTGRES_HOST":            postgresHost,
		"MONITRA_POSTGRES_PORT":            postgresPort,
		"MONITRA_POSTGRES_DATABASE":        "monitra",
		"MONITRA_POSTGRES_USER":            "monitra",
		"MONITRA_POSTGRES_PASSWORD_FILE":   writeSecret(t, testSecret),
		"MONITRA_POSTGRES_SSL_MODE":        "disable",
		"MONITRA_POSTGRES_MAX_CONNECTIONS": "2",
		"MONITRA_POSTGRES_STARTUP_TIMEOUT": startupTimeout,
	})
	process.command.Stdout = &process.output
	process.command.Stderr = &process.output
	if err := process.command.Start(); err != nil {
		t.Fatalf("start core process: %v", err)
	}
	go func() { process.waited <- process.command.Wait() }()
	t.Cleanup(func() {
		if process.command.ProcessState == nil {
			_ = process.command.Process.Kill()
			<-process.waited
		}
	})
	return process
}

func buildCoreProcess(t *testing.T) string {
	t.Helper()
	repository := repositoryRoot(t)
	binary := filepath.Join(t.TempDir(), "monitra")
	command := exec.Command("go", "build", "-o", binary, "./cmd/monitra")
	command.Dir = repository
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build core process: %v\n%s", err, output)
	}
	return binary
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate integration test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func unusedAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve unused address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release unused address: %v", err)
	}
	return address
}

func writeSecret(t *testing.T, secret string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "postgres-password")
	if err := os.WriteFile(path, []byte(secret+"\n"), 0o600); err != nil {
		t.Fatalf("write PostgreSQL secret: %v", err)
	}
	return path
}

func startPostgreSQL(t *testing.T) string {
	t.Helper()
	command := exec.Command(
		"docker", "run", "--rm", "--detach",
		"--publish", "127.0.0.1::5432",
		"--env", "POSTGRES_PASSWORD="+testSecret,
		"--env", "POSTGRES_USER=monitra",
		"--env", "POSTGRES_DB=monitra",
		"postgres:18.4-alpine",
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("start real PostgreSQL container: %v\n%s", err, output)
	}
	containerID := strings.TrimSpace(string(output))
	if len(containerID) != 64 {
		t.Fatalf("docker returned an unexpected container ID %q", containerID)
	}
	t.Cleanup(func() {
		stop := exec.Command("docker", "stop", "--time", "2", containerID)
		if output, err := stop.CombinedOutput(); err != nil {
			t.Errorf("stop PostgreSQL container %s: %v\n%s", containerID[:12], err, output)
		}
	})

	port := exec.Command("docker", "port", containerID, "5432/tcp")
	portOutput, err := port.CombinedOutput()
	if err != nil {
		t.Fatalf("read PostgreSQL published port: %v\n%s", err, portOutput)
	}
	return strings.TrimSpace(string(portOutput))
}

func processEnvironment(values map[string]string) []string {
	environment := make([]string, 0, len(os.Environ())+len(values))
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "MONITRA_") {
			environment = append(environment, entry)
		}
	}
	for name, value := range values {
		environment = append(environment, fmt.Sprintf("%s=%s", name, value))
	}
	return environment
}

func statusCode(client *http.Client, url string) int {
	response, err := client.Get(url) //nolint:gosec // Integration test calls a loopback-only server.
	if err != nil {
		return 0
	}
	defer response.Body.Close()
	return response.StatusCode
}
