package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sentinelops/internal/audit"
	"sentinelops/internal/auth"
	"sentinelops/internal/commands"
	"sentinelops/internal/config"
	"sentinelops/internal/controlplane/httpapi"
	"sentinelops/internal/crypto/authorizedkeys"
	"sentinelops/internal/crypto/hostkeys"
	"sentinelops/internal/forwarding"
	"sentinelops/internal/metrics"
	"sentinelops/internal/observability"
	"sentinelops/internal/persistence"
	"sentinelops/internal/policy"
	"sentinelops/internal/security"
	"sentinelops/internal/server"
	"sentinelops/internal/session"
	"sentinelops/internal/telemetry"
	"sentinelops/internal/transport/sshserver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sentinelops:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	logger := observability.NewLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	telemetryProvider, err := telemetry.Init(ctx, telemetry.Config{
		Enabled:        cfg.TelemetryEnabled,
		ServiceName:    cfg.AppName,
		ServiceVersion: cfg.AppVersion,
		Environment:    cfg.Environment,
		Exporter:       cfg.TelemetryExporter,
		Endpoint:       cfg.TelemetryEndpoint,
		Insecure:       cfg.TelemetryInsecure,
		SampleRate:     cfg.TelemetrySampleRate,
	})
	if err != nil {
		logger.Warn("OpenTelemetry no se pudo inicializar", "error", err)
	} else if cfg.TelemetryEnabled {
		logger.Info("OpenTelemetry inicializado", "exportador", cfg.TelemetryExporter, "endpoint", cfg.TelemetryEndpoint)
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := telemetryProvider.Shutdown(shutdownCtx); err != nil {
				logger.Warn("falló el cierre de OpenTelemetry", "error", err)
			}
		}()
	}

	metricServer := metrics.New(cfg.MetricsAddr, logger)
	stateStore := persistence.NewStore(persistence.Options{
		Enabled:      cfg.StatePersistenceEnabled,
		SessionsPath: cfg.StateSessionsPath,
		TunnelsPath:  cfg.StateTunnelsPath,
	})
	sessionRegistry := session.NewRegistry()
	sessionRegistry.SetOnChange(func(items []session.Snapshot) {
		if err := stateStore.SaveSessions(items); err != nil {
			logger.Warn("falló la persistencia de sesiones", "error", err, "path", stateStore.SessionsPath())
		}
	})

	var tunnelManager *forwarding.Manager
	tunnelManager = forwarding.NewManager(forwarding.Hooks{
		OnOpen: func(t forwarding.Tunnel) {
			metricServer.ObserveTunnelOpened(t.Direction)
			if err := stateStore.SaveTunnels(tunnelManager.Snapshot()); err != nil {
				logger.Warn("falló la persistencia de túneles", "error", err, "path", stateStore.TunnelsPath())
			}
		},
		OnClose: func(t forwarding.Tunnel) {
			metricServer.ObserveTunnelClosed(t.Direction)
			if err := stateStore.SaveTunnels(tunnelManager.Snapshot()); err != nil {
				logger.Warn("falló la persistencia de túneles", "error", err, "path", stateStore.TunnelsPath())
			}
		},
	})
	_ = stateStore.SaveSessions(sessionRegistry.Snapshot())
	_ = stateStore.SaveTunnels(tunnelManager.Snapshot())

	rateLimiter := auth.NewRateLimiter(auth.RateLimitConfig{
		Enabled:     cfg.AuthRateLimitEnabled,
		MaxFailures: cfg.AuthRateLimitMaxFailures,
		Window:      cfg.AuthRateLimitWindow,
		Lockout:     cfg.AuthRateLimitLockout,
	})

	validator := security.NewValidator(security.Options{
		Mode:            cfg.ValidatorMode,
		ExternalEnabled: cfg.ExternalValidatorOn,
		ExternalBinary:  cfg.ExternalValidatorBin,
		FailOpen:        cfg.ExternalValidatorOpen,
		GRPCAddr:        cfg.ValidatorGRPCAddr,
		GRPCTimeout:     cfg.ValidatorGRPCTimeout,
		GRPCFailOpen:    cfg.ValidatorGRPCFailOpen,
	})
	authenticator := auth.NewDefaultService()
	auditService := audit.NewService(audit.NewExternalRunner(cfg))
	policyService := policy.NewService(policy.NewExternalRunner(cfg))
	commandRegistry := commands.NewRegistry(
		commands.NewHelpCommand(),
		commands.NewStatusCommand(),
		commands.NewWhoAmICommand(),
		commands.NewDateCommand(),
		commands.NewProfileCommand(),
		commands.NewAuditCommand(auditService),
		commands.NewPolicyCommand(policyService),
		commands.NewTunnelsCommand(),
	)

	errCh := make(chan error, 3)
	go startService(ctx, errCh, "metrics", logger, metricServer.Start)

	if cfg.ControlAPIEnabled {
		controlAPI := httpapi.New(cfg, logger, sessionRegistry, tunnelManager)
		go startService(ctx, errCh, "control-api", logger, controlAPI.Start)
	}

	go func() {
		errCh <- startTransport(ctx, cfg, logger, metricServer, validator, authenticator, rateLimiter, commandRegistry, sessionRegistry, tunnelManager)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if err != nil {
			stop()
			return err
		}
		return nil
	}
}

func startService(ctx context.Context, errCh chan<- error, name string, logger *slog.Logger, start func(context.Context) error) {
	if err := start(ctx); err != nil {
		errCh <- fmt.Errorf("%s: %w", name, err)
		return
	}
	logger.Info("servicio finalizado", "service", name)
}

func startTransport(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	metricServer *metrics.MetricServer,
	validator security.InputValidator,
	authenticator auth.Authenticator,
	rateLimiter *auth.RateLimiter,
	commandRegistry *commands.Registry,
	sessionRegistry *session.Registry,
	tunnelManager *forwarding.Manager,
) error {
	switch strings.ToLower(strings.TrimSpace(cfg.Transport)) {
	case "tcp", "":
		tcpServer := server.NewTCPServer(cfg, logger, metricServer, validator, authenticator, rateLimiter, commandRegistry, sessionRegistry)
		return tcpServer.Run(ctx)
	case "ssh":
		signer, err := hostkeys.LoadOrCreateSigner(cfg.SSHHostKeyPath)
		if err != nil {
			return fmt.Errorf("load ssh host key: %w", err)
		}
		keyStore := authorizedkeys.NewStore(cfg.SSHAuthorizedKeysDir)
		forwardPolicy := forwarding.NewPolicy(
			cfg.SSHLocalForwardEnabled,
			cfg.SSHForwardAllowlist,
			cfg.SSHLocalAllowedRoles,
			cfg.SSHRemoteForwardEnabled,
			cfg.SSHRemoteBindAllowlist,
			cfg.SSHRemoteAllowedRoles,
		)
		sshServer := sshserver.New(
			cfg,
			logger,
			metricServer,
			validator,
			authenticator,
			rateLimiter,
			commandRegistry,
			signer,
			keyStore,
			forwardPolicy,
			tunnelManager,
			sessionRegistry,
		)
		return sshServer.Run(ctx)
	default:
		return fmt.Errorf("transporte no soportado: %s", cfg.Transport)
	}
}
