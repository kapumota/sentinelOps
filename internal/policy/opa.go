package policy

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"sentinelops/internal/config"
	"sentinelops/internal/security"
	"sentinelops/internal/telemetry"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ExternalRunner evalúa manifiestos con OPA desde binario local o sidecar HTTP.
type ExternalRunner interface {
	Check(input map[string]any) ([]string, []string, error)
}

type OPARunner struct {
	Binary    string
	PolicyDir string
}

type HTTPRunner struct {
	BaseURL      string
	HTTPClient   *http.Client
	CacheEnabled bool
	CacheTTL     time.Duration
	cache        map[string]cachedDecision
	mu           sync.Mutex
}

type cachedDecision struct {
	Denies    []string
	Warnings  []string
	ExpiresAt time.Time
}

func NewExternalRunner(cfg config.Config) ExternalRunner {
	if !cfg.PolicyEnabled {
		return nil
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.PolicyMode))
	switch mode {
	case "http", "sidecar":
		if strings.TrimSpace(cfg.PolicyURL) == "" {
			return nil
		}
		return NewHTTPRunner(HTTPRunnerOptions{
			BaseURL:      cfg.PolicyURL,
			Timeout:      cfg.PolicyTimeout,
			CacheEnabled: cfg.PolicyCacheEnabled,
			CacheTTL:     cfg.PolicyCacheTTL,
		})
	case "exec", "binary", "":
		if strings.TrimSpace(cfg.PolicyBinary) == "" || strings.TrimSpace(cfg.PolicyDir) == "" {
			return nil
		}
		return &OPARunner{
			Binary:    strings.TrimSpace(cfg.PolicyBinary),
			PolicyDir: strings.TrimSpace(cfg.PolicyDir),
		}
	default:
		return nil
	}
}

func (r *OPARunner) Check(input map[string]any) ([]string, []string, error) {
	if r == nil {
		return nil, nil, nil
	}

	inputFile, err := os.CreateTemp("", "sentinelops-policy-*.json")
	if err != nil {
		return nil, nil, fmt.Errorf("crear entrada temporal de política: %w", err)
	}
	defer os.Remove(inputFile.Name())

	if err := json.NewEncoder(inputFile).Encode(input); err != nil {
		_ = inputFile.Close()
		return nil, nil, fmt.Errorf("escribir entrada de política: %w", err)
	}
	if err := inputFile.Close(); err != nil {
		return nil, nil, fmt.Errorf("cerrar entrada de política: %w", err)
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
	binary, err := security.ValidateExecutable(r.Binary)
	if err != nil {
		return nil, err
	}
	policyDir, err := security.ValidateFilesystemPath(r.PolicyDir, "directorio de políticas OPA")
	if err != nil {
		return nil, err
	}
	safeInput, err := security.ValidateFilesystemPath(inputFile, "entrada temporal OPA")
	if err != nil {
		return nil, err
	}
	// #nosec G204 -- binary, policyDir y safeInput son validados antes de ejecutar sin shell.
	cmd := exec.Command(binary, "eval", "--format=json", "--data", policyDir, "--input", safeInput, query)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("opa eval falló para %s: %s", query, msg)
	}

	items, err := extractOPAStrings(stdout.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parsear salida de OPA para %s: %w", query, err)
	}
	return items, nil
}

type HTTPRunnerOptions struct {
	BaseURL      string
	Timeout      time.Duration
	CacheEnabled bool
	CacheTTL     time.Duration
}

func NewHTTPRunner(options HTTPRunnerOptions) *HTTPRunner {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	cacheTTL := options.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 30 * time.Second
	}

	return &HTTPRunner{
		BaseURL:      strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		HTTPClient:   &http.Client{Timeout: timeout},
		CacheEnabled: options.CacheEnabled,
		CacheTTL:     cacheTTL,
		cache:        map[string]cachedDecision{},
	}
}

func (r *HTTPRunner) Check(input map[string]any) ([]string, []string, error) {
	if r == nil || r.BaseURL == "" {
		return nil, nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), r.HTTPClient.Timeout)
	defer cancel()

	tracer := telemetry.Tracer()
	ctx, span := tracer.Start(ctx, "opa.sidecar.check",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("opa.mode", "http"),
			attribute.String("opa.url", r.BaseURL),
		),
	)
	defer span.End()

	cacheKey, err := buildCacheKey(input)
	if err != nil {
		return nil, nil, err
	}
	if r.CacheEnabled {
		if decision, ok := r.getCached(cacheKey); ok {
			span.SetAttributes(attribute.Bool("opa.cache_hit", true))
			return cloneStrings(decision.Denies), cloneStrings(decision.Warnings), nil
		}
	}

	denies, err := r.querySet(ctx, "kubernetes/security/deny", input)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, nil, err
	}
	warnings, err := r.querySet(ctx, "kubernetes/security/warn", input)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, nil, err
	}

	if r.CacheEnabled {
		r.setCached(cacheKey, cachedDecision{Denies: denies, Warnings: warnings, ExpiresAt: time.Now().Add(r.CacheTTL)})
	}
	span.SetAttributes(
		attribute.Bool("opa.cache_hit", false),
		attribute.Int("opa.denies", len(denies)),
		attribute.Int("opa.warnings", len(warnings)),
	)
	return denies, warnings, nil
}

func (r *HTTPRunner) querySet(ctx context.Context, path string, input map[string]any) ([]string, error) {
	payload := map[string]any{"input": input}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("serializar entrada OPA: %w", err)
	}

	url := fmt.Sprintf("%s/v1/data/%s", r.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	telemetry.InjectTracingHeaders(ctx, req.Header)

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("consultar OPA sidecar: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("leer respuesta OPA: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA sidecar respondió %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	items, err := extractOPAHTTPStrings(respBody)
	if err != nil {
		return nil, fmt.Errorf("parsear respuesta OPA sidecar para %s: %w", path, err)
	}
	return items, nil
}

func (r *HTTPRunner) getCached(key string) (cachedDecision, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	decision, ok := r.cache[key]
	if !ok {
		return cachedDecision{}, false
	}
	if time.Now().After(decision.ExpiresAt) {
		delete(r.cache, key)
		return cachedDecision{}, false
	}
	return cachedDecision{Denies: cloneStrings(decision.Denies), Warnings: cloneStrings(decision.Warnings), ExpiresAt: decision.ExpiresAt}, true
}

func (r *HTTPRunner) setCached(key string, decision cachedDecision) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[key] = cachedDecision{Denies: cloneStrings(decision.Denies), Warnings: cloneStrings(decision.Warnings), ExpiresAt: decision.ExpiresAt}
}

func buildCacheKey(input map[string]any) (string, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("serializar cache key de política: %w", err)
	}
	hash := sha256.Sum256(body)
	return hex.EncodeToString(hash[:]), nil
}

func extractOPAHTTPStrings(raw []byte) ([]string, error) {
	var payload struct {
		Result any `json:"result"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return normalizeOPAValue(payload.Result), nil
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
		sort.Strings(out)
		return out
	case map[string]any:
		out := make([]string, 0, len(v))
		for key := range v {
			out = append(out, key)
		}
		sort.Strings(out)
		return out
	default:
		return []string{fmt.Sprint(v)}
	}
}

func cloneStrings(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)
	return out
}
