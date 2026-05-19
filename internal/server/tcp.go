package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"

	"sentinelops/internal/auth"
	"sentinelops/internal/commands"
	"sentinelops/internal/config"
	"sentinelops/internal/metrics"
	"sentinelops/internal/security"
	"sentinelops/internal/session"
)

type TCPServer struct {
	cfg             config.Config
	logger          *slog.Logger
	metrics         *metrics.MetricServer
	validator       security.InputValidator
	authenticator   auth.Authenticator
	rateLimiter     *auth.RateLimiter
	registry        *commands.Registry
	sessionRegistry *session.Registry
	listener        net.Listener
}

func NewTCPServer(
	cfg config.Config,
	logger *slog.Logger,
	metricServer *metrics.MetricServer,
	validator security.InputValidator,
	authenticator auth.Authenticator,
	rateLimiter *auth.RateLimiter,
	registry *commands.Registry,
	sessionRegistry *session.Registry,
) *TCPServer {
	if rateLimiter == nil {
		rateLimiter = auth.NewRateLimiter(auth.RateLimitConfig{Enabled: false})
	}
	return &TCPServer{
		cfg:             cfg,
		logger:          logger,
		metrics:         metricServer,
		validator:       validator,
		authenticator:   authenticator,
		rateLimiter:     rateLimiter,
		registry:        registry,
		sessionRegistry: sessionRegistry,
	}
}

func (s *TCPServer) Run(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("falló la escucha: %w", err)
	}

	s.listener = ln
	s.logger.Info("servidor TCP escuchando", "addr", s.cfg.ListenAddr, "profile", s.cfg.Profile)

	go func() {
		<-ctx.Done()
		_ = s.listener.Close()
	}()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			s.logger.Error("falló la aceptación", "error", err)
			continue
		}
		go s.handleConnection(ctx, conn)
	}
}

func (s *TCPServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	sess := session.New(conn.RemoteAddr().String())
	sess.SetTransport("tcp")
	closeReason := "desconexion_cliente"

	s.metrics.ObserveSessionOpened()
	if s.sessionRegistry != nil {
		s.sessionRegistry.Add(sess)
	}
	defer func() {
		if s.sessionRegistry != nil {
			s.sessionRegistry.Remove(sess.ID)
		}
		s.metrics.ObserveSessionClosed(closeReason)
		s.logger.Info(
			"sesión cerrada",
			"session_id", sess.ID,
			"remote_addr", sess.RemoteAddr,
			"username", sess.Username,
			"role", sess.Role,
			"comandos_ejecutados", sess.CommandCount,
			"motivo", closeReason,
		)
	}()

	s.logger.Info(
		"sesión abierta",
		"session_id", sess.ID,
		"remote_addr", sess.RemoteAddr,
	)

	if err := s.writeLine(conn, s.cfg.Banner); err != nil {
		closeReason = "error_escritura"
		return
	}

	reader := bufio.NewReader(conn)

	if s.cfg.AuthEnabled {
		if err := s.authenticate(conn, reader, sess); err != nil {
			closeReason = "auth_failed"
			_ = s.writeLine(conn, "La autenticación falló. La sesión se cerró.")
			return
		}
	} else {
		sess.SetIdentity("anonymous", "guest")
		sess.SetAuthn("none")
		_ = s.writeLine(conn, "Autenticación deshabilitada. Se asignó una sesión de invitado.")
	}

	if err := s.writeLine(conn, "Escribe 'help' para ver los comandos."); err != nil {
		closeReason = "error_escritura"
		return
	}

	for {
		if err := s.writePrompt(conn, "> "); err != nil {
			closeReason = "error_escritura"
			return
		}

		input, err := s.readLine(conn, reader, s.cfg.IdleTimeout)
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				closeReason = "tiempo_inactivo"
				_ = s.writeLine(conn, "Sesión cerrada por inactividad.")
				return
			}
			if errors.Is(err, io.EOF) {
				closeReason = "desconexion_cliente"
				return
			}

			closeReason = "error_lectura"
			s.logger.Error("falló la lectura", "session_id", sess.ID, "error", err)
			return
		}

		if input == "" {
			continue
		}

		if err := s.validator.Validate(input); err != nil {
			s.metrics.ObserveRejectedInput()
			s.metrics.ObserveCommand("validation", "rejected")
			_ = s.writeLine(conn, "Entrada rechazada: "+err.Error())
			continue
		}

		name, args := splitCommand(input)
		if isQuit(name) {
			closeReason = "quit"
			_ = s.writeLine(conn, "Sesión finalizada.")
			return
		}

		cmd, ok := s.registry.Find(name)
		if !ok {
			s.metrics.ObserveCommand(name, "unknown")
			_ = s.writeLine(conn, "Comando no reconocido: "+name)
			continue
		}

		sess.IncrementCommands()

		runtime := commands.Runtime{
			Session:  sess,
			Config:   s.cfg,
			Registry: s.registry,
		}

		output, err := cmd.Execute(ctx, runtime, args)
		if err != nil {
			s.metrics.ObserveCommand(cmd.Name(), "error")
			_ = s.writeLine(conn, "Error: "+err.Error())
			continue
		}

		s.metrics.ObserveCommand(cmd.Name(), "ok")

		if output != "" {
			if err := s.writeLine(conn, output); err != nil {
				closeReason = "error_escritura"
				return
			}
		}
	}
}

func (s *TCPServer) authenticate(conn net.Conn, reader *bufio.Reader, sess *session.Session) error {
	maxAttempts := s.cfg.AuthMaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	_ = s.writeLine(conn, "Se requiere autenticación.")

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := s.writePrompt(conn, "username: "); err != nil {
			return err
		}
		username, err := s.readLine(conn, reader, s.cfg.ReadTimeout)
		if err != nil {
			return err
		}

		if err := s.writePrompt(conn, "password: "); err != nil {
			return err
		}
		password, err := s.readLine(conn, reader, s.cfg.ReadTimeout)
		if err != nil {
			return err
		}

		rateKey := auth.Key(conn.RemoteAddr().String(), username)
		if decision := s.rateLimiter.Allow(rateKey); !decision.Allowed {
			_ = s.writeLine(conn, fmt.Sprintf("Too many failed login attempts. Retry after %s.", roundDuration(decision.RetryAfter)))
			return errors.New("login rate limit exceeded")
		}

		identity, err := s.authenticator.Authenticate(username, password)
		if err == nil {
			s.rateLimiter.RecordSuccess(rateKey)
			sess.SetIdentity(identity.Username, string(identity.Role))
			sess.SetAuthn("password")
			_ = s.writeLine(conn, "Authentication successful.")
			return nil
		}

		decision := s.rateLimiter.RecordFailure(rateKey)
		if !decision.Allowed {
			_ = s.writeLine(conn, fmt.Sprintf("Invalid credentials (%d/%d). Login temporarily locked; retry after %s.", attempt, maxAttempts, roundDuration(decision.RetryAfter)))
			return errors.New("login rate limit exceeded")
		}

		_ = s.writeLine(conn, fmt.Sprintf("Invalid credentials (%d/%d).", attempt, maxAttempts))
	}

	return errors.New("maximum authentication attempts exceeded")
}

func (s *TCPServer) readLine(conn net.Conn, reader *bufio.Reader, timeout time.Duration) (string, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}

	raw, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(raw), nil
}

func (s *TCPServer) writeLine(conn net.Conn, value string) error {
	if err := conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout)); err != nil {
		return err
	}
	_, err := io.WriteString(conn, value+"\r\n")
	return err
}

func (s *TCPServer) writePrompt(conn net.Conn, value string) error {
	if err := conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteTimeout)); err != nil {
		return err
	}
	_, err := io.WriteString(conn, value)
	return err
}

func splitCommand(input string) (string, []string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil
	}
	if len(parts) == 1 {
		return strings.ToLower(parts[0]), nil
	}
	return strings.ToLower(parts[0]), parts[1:]
}

func isQuit(name string) bool {
	switch strings.ToLower(name) {
	case "quit", "exit", "q":
		return true
	default:
		return false
	}
}

func roundDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return d.Round(time.Second)
}
