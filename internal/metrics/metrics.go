package metrics

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricServer struct {
	addr           string
	logger         *slog.Logger
	sessionsTotal  prometheus.Counter
	activeSessions prometheus.Gauge
	commandsTotal  *prometheus.CounterVec
	rejectedInputs prometheus.Counter
	sessionCloseBy *prometheus.CounterVec

	activeTunnels  *prometheus.GaugeVec
	tunnelEvents   *prometheus.CounterVec
	rejectedTunnel *prometheus.CounterVec
}

func New(addr string, logger *slog.Logger) *MetricServer {
	return &MetricServer{
		addr:           addr,
		logger:         logger,
		sessionsTotal:  promauto.NewCounter(prometheus.CounterOpts{Name: "sentinelops_sessions_total", Help: "Número total de sesiones interactivas aceptadas"}),
		activeSessions: promauto.NewGauge(prometheus.GaugeOpts{Name: "sentinelops_active_sessions", Help: "Número actual de sesiones interactivas activas"}),
		commandsTotal:  promauto.NewCounterVec(prometheus.CounterOpts{Name: "sentinelops_commands_total", Help: "Total de comandos ejecutados por nombre y resultado"}, []string{"command", "result"}),
		rejectedInputs: promauto.NewCounter(prometheus.CounterOpts{Name: "sentinelops_rejected_input_total", Help: "Total de entradas de usuario rechazadas"}),
		sessionCloseBy: promauto.NewCounterVec(prometheus.CounterOpts{Name: "sentinelops_session_close_total", Help: "Total de cierres de sesión por motivo"}, []string{"reason"}),
		activeTunnels:  promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "sentinelops_active_tunnels", Help: "Número actual de túneles activos por dirección"}, []string{"direction"}),
		tunnelEvents:   promauto.NewCounterVec(prometheus.CounterOpts{Name: "sentinelops_tunnel_events_total", Help: "Total de eventos del ciclo de vida de túneles por dirección y evento"}, []string{"direction", "event"}),
		rejectedTunnel: promauto.NewCounterVec(prometheus.CounterOpts{Name: "sentinelops_rejected_tunnels_total", Help: "Total de intentos de túnel rechazados por dirección y motivo"}, []string{"direction", "reason"}),
	}
}

func (m *MetricServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: m.addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	m.logger.Info("servidor de métricas escuchando", "addr", m.addr)
	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (m *MetricServer) ObserveSessionOpened() { m.sessionsTotal.Inc(); m.activeSessions.Inc() }
func (m *MetricServer) ObserveSessionClosed(reason string) {
	m.activeSessions.Dec()
	m.sessionCloseBy.WithLabelValues(reason).Inc()
}
func (m *MetricServer) ObserveCommand(name, result string) {
	m.commandsTotal.WithLabelValues(name, result).Inc()
}
func (m *MetricServer) ObserveRejectedInput() { m.rejectedInputs.Inc() }
func (m *MetricServer) ObserveTunnelOpened(direction string) {
	m.activeTunnels.WithLabelValues(direction).Inc()
	m.tunnelEvents.WithLabelValues(direction, "abierto").Inc()
}
func (m *MetricServer) ObserveTunnelClosed(direction string) {
	m.activeTunnels.WithLabelValues(direction).Dec()
	m.tunnelEvents.WithLabelValues(direction, "cerrado").Inc()
}
func (m *MetricServer) ObserveTunnelRejected(direction, reason string) {
	m.rejectedTunnel.WithLabelValues(direction, reason).Inc()
}
