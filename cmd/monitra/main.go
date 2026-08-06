package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"monitra/internal/foundation/config"
	foundationruntime "monitra/internal/foundation/runtime"
)

func main() {
	os.Exit(run())
}

func run() int {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
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
