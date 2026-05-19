package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sentinelops/internal/config"
)

// ExternalRunner evaluates Kubernetes manifests with an external policy engine.
type ExternalRunner interface {
	Check(input map[string]any) ([]string, []string, error)
}

type OPARunner struct {
	Binary    string
	PolicyDir string
}

func NewExternalRunner(cfg config.Config) ExternalRunner {
	if !cfg.PolicyEnabled || strings.TrimSpace(cfg.PolicyBinary) == "" || strings.TrimSpace(cfg.PolicyDir) == "" {
		return nil
	}
	return &OPARunner{
		Binary:    strings.TrimSpace(cfg.PolicyBinary),
		PolicyDir: strings.TrimSpace(cfg.PolicyDir),
	}
}

func (r *OPARunner) Check(input map[string]any) ([]string, []string, error) {
	if r == nil {
		return nil, nil, nil
	}

	inputFile, err := os.CreateTemp("", "sentinelops-policy-*.json")
	if err != nil {
		return nil, nil, fmt.Errorf("create temporary policy input: %w", err)
	}
	defer os.Remove(inputFile.Name())

	if err := json.NewEncoder(inputFile).Encode(input); err != nil {
		_ = inputFile.Close()
		return nil, nil, fmt.Errorf("write policy input: %w", err)
	}
	if err := inputFile.Close(); err != nil {
		return nil, nil, fmt.Errorf("close policy input: %w", err)
	}

	denies, err := r.eval(inputFile.Name(), "data.kubernetes.security.deny")
	if err != nil {
		return nil, nil, err
	}
	warnings, err := r.eval(inputFile.Name(), "data.kubernetes.security.warn")
	if err != nil {
		return nil, nil, err
	}

	return denies, warnings, nil
}

func (r *OPARunner) eval(inputFile, query string) ([]string, error) {
	cmd := exec.Command(r.Binary, "eval", "--format=json", "--data", r.PolicyDir, "--input", inputFile, query)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("opa eval failed for %s: %s", query, msg)
	}

	items, err := extractOPAStrings(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parse opa output for %s: %w", query, err)
	}
	return items, nil
}

func extractOPAStrings(raw []byte) ([]string, error) {
	var payload struct {
		Result []struct {
			Expressions []struct {
				Value any `json:"value"`
			} `json:"expressions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if len(payload.Result) == 0 || len(payload.Result[0].Expressions) == 0 {
		return []string{}, nil
	}
	return normalizeOPAValue(payload.Result[0].Expressions[0].Value), nil
}

func normalizeOPAValue(value any) []string {
	switch v := value.(type) {
	case nil:
		return []string{}
	case string:
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	case map[string]any:
		out := make([]string, 0, len(v))
		for key := range v {
			out = append(out, key)
		}
		return out
	default:
		return []string{fmt.Sprint(v)}
	}
}
