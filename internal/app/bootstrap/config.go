package bootstrap

import (
	"context"
	"log/slog"
	"strings"

	"i3_next/internal/core/config"
)

// ConfigReport holds the result of loading configuration at startup.
type ConfigReport struct {
	Desktop   config.DesktopConfig
	Plugins   config.PluginsConfig
	Services  config.ServicesConfig
	Shortcuts config.ShortcutsConfig
	Degraded  bool
	Reason    string
	Source    string
}

// WithConfig loads all configuration files (desktop, plugins, services, shortcuts)
// with fallback/last-valid/defaults and registers a startup hook that applies
// degraded state and logs an actionable error if any config is invalid.
// It never causes startup to fail due to invalid user config.
func WithConfig(opts config.LoadOptions) Option {
	return func(a *App) {
		// Load all configs with fallback
		desktopReport, err := config.LoadDesktopWithFallback(opts)
		if err != nil {
			a.AddStartupHook(func(context.Context) error { return err })
			return
		}

		pluginsCfg, pluginsDegraded, pluginsIssues, err := config.LoadPluginsWithFallback(opts)
		if err != nil {
			a.AddStartupHook(func(context.Context) error { return err })
			return
		}

		servicesCfg, servicesDegraded, servicesIssues, err := config.LoadServicesWithFallback(opts)
		if err != nil {
			a.AddStartupHook(func(context.Context) error { return err })
			return
		}

		shortcutsCfg, shortcutsDegraded, shortcutsIssues, err := config.LoadShortcutsWithFallback(opts)
		if err != nil {
			a.AddStartupHook(func(context.Context) error { return err })
			return
		}

		allDegraded := desktopReport.Degraded || pluginsDegraded || servicesDegraded || shortcutsDegraded
		allIssues := desktopReport.Issues
		allIssues = append(allIssues, pluginsIssues...)
		allIssues = append(allIssues, servicesIssues...)
		allIssues = append(allIssues, shortcutsIssues...)

		// Determine source: if any is degraded, use "degraded", else "merged"
		source := "merged"
		if allDegraded {
			source = "degraded"
		}

		a.mu.Lock()
		a.configReport = &ConfigReport{
			Desktop:   desktopReport.Desktop,
			Plugins:   pluginsCfg,
			Services:  servicesCfg,
			Shortcuts: shortcutsCfg,
			Degraded:  allDegraded,
			Reason:    formatIssues(allIssues),
			Source:    source,
		}
		a.mu.Unlock()

		a.AddStartupHook(func(ctx context.Context) error {
			if allDegraded {
				a.setDegraded(true)
				if reason := formatIssues(allIssues); reason != "" {
					a.setLastFailure(configError(reason))
				}
				a.logger.Warn("configuration degraded; using fallback config",
					slog.String("source", source),
					slog.String("error", formatIssues(allIssues)),
				)
			} else {
				a.logger.Info("configuration loaded",
					slog.String("source", source),
				)
			}
			return nil
		})
	}
}

// formatIssues formats all load issues into a single actionable error string.
func formatIssues(issues []config.LoadIssue) string {
	parts := make([]string, 0, len(issues))
	for _, is := range issues {
		s := is.Error()
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " | ")
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
