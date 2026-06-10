package benchmarks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"sentinelops/internal/policy"
)

func BenchmarkOPANoPolicyBaseline(b *testing.B) {
	input := samplePolicyInput()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if len(input) == 0 {
			b.Fatal("entrada vacía")
		}
	}
}

func BenchmarkOPAHTTPNoCache(b *testing.B) {
	server := newFakeOPAServer(b)
	defer server.Close()

	runner := policy.NewHTTPRunner(policy.HTTPRunnerOptions{
		BaseURL:      server.URL,
		Timeout:      2 * time.Second,
		CacheEnabled: false,
	})

	input := samplePolicyInput()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, _, err := runner.Check(input); err != nil {
			b.Fatalf("consulta OPA HTTP falló: %v", err)
		}
	}
}

func BenchmarkOPAHTTPCacheHit(b *testing.B) {
	server := newFakeOPAServer(b)
	defer server.Close()

	runner := policy.NewHTTPRunner(policy.HTTPRunnerOptions{
		BaseURL:      server.URL,
		Timeout:      2 * time.Second,
		CacheEnabled: true,
		CacheTTL:     time.Minute,
	})

	input := samplePolicyInput()
	if _, _, err := runner.Check(input); err != nil {
		b.Fatalf("precalentamiento OPA HTTP falló: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, _, err := runner.Check(input); err != nil {
			b.Fatalf("consulta OPA HTTP cacheada falló: %v", err)
		}
	}
}

func newFakeOPAServer(b *testing.B) *httptest.Server {
	b.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"result": []string{}}); err != nil {
			b.Fatalf("no se pudo escribir respuesta OPA falsa: %v", err)
		}
	}))
}

func samplePolicyInput() map[string]any {
	return map[string]any{
		"kind": "Pod",
		"metadata": map[string]any{
			"name":      "sentinelops-bench",
			"namespace": "default",
		},
		"spec": map[string]any{
			"containers": []any{
				map[string]any{
					"name":  "app",
					"image": "sentinelops:bench",
				},
			},
		},
	}
}
