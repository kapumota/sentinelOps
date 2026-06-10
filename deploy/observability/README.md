### Observabilidad operacional

#### Objetivo

Esta carpeta contiene la configuración reproducible de observabilidad de SentinelOps para runtime local con Docker Compose.

#### Componentes

| Componente | Puerto local | Uso |
|---|---:|---|
| Prometheus | 9090 | Recolección de métricas |
| Grafana | 3000 | Dashboards operacionales |
| Jaeger | 16686 | Trazas OpenTelemetry |
| SentinelOps metrics | 9101 | Endpoint `/metrics` |
| SentinelOps API | 9444 | Control Plane HTTPS |

#### Comandos

```bash
make observability-up
make observability-smoke
make runtime-evidence
make observability-down
```

#### Credenciales locales

Grafana usa credenciales generadas en `.env.local` por `make generate-secrets`:

```text
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=<generado>
```

Estas credenciales no se versionan y deben rotarse si el entorno local se comparte.
