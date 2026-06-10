package benchmarks

import (
	"os"
	"testing"
	"time"

	"sentinelops/internal/security"
)

func BenchmarkValidatorGoNative(b *testing.B) {
	validator := security.NewDefaultValidator()
	input := "status"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := validator.Validate(input); err != nil {
			b.Fatalf("validación Go falló: %v", err)
		}
	}
}

func BenchmarkValidatorGoRejected(b *testing.B) {
	validator := security.NewDefaultValidator()
	input := "status && whoami"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := validator.Validate(input); err == nil {
			b.Fatal("se esperaba rechazo de entrada inválida")
		}
	}
}

func BenchmarkValidatorRustGRPC(b *testing.B) {
	addr := os.Getenv("BENCH_VALIDATOR_GRPC_ADDR")
	if addr == "" {
		b.Skip("defina BENCH_VALIDATOR_GRPC_ADDR para medir el validador Rust gRPC")
	}

	validator := security.NewValidator(security.Options{
		Mode:         "grpc",
		GRPCAddr:     addr,
		GRPCTimeout:  2 * time.Second,
		GRPCFailOpen: false,
	})
	input := "status"

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := validator.Validate(input); err != nil {
			b.Fatalf("validación Rust gRPC falló: %v", err)
		}
	}
}
