package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"i3_next/internal/app/bootstrap"
	"i3_next/internal/core/logging"
	"i3_next/internal/version"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "Show version")
		debug       = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("i3_next %s (%s, %s)\n", version.Version, version.Commit, version.BuildDate)
		return
	}

	logger := logging.New(*debug)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	app, err := bootstrap.New(logger)
	if err != nil {
		logger.Error("failed to initialize application",
			slog.Any("error", err))
		stop()
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("application stopped with error",
			slog.Any("error", err))
		stop()
		os.Exit(1)
	}

	stop()
	logger.Info("i3_next stopped successfully")
}
