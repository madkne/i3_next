package bootstrap

import (
	"context"
	"log/slog"

	"i3_next/internal/core/config"
)

// ConfigReport holds the result of loading configuration at startup.
type ConfigReport struct {
	Desktop  config.DesktopConfig
	Degraded bool
	Reason   string
	Source   string
}

// WithConfig loads desktop configuration (with fallback/last-valid/defaults)
// and registers a startup hook that applies degraded state and logs an
// actionable error if the config is invalid. It never causes startup to fail
// due to invalid user config.
func WithConfig(opts config.LoadOptions) Option {
	return func(a *App) {
		report, err := config.LoadDesktopWithFallback(opts)
		if err != nil {
			// Internal failure (e.g. embedded defaults broken): treat as fatal
			// by surfacing via startup hook.
			a.AddStartupHook(func(context.Context) error { return err })
			return
		}
		a.mu.Lock()
		a.configReport = &ConfigReport{
			Desktop:  report.Desktop,
			Degraded: report.Degraded,
			Reason:   report.ActionableError(),
			Source:   report.Source,
		}
		a.mu.Unlock()
		a.AddStartupHook(func(ctx context.Context) error {
			if report.Degraded {
				a.setDegraded(true)
				if report.ActionableError() != "" {
					a.setLastFailure(configError(report.ActionableError()))
				}
				a.logger.Warn("configuration degraded; using fallback config",
					slog.String("source", report.Source),
					slog.String("error", report.ActionableError()),
				)
			} else {
				a.logger.Info("configuration loaded",
					slog.String("source", report.Source),
				)
			}
			return nil
		})
	}
}

// configError wraps a config message as an error value for status reporting.
type configError string

func (e configError) Error() string { return string(e) }

// ConfigReport returns the loaded config report, if any.
func (a *App) ConfigReport() *ConfigReport {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.configReport == nil {
		return nil
	}
	cp := *a.configReport
	return &cp
}
