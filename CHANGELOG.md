### Changelog

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
