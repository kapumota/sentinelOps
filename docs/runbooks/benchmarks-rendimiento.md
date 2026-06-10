### Runbook de benchmarks de rendimiento

#### Objetivo

Ejecutar benchmarks reproducibles para medir costos de red, validación y política en SentinelOps.

#### Preparación

Verifica que el árbol esté limpio antes de ejecutar mediciones comparables.

    git status --short
    make check-secrets
    make vet
    make test

#### Ejecutar todos los benchmarks

    make benchmarks

#### Ejecutar benchmarks de red

    make benchmark-network

Este bloque mide throughput de conexiones locales TCP y SSH mediante `conn/s`.

#### Ejecutar benchmarks de validación

    make benchmark-validator

El benchmark Go nativo siempre se ejecuta. El benchmark Rust gRPC se ejecuta solo si existe la variable:

    BENCH_VALIDATOR_GRPC_ADDR=127.0.0.1:50051

#### Ejecutar benchmarks OPA

    make benchmark-opa

Este bloque mide baseline sin política, OPA HTTP sin cache y OPA HTTP con cache usando un servidor HTTP local de prueba.

#### Variables útiles

    BENCH_PATTERN=BenchmarkValidator
    BENCH_TIME=5s
    BENCH_COUNT=5
    REPORT_DIR=reports/benchmarks/manual

#### Limpieza

    make benchmarks-clean

#### Interpretación

Compara siempre resultados entre ejecuciones hechas en la misma máquina y con carga similar. No mezcles resultados de laptop, servidor remoto y CI como si fueran equivalentes.
