package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"monitra/internal/foundation/config"
	"monitra/internal/foundation/httpinterface"
	foundationruntime "monitra/internal/foundation/runtime"
	"monitra/internal/foundation/runtimeconfig"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	os.Exit(run(os.Args[1:], logger))
}

func run(arguments []string, logger *slog.Logger) int {
	if len(arguments) > 0 {
		switch arguments[0] {
		case "project-runtime-config":
			return runProjectRuntimeConfig(arguments[1:], logger)
		case "check-readiness":
			return runCheckReadiness(arguments[1:], logger)
		default:
			logger.Error("unsupported command")
			return 2
		}
	}
	return runCore(logger)
}

func runCore(logger *slog.Logger) int {
	configuration, err := config.Load()
	if err != nil {
		logger.Error("startup configuration rejected", "reason", err.Error())
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := foundationruntime.Run(ctx, configuration, logger); err != nil {
		logger.Error("core process exited", "reason", "runtime_failed")
		return 1
	}
	return 0
}

func runProjectRuntimeConfig(arguments []string, logger *slog.Logger) int {
	flags := flag.NewFlagSet("project-runtime-config", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	output := flags.String("output", "", "runtime config output path")
	if err := flags.Parse(arguments); err != nil || *output == "" || flags.NArg() != 0 {
		logger.Error("runtime config projection rejected", "reason", "arguments_invalid")
		return 2
	}

	configuration := runtimeconfig.Configuration{
		ExpectedAPIMajor:        httpinterface.APIMajor,
		ExpectedReleaseIdentity: os.Getenv("MONITRA_RELEASE_IDENTITY"),
	}
	if err := projectRuntimeConfig(*output, configuration); err != nil {
		logger.Error("runtime config projection rejected", "reason", "configuration_invalid")
		return 1
	}
	logger.Info("runtime config projected", "output", *output)
	return 0
}

func projectRuntimeConfig(outputPath string, configuration runtimeconfig.Configuration) error {
	directory := filepath.Dir(outputPath)
	temporary, err := os.CreateTemp(directory, ".runtime-config-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()

	if err := runtimeconfig.Project(temporary, configuration); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, outputPath)
}

func runCheckReadiness(arguments []string, logger *slog.Logger) int {
	flags := flag.NewFlagSet("check-readiness", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	url := flags.String("url", "http://127.0.0.1:9090/readyz", "private readiness URL")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return 2
	}

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(*url)
	if err != nil {
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		logger.Error("readiness response rejected", "reason", "body_unreadable")
		return 1
	}
	return 0
}
