package bootstrap

import (
	"context"
	"log/slog"
)

type App struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) (*App, error) {
	if logger == nil {
		return nil, ErrLoggerRequired
	}

	return &App{
		logger: logger,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting i3_next desktop core",
		slog.String("version", "0.1.0"),
	)

	<-ctx.Done()

	a.logger.Info("shutting down i3_next")

	return nil
}
