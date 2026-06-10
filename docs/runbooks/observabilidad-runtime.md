### Runbook de observabilidad runtime

#### Objetivo

Este runbook describe cómo levantar, verificar y recolectar evidencia operacional de SentinelOps en un entorno local reproducible.

#### Requisitos

- Docker y Docker Compose.
- Archivo `.env.local` generado con `make generate-secrets`.
- Puertos locales disponibles: `3000`, `9090`, `9101`, `9444` y `16686`.

#### Levantar el stack

```bash
make generate-secrets
make observability-up
```

#### Verificar servicios

```bash
make observability-smoke
```

Endpoints esperados:

| Servicio | URL |
|---|---|
| Métricas SentinelOps | `http://localhost:9101/metrics` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` |
| Jaeger | `http://localhost:16686` |
| Control Plane | `https://localhost:9444` |

#### Recolectar evidencia

```bash
make runtime-evidence
```

El reporte se guarda en:

```text
reports/runtime/<timestamp>/
```

#### Señales mínimas esperadas

- Prometheus debe mostrar el target `sentinelops` en estado `UP`.
- Grafana debe cargar el dashboard `SentinelOps runtime`.
- Jaeger debe exponer la API `/api/services`.
- El endpoint `/metrics` debe incluir métricas con prefijo `sentinelops_`.
- Los health checks `/healthz/live`, `/healthz/ready` y `/healthz/startup` deben responder.

#### Diagnóstico rápido

```bash
docker compose -f docker-compose.observability.yml ps
docker compose -f docker-compose.observability.yml logs --tail=100 sentinelops
curl -s http://localhost:9101/metrics | grep sentinelops_
curl -s http://localhost:9090/api/v1/targets
```

#### Apagar el stack

```bash
make observability-down
```
