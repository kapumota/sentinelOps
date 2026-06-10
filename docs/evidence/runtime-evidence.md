### Evidencia de runtime

#### Objetivo

La evidencia de runtime permite demostrar que SentinelOps está operando con métricas, trazas, health checks y estado observable en tiempo de ejecución.

#### Generación

```bash
make runtime-evidence
```

#### Contenido del reporte

| Archivo | Descripción |
|---|---|
| `metadata.json` | Metadatos de fecha, rama, commit y endpoints |
| `metrics.prom` | Métricas Prometheus de SentinelOps |
| `health-live.json` | Estado live del Control Plane |
| `health-ready.json` | Estado ready del Control Plane |
| `health-startup.json` | Estado startup del Control Plane |
| `admin-status.json` | Estado autenticado si se configuró contraseña |
| `prometheus-targets.json` | Targets vistos por Prometheus |
| `grafana-health.json` | Health de Grafana |
| `jaeger-services.json` | Servicios vistos por Jaeger |
| `docker-ps.txt` | Estado de contenedores si Docker está disponible |

#### Uso recomendado

Adjunta el directorio generado como evidencia de validación local, demo técnica o revisión de release. No versionar reportes completos salvo que sean evidencia curada para una release específica.

#### Criterios de aceptación

- El reporte no debe contener secretos.
- El archivo `metrics.prom` debe incluir métricas `sentinelops_`.
- Prometheus debe mostrar el target `sentinelops` como activo.
- Los health checks deben responder sin error.
- El reporte debe indicar commit y rama usados para la ejecución.
