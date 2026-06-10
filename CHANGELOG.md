### Fase 8: observabilidad operacional y evidencia runtime

#### Agregado

- Se agrega configuración operacional de Prometheus para runtime local.
- Se agrega Grafana con provisioning y dashboard `SentinelOps runtime`.
- Se agrega script `scripts/observability-smoke.sh`.
- Se agrega script `scripts/runtime-evidence.sh`.
- Se agregan runbooks y documentación de evidencia runtime.
- Se agregan targets `observability-up`, `observability-down`, `observability-logs`, `observability-smoke`, `runtime-evidence` y `observability-clean`.

### Changelog

### Fase 7: CI/CD DevSecOps

#### Agregado

- Se agrega workflow principal `ci-devsecops.yml`.
- Se agrega workflow `codeql.yml` para análisis estático Go.
- Se agrega workflow `release.yml` para releases por tag.
- Se agrega configuración de Dependabot.
- Se agrega `scripts/ci-check.sh` para validación local.
- Se agrega `.yamllint.yml`.
- Se agregan targets `ci-check`, `ci-openapi`, `ci-proto`, `ci-security`, `ci-clean` y `release-tag`.

#### Cambiado

- Se documenta el flujo de validación local y remota.
- Se documenta la limpieza de artefactos generados por proto, OPA, coverage y Rust.


### Unreleased - fase 5 OpenAPI y versionado de API

#### Agregado
- Endpoints versionados `/api/v1/admin/status`, `/api/v1/admin/sessions`, `/api/v1/admin/tunnels` y `/api/v1/admin/tunnels/{id}/close`.
- Health checks separados `/healthz/live`, `/healthz/ready` y `/healthz/startup` para probes de Kubernetes.
- Especificación OpenAPI 3.0 en `docs/openapi.json`, `docs/swagger.json` y `docs/openapi.yaml`.
- Documentación ligera en `/api/v1/docs/swagger/` y especificación JSON en `/api/v1/docs/swagger.json`.
- Target `make docs` para regenerar la documentación OpenAPI versionada.
- Target `make docs-check` para validar el JSON de la especificación.
- Target `make api-smoke` para probar health, documentación y estado administrativo de la API v1.
- Probes HTTP HTTPS en Helm usando el puerto `control-api`.

#### Mantenido
- Los endpoints legados `/api/admin/...` y `/healthz` siguen disponibles para compatibilidad.
- No se agregan dependencias externas de Swagger para mantener el build reproducible y liviano.

### Unreleased - fase 4 OPA sidecar runtime

#### Agregado
- Cliente OPA HTTP con cache TTL para consultar un sidecar en runtime.
- Configuración `OPA_POLICY_MODE=exec|http` para alternar entre binario local y sidecar HTTP.
- `docker-compose.opa.yml` para ejecutar SentinelOps con OPA sidecar.
- Targets `run-opa-sidecar`, `stop-opa-sidecar`, `opa-test`, `opa-build`, `opa-run` y `opa-ci`.
- Soporte Helm opcional para OPA sidecar en el mismo pod.
- Pruebas unitarias del cliente OPA HTTP y su cache.

#### Mantenido
- El modo `exec` sigue siendo el default para no exigir OPA como servicio externo en pruebas unitarias ni ejecución local básica.

### Unreleased - fase 2 tests de integración

#### Agregado
- Targets `test-unit`, `test-integration`, `test-race`, `test-coverage`, `test-all` y `test-e2e-containers`.
- Helpers compartidos para pruebas con `testcontainers-go`.
- Pruebas de integración para autenticación, rate limiting, sesiones, forwarding y métricas.
- Workflow `sentinelops-tests` con jobs de unit tests, integración, race detector y E2E manual.

#### Mejorado
- Las pruebas que requieren Docker usan el build tag `containers` para no afectar la suite rápida `make test`.
- El E2E de imagen completa queda parametrizado con `SENTINELOPS_E2E_IMAGE`.

### Unreleased - fase 1 secrets dinámicos

#### Corregido
- Se eliminaron credenciales de laboratorio hardcodeadas en código, scripts, Docker, Helm y documentación.
- Se agregó generación local de `.env.local` con contraseñas aleatorias mediante `make generate-secrets`.

#### Agregado
- `.env.example` como plantilla versionada sin secretos reales.
- Verificación `make check-secrets` para bloquear credenciales conocidas antes de integrar cambios.

### 2.4.1 - integración y liberación del estado

#### Mantenido
- Se documentan como placeholders las credenciales de laboratorio: `<LAB_PASSWORD_STUDENT>`, `<LAB_PASSWORD_TEACHER>`, `<LAB_PASSWORD_AUDITOR>` y `<LAB_PASSWORD_ADMIN>`.

#### Agregado
- Rate limiting configurable para login TCP y autenticación SSH por contraseña.
- Persistencia opcional de snapshots JSON para sesiones y túneles activos.
- Endpoints administrativos `/api/admin/state/sessions` y `/api/admin/state/tunnels`.
- Prueba de integración HTTPS con TLS real, Basic Auth y validación de cabeceras defensivas.
- Prueba de integración de forwarding SSH local `direct-tcpip` y remoto `tcpip-forward`.
- Prueba de integración del bloqueo temporal de login en transporte TCP.
- Pruebas unitarias del rate limiter y del store JSON de persistencia.

#### Mejorado
- Helm y archivos `env/*.env` exponen variables de rate limiting y persistencia.
- La API de control reporta el estado de persistencia en `/api/admin/status`.

### 2.4.0 - liberación corregida limpia

#### Corregido
- Se tomó `sentinelops_v2.3_main-clean-fixed` como base estable y se descartaron artefactos generados de Python.
- Se corrigió el default de `APP_SSH_REMOTE_FORWARD_ENABLED` para que el modo hardened no habilite reenvío remoto por omisión.
- Se corrigió el default de `EXTERNAL_VALIDATOR_FAIL_OPEN` para fallar cerrado por omisión.
- Se eliminó una doble escritura de resultado en `scripts/test-e2e-full.sh`.

#### Mejorado
- API de control con comparación constante para Basic Auth, timeouts HTTP completos, TLS mínimo 1.2 y cabeceras `nosniff`/`no-store`.
- Helm mueve secretos desde ConfigMap a Secret y permite `existingSecret`.
- NetworkPolicy de Helm expone también el puerto de la API de control cuando se instala el chart.
- CI verifica módulos Go con `go mod verify`, comprueba consistencia de `go.mod/go.sum` y fija OPA en `0.67.1`.
- Se agregaron pruebas unitarias para defaults de configuración y autenticación con variables de entorno.

#### Pendiente recomendado
- Reemplazar contraseñas de laboratorio por secretos generados o un backend de identidad si se usa fuera del entorno académico.
- Ejecutar CI completo en un runner con Go 1.25, Rust/Cargo, Docker, Helm y OPA disponibles.

### Fase 3 - OpenTelemetry y tracing distribuido

#### Agregado

- Se agrega paquete `internal/telemetry` para configurar OpenTelemetry.
- Se agrega middleware HTTP con correlation IDs y trace IDs.
- Se agregan spans para sesiones TCP, sesiones SSH, autenticación, validación, comandos y forwarding.
- Se agrega `docker-compose.observability.yml` con Jaeger, Prometheus y SentinelOps.
- Se agregan targets `run-jaeger`, `stop-jaeger`, `run-ssh-telemetry` y `docker-observability-up`.
- Se documenta el flujo de validación de trazas en README y VALIDACION.

#### Compatibilidad

- La telemetría queda deshabilitada por defecto con `OTEL_TRACES_ENABLED=false`.
- El flujo normal de `make test` no requiere Jaeger ni collector externo.

### Fase 6 - Validador Rust gRPC

#### Agregado

- Se agrega contrato `proto/validator/v1/validator.proto` para el servicio `validator.v1.Validator`.
- Se agrega crate `rust/input-guard-grpc` con servidor gRPC basado en `tonic`.
- Se agrega `docker-compose.grpc.yml` para ejecutar SentinelOps con `input-guard-grpc` como servicio lateral.
- Se agregan variables `VALIDATOR_MODE`, `VALIDATOR_GRPC_ADDR`, `VALIDATOR_GRPC_TIMEOUT` y `VALIDATOR_GRPC_FAIL_OPEN`.
- Se agregan targets `proto-go`, `proto-clean`, `validator-grpc-build`, `validator-grpc-test`, `validator-grpc-up`, `validator-grpc-down` y `validator-grpc-smoke`.
- Se agrega soporte Helm opcional para desplegar `input-guard-grpc` como sidecar.

#### Compatibilidad

- El modo por defecto sigue siendo `binary` para no romper el flujo existente.
- El validador por binario local se mantiene como fallback operativo.
### Fase 9: persistencia PostgreSQL y Redis

#### Agregado

- Se agrega interfaz `Store` para sesiones, túneles, auditoría y rate limiting.
- Se agrega implementación en memoria como modo por defecto de laboratorio.
- Se agrega implementación PostgreSQL para persistencia durable.
- Se agrega implementación Redis para cache, TTL y rate limiting.
- Se agrega `docker-compose.storage.yml`.
- Se agregan migraciones SQL y script de inicialización PostgreSQL.
- Se agregan comandos `storage-up`, `storage-down`, `storage-smoke`, `storage-test` y `storage-clean`.
- Se actualiza documentación de uso y validación.

### Fase 9.1: hardening de alertas de seguridad

#### Agregado

- Se agrega helper de validación de rutas y ejecutables.
- Se agrega documentación de alertas de Code scanning.

#### Corregido

- Se reduce exposición de secretos en logs.
- Se valida conversión de puertos antes de convertir a `uint32`.
- Se documentan ejecuciones externas auditadas sin shell.
- Se evita `context.Background` directo en goroutines de apagado.
### Fase 10: benchmarks de rendimiento

#### Agregado

- Se agregan benchmarks Go para throughput de conexiones TCP frente a SSH.
- Se agregan benchmarks de validación Go nativa y Rust gRPC.
- Se agregan benchmarks de overhead OPA HTTP con y sin cache.
- Se agregan scripts `scripts/run-benchmarks.sh` y `scripts/benchmark-summary.sh`.
- Se agregan targets de Makefile para ejecutar y limpiar benchmarks.
- Se agrega documentación en `docs/benchmarks/` y `docs/runbooks/`.
