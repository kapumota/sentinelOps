### Benchmarks de rendimiento

#### Objetivo

Este documento registra resultados reproducibles de la fase 10. Los benchmarks comparan conexiones TCP frente a conexiones SSH, validación Go nativa frente a Rust gRPC y overhead de evaluación OPA.

#### Comandos base

    make benchmarks
    make benchmark-network
    make benchmark-validator
    make benchmark-opa

#### Ubicación de reportes

Los resultados locales se generan en:

    reports/benchmarks/<timestamp>/

Cada ejecución contiene:

    metadata.txt
    go-benchmarks.txt

#### Criterios de lectura

Para conexiones TCP frente a SSH se revisa `conn/s` y asignaciones de memoria.

Para validación Go frente a Rust gRPC se revisa `ns/op`, `B/op` y `allocs/op`.

Para OPA se revisa la diferencia entre baseline, HTTP sin cache y HTTP con cache.

#### Resultados de referencia

Los resultados dependen de CPU, Docker, kernel, carga local y configuración de red. No se versionan reportes generados automáticamente.
