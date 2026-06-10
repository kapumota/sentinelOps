package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"sentinelops/internal/config"
	"sentinelops/internal/security"
)

// ExternalRunner executes an optional external audit tool and converts its
// JSON output into SentinelOps findings.
type ExternalRunner interface {
	Run(profile string) ([]Finding, error)
}

type CommandRunner struct {
	Command     string
	Script      string
	ProjectRoot string
}

func NewExternalRunner(cfg config.Config) ExternalRunner {
	if !cfg.ExternalAuditEnabled || strings.TrimSpace(cfg.ExternalAuditCommand) == "" || strings.TrimSpace(cfg.ExternalAuditScript) == "" {
		return nil
	}
	return &CommandRunner{
		Command:     strings.TrimSpace(cfg.ExternalAuditCommand),
		Script:      strings.TrimSpace(cfg.ExternalAuditScript),
		ProjectRoot: strings.TrimSpace(cfg.ProjectRoot),
	}
}

func (r *CommandRunner) Run(profile string) ([]Finding, error) {
	if r == nil {
		return nil, nil
	}
	projectRoot := r.ProjectRoot
	if projectRoot == "" {
		projectRoot = "."
	}

	command, err := security.ValidateExecutable(r.Command)
	if err != nil {
		return nil, err
	}
	script, err := security.ValidateFilesystemPath(r.Script, "script de auditoría externa")
	if err != nil {
		return nil, err
	}
	// #nosec G204 -- command y script son validados antes de ejecutar sin shell.
	cmd := exec.Command(command, script, "--profile", profile, "--project-root", projectRoot)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("external audit failed: %s", msg)
	}

	var payload struct {
		Findings []Finding `json:"findings"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("parse external audit output: %w", err)
	}

	return payload.Findings, nil
}
