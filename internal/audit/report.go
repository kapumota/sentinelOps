package audit

import (
	"fmt"
	"time"

	"sentinelops/internal/config"
	"sentinelops/internal/session"
)

type Finding struct {
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	Category       string `json:"category"`
	Source         string `json:"source"`
	Message        string `json:"message"`
	Recommendation string `json:"recommendation"`
}

type Report struct {
	Status        string    `json:"status"`
	Profile       string    `json:"profile"`
	SessionID     string    `json:"session_id"`
	Username      string    `json:"username"`
	Timestamp     time.Time `json:"timestamp"`
	TotalFindings int       `json:"total_findings"`
	Findings      []Finding `json:"findings"`
}

type Service struct {
	externalRunner ExternalRunner
}

func NewService(externalRunner ExternalRunner) *Service {
	return &Service{externalRunner: externalRunner}
}

func (s *Service) Run(cfg config.Config, sess *session.Session) Report {
	findings := baseFindings(cfg)

	if cfg.ExternalAuditEnabled && s.externalRunner != nil {
		externalFindings, err := s.externalRunner.Run(cfg.Profile)
		if err != nil {
			findings = append(findings, Finding{
				ID:             "AUD-EXT-001",
				Severity:       "low",
				Category:       "tooling",
				Source:         "internal-go",
				Message:        fmt.Sprintf("No se pudo ejecutar la auditoría externa: %v", err),
				Recommendation: "Verificar Python 3 y la ruta configurada en EXTERNAL_AUDIT_SCRIPT.",
			})
		} else {
			findings = append(findings, externalFindings...)
		}
	}

	status := "pass"
	if len(findings) > 0 {
		status = "fail"
	}

	return Report{
		Status:        status,
		Profile:       cfg.Profile,
		SessionID:     sess.ID,
		Username:      sess.Username,
		Timestamp:     time.Now().UTC(),
		TotalFindings: len(findings),
		Findings:      findings,
	}
}

func baseFindings(cfg config.Config) []Finding {
	findings := make([]Finding, 0)

	if cfg.Profile == "insecure" {
		findings = append(findings,
			Finding{
				ID:             "APP-001",
				Severity:       "high",
				Category:       "container",
				Source:         "internal-go",
				Message:        "El perfil insecure permite configuraciones débiles de contenedor.",
				Recommendation: "Usar el perfil hardened y aplicar securityContext restrictivo.",
			},
			Finding{
				ID:             "APP-002",
				Severity:       "critical",
				Category:       "runtime",
				Source:         "internal-go",
				Message:        "Se detecta posibilidad de escalamiento por configuración permisiva.",
				Recommendation: "Deshabilitar privilegios innecesarios y forzar allowPrivilegeEscalation=false.",
			},
			Finding{
				ID:             "APP-003",
				Severity:       "medium",
				Category:       "supply-chain",
				Source:         "internal-go",
				Message:        "El perfil inseguro está pensado para demostración y no para operación real.",
				Recommendation: "Versionar imágenes y evitar tags mutables.",
			},
		)
	}

	if !cfg.AuthEnabled {
		findings = append(findings, Finding{
			ID:             "APP-004",
			Severity:       "medium",
			Category:       "identity",
			Source:         "internal-go",
			Message:        "La autenticación interactiva está deshabilitada.",
			Recommendation: "Habilitar APP_AUTH_ENABLED=true en entornos de laboratorio controlado.",
		})
	}

	return findings
}
