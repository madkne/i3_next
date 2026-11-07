package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"i3_next/internal/adapters/sway"
	"i3_next/internal/api/events"
	"i3_next/internal/app/bootstrap"
	"i3_next/internal/app/processsvc"
	"i3_next/internal/app/supervisor"
	"i3_next/internal/core/config"
	"i3_next/internal/core/eventbus"
	"i3_next/internal/core/logging"
	"i3_next/internal/core/registry"
	"i3_next/internal/services/workspace"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// Load the service configuration (embedded defaults, config dir, env).
	merged, err := config.LoadMergedBytes("services.yaml", config.DefaultServices, config.LoadOptions{Env: true})
	if err != nil {
		logger.Error("failed to load services configuration", slog.Any("error", err))
		stop()
		os.Exit(1)
	}
	servicesCfg, err := config.ParseServices(merged)
	if err != nil {
		logger.Error("failed to parse services configuration", slog.Any("error", err))
		stop()
		os.Exit(1)
	}

	bus := eventbus.New()

	// Log ServiceStateChanged transitions with detail.
	cancelState := eventbus.SubscribeFunc[events.Event](bus, events.TopicServiceStateChanged, func(ev events.Event) {
		p := ev.Payload.(events.ServiceStateChanged)
		logger.Info("ServiceStateChanged",
			slog.String("service", p.Name),
			slog.String("state", p.State),
			slog.String("health", p.Health),
			slog.String("reason", p.Reason))
	})
	defer cancelState()

	// Log every other daemon event so workspace/output changes appear in the log.
	for _, topic := range []string{
		events.TopicSwayConnected,
		events.TopicSwayDisconnected,
		events.TopicWorkspaceChanged,
		events.TopicOutputChanged,
		events.TopicShutdownRequested,
	} {
		topic := topic
		cancel := eventbus.SubscribeFunc[events.Event](bus, topic, func(ev events.Event) {
			logger.Info("event",
				slog.String("type", string(ev.Type)),
				slog.String("source", ev.Source))
		})
		defer cancel()
	}

	reg := registry.New()

	// The sway adapter is built first: the workspace service depends on it.
	adapter := sway.NewReconnectingAdapter(logger)

	sup := supervisor.New(logger,
		supervisor.WithEnvAllowlist([]string{
			"PATH", "HOME", "USER", "LOGNAME", "SHELL", "LANG", "LC_*",
			"TERM", "DISPLAY", "WAYLAND_DISPLAY", "SWAYSOCK", "XDG_*",
			"XAUTHORITY", "TMPDIR", "TZ", "WLR_*",
		}))

	var appServices []any

	restartPolicy := supervisor.RestartPolicy{
		Enabled:        true,
		MaxRestarts:    3,
		Window:         time.Minute,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     5 * time.Second,
		Multiplier:     2,
	}

	// Register each enabled service in the daemon registry, honouring the
	// enabled/required/depends_on flags from services.yaml, then hand the
	// (unwrapped) service to bootstrap so its DependsOn() drives startup
	// ordering and its Optional() marker governs required-vs-optional handling.
	add := func(name string, entry config.ServiceEntry, svc registry.Service) {
		if !entry.Enabled {
			logger.Info("service disabled in services.yaml; skipping",
				slog.String("service", name))
			return
		}
		var class registry.Requirement = registry.Required
		if !entry.Required {
			class = registry.Optional
		}
		reg.MustRegister(svc, class, name)
		appServices = append(appServices, svc)
		logger.Info("registered service",
			slog.String("service", name),
			slog.String("requirement", string(class)),
			slog.Bool("required", entry.Required),
		)
	}

	for name, entry := range servicesCfg.Services {
		switch name {
		case "sway":
			add(name, entry, adapter)
		case "workspace":
			add(name, entry, workspace.New(adapter, bus))
		case "shell":
			spec := supervisor.ProcessSpec{Command: "quickshell", Restart: restartPolicy}
			add(name, entry, processsvc.New("shell", []string{"workspace"}, sup, spec, bus, logger))
		case "wallpaper":
			spec := supervisor.ProcessSpec{Command: "swaybg", Restart: restartPolicy}
			add(name, entry, processsvc.New("wallpaper", []string{"workspace"}, sup, spec, bus, logger))
		default:
			logger.Warn("unknown service in services.yaml; skipping", slog.String("service", name))
		}
	}

	opts := []bootstrap.Option{bootstrap.WithConfig(config.LoadOptions{Env: true})}
	for _, svc := range appServices {
		opts = append(opts, bootstrap.WithService(svc))
	}
	app, err := bootstrap.New(logger, opts...)
	if err != nil {
		logger.Error("failed to initialize application", slog.Any("error", err))
		stop()
		os.Exit(1)
	}

	if err := app.Run(ctx); err != nil {
		logger.Error("application stopped with error", slog.Any("error", err))
		stop()
		os.Exit(1)
	}

	stop()
	logger.Info("i3_next stopped successfully")
}
