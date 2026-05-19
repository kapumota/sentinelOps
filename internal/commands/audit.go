package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	auditpkg "sentinelops/internal/audit"
)

type AuditCommand struct {
	auditService *auditpkg.Service
}

func NewAuditCommand(auditService *auditpkg.Service) Command {
	return AuditCommand{auditService: auditService}
}

func (c AuditCommand) Name() string {
	return "audit"
}

func (c AuditCommand) Aliases() []string {
	return []string{"report"}
}

func (c AuditCommand) Description() string {
	return "Ejecuta una auditoría interna y externa del perfil activo"
}

func (c AuditCommand) Usage() string {
	return "audit [text|json]"
}

func (c AuditCommand) Execute(_ context.Context, rt Runtime, args []string) (string, error) {
	report := c.auditService.Run(rt.Config, rt.Session)

	if len(args) > 0 && strings.EqualFold(args[0], "json") {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "audit_status=%s\n", report.Status)
	fmt.Fprintf(&b, "profile=%s\n", report.Profile)
	fmt.Fprintf(&b, "session_id=%s\n", report.SessionID)
	fmt.Fprintf(&b, "username=%s\n", report.Username)
	fmt.Fprintf(&b, "total_findings=%d\n", report.TotalFindings)

	if len(report.Findings) == 0 {
		b.WriteString("findings=none")
		return b.String(), nil
	}

	b.WriteString("findings:\n")
	for _, finding := range report.Findings {
		fmt.Fprintf(
			&b,
			"- [%s] %s | %s | source=%s | %s | recommendation=%s\n",
			finding.Severity,
			finding.ID,
			finding.Category,
			finding.Source,
			finding.Message,
			finding.Recommendation,
		)
	}

	return strings.TrimRight(b.String(), "\n"), nil
}
