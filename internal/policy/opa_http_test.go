package policy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPRunnerCheckQueriesOPASidecar(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		if r.Method != http.MethodPost {
			t.Fatalf("método inesperado: %s", r.Method)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("payload inválido: %v", err)
		}
		if _, ok := payload["input"]; !ok {
			t.Fatal("payload sin input")
		}

		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/data/kubernetes/security/deny":
			_, _ = w.Write([]byte(`{"result":["denegado"]}`))
		case "/v1/data/kubernetes/security/warn":
			_, _ = w.Write([]byte(`{"result":["advertencia"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	runner := NewHTTPRunner(HTTPRunnerOptions{
		BaseURL:      server.URL,
		Timeout:      time.Second,
		CacheEnabled: false,
	})

	denies, warnings, err := runner.Check(BuildDeploymentInput("insecure"))
	if err != nil {
		t.Fatalf("Check devolvió error: %v", err)
	}
	if len(denies) != 1 || denies[0] != "denegado" {
		t.Fatalf("denegaciones inesperadas: %v", denies)
	}
	if len(warnings) != 1 || warnings[0] != "advertencia" {
		t.Fatalf("advertencias inesperadas: %v", warnings)
	}
	if requests["/v1/data/kubernetes/security/deny"] != 1 {
		t.Fatalf("consultas deny inesperadas: %d", requests["/v1/data/kubernetes/security/deny"])
	}
	if requests["/v1/data/kubernetes/security/warn"] != 1 {
		t.Fatalf("consultas warn inesperadas: %d", requests["/v1/data/kubernetes/security/warn"])
	}
}

func TestHTTPRunnerCheckUsesCache(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":[]}`))
	}))
	defer server.Close()

	runner := NewHTTPRunner(HTTPRunnerOptions{
		BaseURL:      server.URL,
		Timeout:      time.Second,
		CacheEnabled: true,
		CacheTTL:     time.Minute,
	})

	input := BuildDeploymentInput("hardened")
	if _, _, err := runner.Check(input); err != nil {
		t.Fatalf("primera consulta falló: %v", err)
	}
	if _, _, err := runner.Check(input); err != nil {
		t.Fatalf("segunda consulta falló: %v", err)
	}

	if calls != 2 {
		t.Fatalf("se esperaban 2 llamadas HTTP por la primera evaluación, got %d", calls)
	}
}

func TestNormalizeOPAValueSortsMapKeys(t *testing.T) {
	items := normalizeOPAValue(map[string]any{"b": true, "a": true})
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Fatalf("normalización inesperada: %v", items)
	}
}
