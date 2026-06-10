### Fase 9.1: hardening de alertas de seguridad

#### Objetivo

Reducir alertas de Code scanning asociadas a gosec, CodeQL y Trivy antes de iniciar benchmarks de rendimiento.

#### Alcance

Esta fase se enfoca en alertas de código que no dependen solo de actualizar versiones:

- Rutas de archivo controladas por variables.
- Subprocesos lanzados con rutas o argumentos variables.
- Conversión segura de enteros hacia `uint32`.
- Uso explícito de modo no estricto para known hosts SSH.
- Uso de contexto de apagado sin `context.Background` directo.
- Redacción de secretos en logs.

#### Decisiones

Las alertas de dependencias productivas se corrigen con `go get` y `go mod tidy`.

Las alertas de dependencias marcadas como `Test` deben revisarse por separado. Si solo afectan `tests/integration/go.mod`, pueden documentarse como riesgo de entorno de pruebas o actualizarse en la misma fase si no rompe Testcontainers.

#### Validación

Comandos sugeridos:

    make check-secrets
    make vet
    make test
    make storage-test
    TESTCONTAINERS_RYUK_DISABLED=true make test-integration
    make rust-test
    make validator-grpc-build
    make validator-grpc-test
    git diff --check

Si las herramientas están instaladas localmente:

    gosec ./...
    trivy fs --scanners vuln,secret,misconfig .

#### Criterio de cierre

La fase se considera cerrada cuando:

- No quedan alertas gosec corregibles sin justificación.
- Las alertas de rutas y subprocesos tienen validación o justificación explícita.
- Las conversiones numéricas están acotadas antes de convertir.
- Los secretos no se registran en claro.
- Las alertas restantes de tests están documentadas o postergadas con criterio.
