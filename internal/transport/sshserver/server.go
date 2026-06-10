package sshserver

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"sentinelops/internal/auth"
	"sentinelops/internal/commands"
	"sentinelops/internal/config"
	"sentinelops/internal/crypto/authorizedkeys"
	"sentinelops/internal/forwarding"
	"sentinelops/internal/metrics"
	"sentinelops/internal/security"
	"sentinelops/internal/session"
	"sentinelops/internal/telemetry"
)

type Server struct {
	cfg             config.Config
	logger          *slog.Logger
	metrics         *metrics.MetricServer
	validator       security.InputValidator
	authenticator   auth.Authenticator
	rateLimiter     *auth.RateLimiter
	registry        *commands.Registry
	hostSigner      ssh.Signer
	authorizedKeys  *authorizedkeys.Store
	forwardPolicy   *forwarding.Policy
	tunnelManager   *forwarding.Manager
	sessionRegistry *session.Registry
	listener        net.Listener
}

type remoteForwardRegistry struct {
	mu        sync.Mutex
	listeners map[string]net.Listener
}

func newRemoteForwardRegistry() *remoteForwardRegistry {
	return &remoteForwardRegistry{listeners: make(map[string]net.Listener)}
}
func (r *remoteForwardRegistry) Put(bind string, l net.Listener) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.listeners[bind] = l
}
func (r *remoteForwardRegistry) Remove(bind string) net.Listener {
	r.mu.Lock()
	defer r.mu.Unlock()
	l := r.listeners[bind]
	delete(r.listeners, bind)
	return l
}
func (r *remoteForwardRegistry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, l := range r.listeners {
		_ = l.Close()
	}
	r.listeners = map[string]net.Listener{}
}

func New(
	cfg config.Config,
	logger *slog.Logger,
	metricServer *metrics.MetricServer,
	validator security.InputValidator,
	authenticator auth.Authenticator,
	rateLimiter *auth.RateLimiter,
	registry *commands.Registry,
	hostSigner ssh.Signer,
	authorizedKeys *authorizedkeys.Store,
	forwardPolicy *forwarding.Policy,
	tunnelManager *forwarding.Manager,
	sessionRegistry *session.Registry,
) *Server {
	if rateLimiter == nil {
		rateLimiter = auth.NewRateLimiter(auth.RateLimitConfig{Enabled: false})
	}
	return &Server{
		cfg:             cfg,
		logger:          logger,
		metrics:         metricServer,
		validator:       validator,
		authenticator:   authenticator,
		rateLimiter:     rateLimiter,
		registry:        registry,
		hostSigner:      hostSigner,
		authorizedKeys:  authorizedKeys,
		forwardPolicy:   forwardPolicy,
		tunnelManager:   tunnelManager,
		sessionRegistry: sessionRegistry,
	}
}

func (s *Server) Run(ctx context.Context) error {
	if !s.cfg.SSHPasswordAuthEnabled && !s.cfg.SSHPublicKeyAuthEnable {
		return fmt.Errorf("el servidor SSH no tiene métodos de autenticación habilitados")
	}
	ln, err := net.Listen("tcp", s.cfg.SSHListenAddr)
	if err != nil {
		return fmt.Errorf("falló la escucha SSH: %w", err)
	}
	s.listener = ln
	s.logger.Info(
		"servidor SSH escuchando",
		"addr", s.cfg.SSHListenAddr,
		"profile", s.cfg.Profile,
		"autenticacion_password", s.cfg.SSHPasswordAuthEnabled,
		"autenticacion_clave_publica", s.cfg.SSHPublicKeyAuthEnable,
		"reenvio_local_habilitado", s.cfg.SSHLocalForwardEnabled,
		"reenvio_remoto_habilitado", s.cfg.SSHRemoteForwardEnabled,
		"lista_permitidos_local", s.cfg.SSHForwardAllowlist,
		"lista_permitidos_bind_remoto", s.cfg.SSHRemoteBindAllowlist,
		"roles_permitidos_remoto", s.cfg.SSHRemoteAllowedRoles,
	)
	go func() {
		<-ctx.Done()
		_ = s.listener.Close()
	}()
	for {
		rawConn, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			s.logger.Error("falló la aceptación SSH", "error", err)
			continue
		}
		go s.handleConnection(ctx, rawConn)
	}
}

func (s *Server) handleConnection(ctx context.Context, rawConn net.Conn) {
	defer rawConn.Close()
	serverConfig := &ssh.ServerConfig{ServerVersion: s.cfg.SSHServerVersion}

	if s.cfg.SSHPasswordAuthEnabled {
		serverConfig.PasswordCallback = func(meta ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			_, authSpan := telemetry.StartAuthSpan(context.Background(), "password", meta.User(), meta.RemoteAddr().String())
			defer authSpan.End()
			rateKey := auth.Key(meta.RemoteAddr().String(), meta.User())
			if decision := s.rateLimiter.Allow(rateKey); !decision.Allowed {
				err := fmt.Errorf("login rate limit exceeded; retry after %s", roundDuration(decision.RetryAfter))
				telemetry.SetSpanError(authSpan, err)
				return nil, err
			}

			identity, err := s.authenticator.Authenticate(meta.User(), string(pass))
			if err != nil {
				decision := s.rateLimiter.RecordFailure(rateKey)
				if !decision.Allowed {
					err := fmt.Errorf("login rate limit exceeded; retry after %s", roundDuration(decision.RetryAfter))
					telemetry.SetSpanError(authSpan, err)
					return nil, err
				}
				telemetry.SetSpanError(authSpan, err)
				return nil, err
			}
			telemetry.SetSpanOK(authSpan, "autenticación correcta")
			s.rateLimiter.RecordSuccess(rateKey)
			return &ssh.Permissions{Extensions: map[string]string{"username": identity.Username, "role": string(identity.Role), "authn": "password"}}, nil
		}
	}

	if s.cfg.SSHPublicKeyAuthEnable {
		serverConfig.PublicKeyCallback = func(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			_, authSpan := telemetry.StartAuthSpan(context.Background(), "publickey", meta.User(), meta.RemoteAddr().String())
			defer authSpan.End()
			if s.authorizedKeys == nil {
				return nil, fmt.Errorf("el almacén de autenticación por clave pública no está configurado")
			}
			ok, err := s.authorizedKeys.IsAuthorized(meta.User(), key)
			if err != nil {
				return nil, err
			}
			if !ok {
				err := fmt.Errorf("la clave pública no está autorizada para el usuario %s", meta.User())
				telemetry.SetSpanError(authSpan, err)
				return nil, err
			}
			telemetry.SetSpanOK(authSpan, "autenticación correcta")
			return &ssh.Permissions{Extensions: map[string]string{"username": meta.User(), "role": defaultRoleFor(meta.User()), "authn": "publickey"}}, nil
		}
	}

	serverConfig.AddHostKey(s.hostSigner)
	conn, chans, reqs, err := ssh.NewServerConn(rawConn, serverConfig)
	if err != nil {
		s.logger.Warn("falló el handshake SSH", "remote_addr", rawConn.RemoteAddr().String(), "error", err)
		return
	}
	defer conn.Close()

	sess := session.New(conn.RemoteAddr().String())
	sess.SetTransport("ssh")
	username, role, authn := conn.User(), "desconocido", "desconocido"
	if conn.Permissions != nil {
		if v := conn.Permissions.Extensions["username"]; v != "" {
			username = v
		}
		if v := conn.Permissions.Extensions["role"]; v != "" {
			role = v
		}
		if v := conn.Permissions.Extensions["authn"]; v != "" {
			authn = v
		}
	}
	sess.SetIdentity(username, role)
	sess.SetAuthn(authn)
	ctx, sessionSpan := telemetry.StartSessionSpan(ctx, "ssh", sess.ID, conn.RemoteAddr().String())
	defer sessionSpan.End()
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
		s.logger.Info("sesión SSH cerrada", "session_id", sess.ID, "remote_addr", sess.RemoteAddr, "username", sess.Username, "role", sess.Role, "authn", authn, "comandos_ejecutados", sess.CommandCount, "motivo", closeReason)
	}()
	s.logger.Info("sesión SSH abierta", "session_id", sess.ID, "remote_addr", sess.RemoteAddr, "username", sess.Username, "role", sess.Role, "authn", authn)

	remoteRegistry := newRemoteForwardRegistry()
	defer remoteRegistry.CloseAll()
	go s.handleGlobalRequests(conn, reqs, sess, remoteRegistry)

	for newChannel := range chans {
		switch newChannel.ChannelType() {
		case "session":
			channel, requests, err := newChannel.Accept()
			if err != nil {
				s.logger.Error("falló la aceptación del canal de sesión SSH", "session_id", sess.ID, "error", err)
				continue
			}
			go func() {
				if motivo := s.handleSessionChannel(ctx, channel, requests, sess); motivo != "" {
					closeReason = motivo
				}
			}()
		case "direct-tcpip":
			go s.handleDirectTCPIP(ctx, newChannel, sess)
		default:
			_ = newChannel.Reject(ssh.UnknownChannelType, "tipo de canal no soportado")
		}
	}
}

func (s *Server) handleGlobalRequests(conn *ssh.ServerConn, reqs <-chan *ssh.Request, sess *session.Session, registry *remoteForwardRegistry) {
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			s.handleTCPIPForwardRequest(conn, req, sess, registry)
		case "cancel-tcpip-forward":
			s.handleCancelTCPIPForwardRequest(req, sess, registry)
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

func (s *Server) handleTCPIPForwardRequest(conn *ssh.ServerConn, req *ssh.Request, sess *session.Session, registry *remoteForwardRegistry) {
	type payload struct {
		BindAddr string
		BindPort uint32
	}
	var p payload
	if err := ssh.Unmarshal(req.Payload, &p); err != nil {
		if req.WantReply {
			_ = req.Reply(false, nil)
		}
		return
	}
	if s.forwardPolicy == nil || !s.forwardPolicy.AllowRemoteBind(sess.Role, p.BindAddr, p.BindPort) {
		if req.WantReply {
			_ = req.Reply(false, nil)
		}
		s.metrics.ObserveTunnelRejected("remote", "politica_denegada")
		s.logger.Warn("reenvío remoto denegado", "session_id", sess.ID, "username", sess.Username, "role", sess.Role, "bind", forwarding.NormalizeTarget(p.BindAddr, p.BindPort))
		return
	}
	bind := forwarding.NormalizeTarget(p.BindAddr, p.BindPort)
	listener, err := net.Listen("tcp", bind)
	if err != nil {
		if req.WantReply {
			_ = req.Reply(false, nil)
		}
		s.metrics.ObserveTunnelRejected("remote", "escucha_fallida")
		s.logger.Warn("falló la escucha del reenvío remoto", "session_id", sess.ID, "username", sess.Username, "bind", bind, "error", err)
		return
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		s.metrics.ObserveTunnelRejected("remote", "puerto_invalido")
		s.logger.Warn("dirección de escucha no TCP", "session_id", sess.ID, "username", sess.Username, "bind", bind)
		_ = listener.Close()
		return
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint32 {
		s.metrics.ObserveTunnelRejected("remote", "puerto_fuera_de_rango")
		s.logger.Warn("puerto de escucha fuera de rango", "session_id", sess.ID, "username", sess.Username, "bind", bind, "port", tcpAddr.Port)
		_ = listener.Close()
		return
	}
	actualPort := uint32(tcpAddr.Port)
	actualBind := forwarding.NormalizeTarget(p.BindAddr, actualPort)
	registry.Put(actualBind, listener)
	if req.WantReply {
		if p.BindPort == 0 {
			type response struct{ Port uint32 }
			_ = req.Reply(true, ssh.Marshal(response{Port: actualPort}))
		} else {
			_ = req.Reply(true, nil)
		}
	}
	s.logger.Info("reenvío remoto registrado", "session_id", sess.ID, "username", sess.Username, "bind", actualBind)
	go s.acceptRemoteForwardLoop(conn, sess, listener, p.BindAddr, actualPort, actualBind)
}

func (s *Server) handleCancelTCPIPForwardRequest(req *ssh.Request, sess *session.Session, registry *remoteForwardRegistry) {
	type payload struct {
		BindAddr string
		BindPort uint32
	}
	var p payload
	if err := ssh.Unmarshal(req.Payload, &p); err != nil {
		if req.WantReply {
			_ = req.Reply(false, nil)
		}
		return
	}
	bind := forwarding.NormalizeTarget(p.BindAddr, p.BindPort)
	l := registry.Remove(bind)
	if l != nil {
		_ = l.Close()
	}
	if req.WantReply {
		_ = req.Reply(true, nil)
	}
	s.logger.Info("reenvío remoto cancelado", "session_id", sess.ID, "username", sess.Username, "bind", bind)
}

func (s *Server) acceptRemoteForwardLoop(conn *ssh.ServerConn, sess *session.Session, listener net.Listener, bindAddr string, bindPort uint32, bind string) {
	for {
		incoming, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer incoming.Close()
			originHost, originPort := splitAddr(incoming.RemoteAddr().String())
			type payload struct {
				ConnectedAddress  string
				ConnectedPort     uint32
				OriginatorAddress string
				OriginatorPort    uint32
			}
			ch, reqs, err := conn.OpenChannel("forwarded-tcpip", ssh.Marshal(payload{
				ConnectedAddress:  bindAddr,
				ConnectedPort:     bindPort,
				OriginatorAddress: originHost,
				OriginatorPort:    originPort,
			}))
			if err != nil {
				s.metrics.ObserveTunnelRejected("remote", "apertura_canal_fallida")
				s.logger.Warn("falló la apertura del canal remoto reenviado", "session_id", sess.ID, "username", sess.Username, "bind", bind, "origin", net.JoinHostPort(originHost, strconv.Itoa(int(originPort))), "error", err)
				return
			}
			defer ch.Close()
			go ssh.DiscardRequests(reqs)

			stopOnce := &sync.Once{}
			stopFn := func() { stopOnce.Do(func() { _ = ch.Close(); _ = incoming.Close() }) }

			tunnelID := ""
			if s.tunnelManager != nil {
				t := s.tunnelManager.OpenRemote(sess.ID, sess.Username, bind, net.JoinHostPort(originHost, strconv.Itoa(int(originPort))), stopFn)
				tunnelID = t.ID
			}
			s.logger.Info("túnel remoto abierto", "tunnel_id", tunnelID, "session_id", sess.ID, "username", sess.Username, "bind", bind, "origin", net.JoinHostPort(originHost, strconv.Itoa(int(originPort))))

			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); _, _ = io.Copy(ch, incoming) }()
			go func() { defer wg.Done(); _, _ = io.Copy(incoming, ch) }()
			wg.Wait()
			stopFn()
			if tunnelID != "" {
				_ = s.tunnelManager.Close(tunnelID)
			}
			s.logger.Info("túnel remoto cerrado", "tunnel_id", tunnelID, "session_id", sess.ID, "username", sess.Username, "bind", bind, "origin", net.JoinHostPort(originHost, strconv.Itoa(int(originPort))))
		}()
	}
}

func (s *Server) handleDirectTCPIP(ctx context.Context, newChannel ssh.NewChannel, sess *session.Session) {
	if s.forwardPolicy == nil || !s.forwardPolicy.LocalEnabled() {
		_ = newChannel.Reject(ssh.Prohibited, "el reenvío local está deshabilitado")
		s.metrics.ObserveTunnelRejected("local", "disabled")
		return
	}
	var payload struct {
		HostToConnect  string
		PortToConnect  uint32
		OriginatorHost string
		OriginatorPort uint32
	}
	if err := ssh.Unmarshal(newChannel.ExtraData(), &payload); err != nil {
		_ = newChannel.Reject(ssh.ConnectionFailed, "payload direct-tcpip inválido")
		s.metrics.ObserveTunnelRejected("local", "bad_payload")
		return
	}
	if !s.forwardPolicy.AllowLocalTarget(sess.Role, payload.HostToConnect, payload.PortToConnect) {
		_ = newChannel.Reject(ssh.Prohibited, "el destino del reenvío no está permitido")
		s.metrics.ObserveTunnelRejected("local", "politica_denegada")
		return
	}
	target := net.JoinHostPort(payload.HostToConnect, strconv.Itoa(int(payload.PortToConnect)))
	origin := net.JoinHostPort(payload.OriginatorHost, strconv.Itoa(int(payload.OriginatorPort)))
	_, forwardSpan := telemetry.StartForwardingSpan(ctx, "local", origin, target, sess.ID)
	defer forwardSpan.End()
	upstream, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		telemetry.SetSpanError(forwardSpan, err)
		_ = newChannel.Reject(ssh.ConnectionFailed, "falló la conexión al destino")
		s.metrics.ObserveTunnelRejected("local", "dial_failed")
		s.logger.Warn("falló el marcado del reenvío", "session_id", sess.ID, "username", sess.Username, "target", target, "origin", origin, "error", err)
		return
	}
	channel, requests, err := newChannel.Accept()
	if err != nil {
		_ = upstream.Close()
		s.metrics.ObserveTunnelRejected("local", "channel_accept_failed")
		s.logger.Error("falló la aceptación del canal de reenvío", "session_id", sess.ID, "username", sess.Username, "target", target, "origin", origin, "error", err)
		return
	}
	go ssh.DiscardRequests(requests)

	stopOnce := &sync.Once{}
	stopFn := func() { stopOnce.Do(func() { _ = channel.Close(); _ = upstream.Close() }) }

	tunnelID := ""
	if s.tunnelManager != nil {
		t := s.tunnelManager.OpenLocal(sess.ID, sess.Username, target, origin, stopFn)
		tunnelID = t.ID
	}
	s.logger.Info("túnel local abierto", "tunnel_id", tunnelID, "session_id", sess.ID, "username", sess.Username, "target", target, "origin", origin)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); _, _ = io.Copy(channel, upstream) }()
	go func() { defer wg.Done(); _, _ = io.Copy(upstream, channel) }()
	wg.Wait()
	stopFn()
	if tunnelID != "" {
		_ = s.tunnelManager.Close(tunnelID)
	}
	telemetry.SetSpanOK(forwardSpan, "túnel local cerrado")
	s.logger.Info("túnel local cerrado", "tunnel_id", tunnelID, "session_id", sess.ID, "username", sess.Username, "target", target, "origin", origin)
}

func (s *Server) handleSessionChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, sess *session.Session) string {
	defer channel.Close()
	for req := range requests {
		switch req.Type {
		case "pty-req":
			_ = req.Reply(false, nil)
		case "shell":
			_ = req.Reply(true, nil)
			s.runShell(ctx, channel, sess)
			return "shell_cerrada"
		case "exec":
			var payload struct{ Value string }
			if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
				_ = req.Reply(false, nil)
				s.sendExitStatus(channel, 1)
				return "payload_exec_invalido"
			}
			_ = req.Reply(true, nil)
			status := s.runExec(ctx, channel, sess, payload.Value)
			s.sendExitStatus(channel, status)
			return "exec_completado"
		default:
			_ = req.Reply(false, nil)
		}
	}
	return "canal_cerrado"
}

func (s *Server) runShell(ctx context.Context, channel ssh.Channel, sess *session.Session) {
	_ = s.writeLine(channel, s.cfg.Banner)
	_ = s.writeLine(channel, "Transporte SSH habilitado. Escribe 'help' para ver los comandos.")
	reader := bufio.NewReader(channel)
	for {
		if _, err := io.WriteString(channel, "> "); err != nil {
			return
		}
		raw, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		input := strings.TrimSpace(raw)
		if input == "" {
			continue
		}
		output, status, closeSession := s.executeInput(ctx, sess, input)
		if output != "" {
			_ = s.writeLine(channel, output)
		}
		if status != 0 && closeSession {
			s.sendExitStatus(channel, status)
		}
		if closeSession {
			return
		}
	}
}

func (s *Server) runExec(ctx context.Context, channel ssh.Channel, sess *session.Session, input string) uint32 {
	output, status, _ := s.executeInput(ctx, sess, input)
	if output != "" {
		_ = s.writeLine(channel, output)
	}
	return status
}

func (s *Server) executeInput(ctx context.Context, sess *session.Session, input string) (string, uint32, bool) {
	_, validationSpan := telemetry.StartValidationSpan(ctx, "hybrid", len(input))
	if err := s.validator.Validate(input); err != nil {
		telemetry.SetSpanError(validationSpan, err)
		validationSpan.End()
		s.metrics.ObserveRejectedInput()
		s.metrics.ObserveCommand("validation", "rejected")
		return "Entrada rechazada: " + err.Error(), 1, false
	}
	telemetry.SetSpanOK(validationSpan, "entrada válida")
	validationSpan.End()
	name, args := splitCommand(input)
	if isQuit(name) {
		return "Sesión finalizada.", 0, true
	}
	cmd, ok := s.registry.Find(name)
	if !ok {
		s.metrics.ObserveCommand(name, "desconocido")
		return "Comando no reconocido: " + name, 127, false
	}
	sess.IncrementCommands()
	runtime := commands.Runtime{Session: sess, Config: s.cfg, Registry: s.registry, Forwarding: s.tunnelManager}
	commandCtx, commandSpan := telemetry.StartCommandSpan(ctx, cmd.Name(), sess.ID, sess.Username)
	output, err := cmd.Execute(commandCtx, runtime, args)
	if err != nil {
		telemetry.SetSpanError(commandSpan, err)
		commandSpan.End()
		s.metrics.ObserveCommand(cmd.Name(), "error")
		return "Error: " + err.Error(), 1, false
	}
	telemetry.SetSpanOK(commandSpan, "comando ejecutado")
	commandSpan.End()
	s.metrics.ObserveCommand(cmd.Name(), "ok")
	return output, 0, false
}

func (s *Server) sendExitStatus(channel ssh.Channel, status uint32) {
	type exitStatus struct{ Status uint32 }
	_, _ = channel.SendRequest("exit-status", false, ssh.Marshal(exitStatus{Status: status}))
}

func (s *Server) writeLine(w io.Writer, value string) error {
	normalized := strings.ReplaceAll(value, "\n", "\r\n")
	_, err := io.WriteString(w, normalized+"\r\n")
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

func defaultRoleFor(username string) string {
	switch strings.ToLower(strings.TrimSpace(username)) {
	case "student":
		return "student"
	case "teacher":
		return "teacher"
	case "auditor":
		return "auditor"
	case "admin":
		return "admin"
	default:
		return "student"
	}
}

func splitAddr(value string) (string, uint32) {
	host, rawPort, err := net.SplitHostPort(value)
	if err != nil {
		return value, 0
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < 0 || port > 65535 {
		return host, 0
	}
	return host, uint32(port)
}

func roundDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return d.Round(time.Second)
}
