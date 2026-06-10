package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sentinelops/internal/config"
	"sentinelops/internal/forwarding"
	"sentinelops/internal/persistence"
	"sentinelops/internal/session"
	"sentinelops/internal/telemetry"
)

type Server struct {
	cfg        config.Config
	logger     *slog.Logger
	sesiones   *session.Registry
	tuneles    forwarding.Controller
	httpServer *http.Server
}

func New(cfg config.Config, logger *slog.Logger, sesiones *session.Registry, tuneles forwarding.Controller) *Server {
	return &Server{
		cfg:      cfg,
		logger:   logger,
		sesiones: sesiones,
		tuneles:  tuneles,
	}
}

func (s *Server) Start(ctx context.Context) error {
	if !s.cfg.ControlAPIEnabled {
		return nil
	}

	if err := ensureCertificatePair(s.cfg.ControlAPICertPath, s.cfg.ControlAPIKeyPath, parseHosts(s.cfg.ControlAPICertHosts)); err != nil {
		return fmt.Errorf("preparar certificado de la API de control: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/healthz/live", s.handleHealthz)
	mux.HandleFunc("/healthz/ready", s.handleReady)
	mux.HandleFunc("/healthz/startup", s.handleStartup)
	mux.HandleFunc("/api/v1/docs/swagger.json", s.handleOpenAPIJSON)
	mux.HandleFunc("/api/v1/docs/openapi.json", s.handleOpenAPIJSON)
	mux.HandleFunc("/api/v1/docs/swagger/", s.handleSwaggerUI)
	mux.HandleFunc("/api/v1/docs/swagger", s.handleSwaggerUI)
	mux.HandleFunc("/api/v1/admin/status", s.withAuth(s.handleStatus))
	mux.HandleFunc("/api/v1/admin/sessions", s.withAuth(s.handleSessions))
	mux.HandleFunc("/api/v1/admin/tunnels", s.withAuth(s.handleTunnels))
	mux.HandleFunc("/api/v1/admin/tunnels/", s.withAuth(s.handleV1TunnelByID))
	mux.HandleFunc("/api/admin/status", s.withAuth(s.handleStatus))
	mux.HandleFunc("/api/admin/sessions", s.withAuth(s.handleSessions))
	mux.HandleFunc("/api/admin/sesiones", s.withAuth(s.handleSessions))
	mux.HandleFunc("/api/admin/tunnels", s.withAuth(s.handleTunnels))
	mux.HandleFunc("/api/admin/tuneles", s.withAuth(s.handleTunnels))
	mux.HandleFunc("/api/admin/tunnels/", s.withAuth(s.handleTunnelByID))
	mux.HandleFunc("/api/admin/tuneles/", s.withAuth(s.handleTunnelByID))
	mux.HandleFunc("/api/admin/state/sessions", s.withAuth(s.handlePersistedSessions))
	mux.HandleFunc("/api/admin/state/tunnels", s.withAuth(s.handlePersistedTunnels))

	handler := telemetry.HTTPMiddleware("control_api.request", mux)

	s.httpServer = &http.Server{
		Addr:              s.cfg.ControlAPIAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       s.cfg.ReadTimeout,
		WriteTimeout:      s.cfg.WriteTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}()

	s.logger.Info(
		"API de control escuchando",
		"direccion", s.cfg.ControlAPIAddr,
		"cert", s.cfg.ControlAPICertPath,
		"usuario", s.cfg.ControlAPIUser,
	)

	err := s.httpServer.ListenAndServeTLS(s.cfg.ControlAPICertPath, s.cfg.ControlAPIKeyPath)
	if err == nil || err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		usuario, pass, ok := r.BasicAuth()
		if !ok || !constantTimeEqual(usuario, s.cfg.ControlAPIUser) || !constantTimeEqual(pass, s.cfg.ControlAPIPassword) {
			w.Header().Set("WWW-Authenticate", `Basic realm="sentinelops-control"`)
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": "no_autorizado",
			})
			return
		}
		next(w, r)
	}
}

func constantTimeEqual(actual, expected string) bool {
	if expected == "" {
		return false
	}
	if len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"servicio":     s.cfg.AppName,
		"profile":      s.cfg.Profile,
		"transporte":   s.cfg.Transport,
		"version_api":  apiVersion,
		"marca_tiempo": time.Now().UTC(),
	})
}

func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	componentes := map[string]any{
		"control_api":       s.cfg.ControlAPIEnabled,
		"sesiones":          s.sesiones != nil,
		"tuneles":           s.tuneles != nil,
		"opa_habilitado":    s.cfg.PolicyEnabled,
		"opa_modo":          s.cfg.PolicyMode,
		"persistencia":      s.cfg.StatePersistenceEnabled,
		"telemetria_trazas": s.cfg.TelemetryEnabled,
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ready",
		"servicio":     s.cfg.AppName,
		"version_api":  apiVersion,
		"componentes":  componentes,
		"marca_tiempo": time.Now().UTC(),
	})
}

func (s *Server) handleStartup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "started",
		"servicio":     s.cfg.AppName,
		"version_api":  apiVersion,
		"marca_tiempo": time.Now().UTC(),
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "ok",
		"servicio":         s.cfg.AppName,
		"profile":          s.cfg.Profile,
		"transporte":       s.cfg.Transport,
		"version_api":      apiVersion,
		"sesiones_activas": sessionCount(s.sesiones),
		"tuneles_activos":  tunelCount(s.tuneles),
		"control_api": map[string]any{
			"habilitado": true,
			"direccion":  s.cfg.ControlAPIAddr,
			"tls":        true,
		},
		"persistencia": map[string]any{
			"habilitada":       s.cfg.StatePersistenceEnabled,
			"archivo_sesiones": s.cfg.StateSessionsPath,
			"archivo_tuneles":  s.cfg.StateTunnelsPath,
		},
		"marca_tiempo": time.Now().UTC(),
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	items := []session.Snapshot{}
	if s.sesiones != nil {
		items = s.sesiones.Snapshot()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cantidad": len(items),
		"sesiones": items,
	})
}

func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	items := []forwarding.Tunnel{}
	if s.tuneles != nil {
		items = s.tuneles.Snapshot()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"cantidad": len(items),
		"tuneles":  items,
	})
}

func (s *Server) handlePersistedSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	if !s.cfg.StatePersistenceEnabled {
		writeJSON(w, http.StatusOK, map[string]any{"habilitada": false, "cantidad": 0, "sesiones": []session.Snapshot{}})
		return
	}
	snapshot, err := persistence.LoadSessions(s.cfg.StateSessionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]any{"habilitada": true, "cantidad": 0, "sesiones": []session.Snapshot{}})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "persistencia_sesiones_no_disponible"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"habilitada": true, "generado_en": snapshot.Generated, "cantidad": len(snapshot.Items), "sesiones": snapshot.Items})
}

func (s *Server) handlePersistedTunnels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	if !s.cfg.StatePersistenceEnabled {
		writeJSON(w, http.StatusOK, map[string]any{"habilitada": false, "cantidad": 0, "tuneles": []forwarding.Tunnel{}})
		return
	}
	snapshot, err := persistence.LoadTunnels(s.cfg.StateTunnelsPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, map[string]any{"habilitada": true, "cantidad": 0, "tuneles": []forwarding.Tunnel{}})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "persistencia_tuneles_no_disponible"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"habilitada": true, "generado_en": snapshot.Generated, "cantidad": len(snapshot.Items), "tuneles": snapshot.Items})
}

func (s *Server) handleTunnelByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/admin/tunnels/")
	id = strings.TrimPrefix(id, "/api/admin/tuneles/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "falta_id_tunel"})
		return
	}

	switch r.Method {
	case http.MethodDelete:
		if s.tuneles == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "reenvio_no_disponible"})
			return
		}
		tunel, ok := s.tuneles.Get(id)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "tunel_no_encontrado", "id": id})
			return
		}
		cerrado := s.tuneles.Close(id)
		writeJSON(w, http.StatusOK, map[string]any{
			"cerrado": cerrado,
			"tunel":   tunel,
		})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
	}
}

func (s *Server) handleV1TunnelByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/admin/tunnels/")
	path = strings.TrimSpace(path)
	id, action, hasAction := strings.Cut(path, "/")
	id = strings.TrimSpace(id)
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "falta_id_tunel"})
		return
	}

	if hasAction {
		if action != "close" {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "endpoint_no_encontrado"})
			return
		}
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
			return
		}
		s.closeTunnelByID(w, id)
		return
	}

	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "metodo_no_permitido"})
		return
	}
	s.closeTunnelByID(w, id)
}

func (s *Server) closeTunnelByID(w http.ResponseWriter, id string) {
	if s.tuneles == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "reenvio_no_disponible"})
		return
	}
	tunel, ok := s.tuneles.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "tunel_no_encontrado", "id": id})
		return
	}
	cerrado := s.tuneles.Close(id)
	writeJSON(w, http.StatusOK, map[string]any{
		"cerrado":      cerrado,
		"id":           id,
		"version_api":  apiVersion,
		"marca_tiempo": time.Now().UTC(),
		"tunel":        tunel,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

func sessionCount(reg *session.Registry) int {
	if reg == nil {
		return 0
	}
	return reg.Count()
}

func tunelCount(ctrl forwarding.Controller) int {
	if ctrl == nil {
		return 0
	}
	return ctrl.Count()
}

func parseHosts(value string) []string {
	out := make([]string, 0)
	for _, item := range strings.Split(value, ",") {
		v := strings.TrimSpace(item)
		if v != "" {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		out = append(out, "localhost", "127.0.0.1")
	}
	return out
}

func ensureCertificatePair(certPath, keyPath string, hosts []string) error {
	if certPath == "" || keyPath == "" {
		return fmt.Errorf("la ruta del certificado está vacía")
	}

	if err := os.MkdirAll(filepath.Dir(certPath), 0o700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return err
	}

	if fileExists(certPath) && fileExists(keyPath) {
		if _, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
			return nil
		}
	}

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return err
	}

	tpl := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName:   "SentinelOps Control API",
			Organization: []string{"SentinelOps Lab"},
		},
		NotBefore:             time.Now().UTC().Add(-1 * time.Hour),
		NotAfter:              time.Now().UTC().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tpl.IPAddresses = append(tpl.IPAddresses, ip)
		} else {
			tpl.DNSNames = append(tpl.DNSNames, h)
		}
	}

	der, err := x509.CreateCertificate(rand.Reader, tpl, tpl, &priv.PublicKey, priv)
	if err != nil {
		return err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	keyBytes := x509.MarshalPKCS1PrivateKey(priv)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})

	if err := os.WriteFile(certPath, certPEM, 0o600); err != nil {
		return err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return err
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
