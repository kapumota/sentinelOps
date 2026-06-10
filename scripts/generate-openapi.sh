#!/usr/bin/env bash
set -euo pipefail

mkdir -p docs
cp internal/controlplane/httpapi/openapi.json docs/openapi.json
cp internal/controlplane/httpapi/openapi.json docs/swagger.json

cat > docs/openapi.yaml <<'YAML'
openapi: 3.0.3
info:
  title: SentinelOps Control API
  version: v1
  description: API administrativa versionada para health checks, estado operativo, sesiones, túneles y documentación OpenAPI de SentinelOps.
servers:
  - url: https://localhost:9443
    description: Servidor local HTTPS
  - url: https://localhost:9445
    description: Servidor local HTTPS con OPA sidecar
tags:
  - name: health
    description: Endpoints para liveness, readiness y startup probes
  - name: admin
    description: Operaciones administrativas protegidas con Basic Auth
  - name: docs
    description: Documentación OpenAPI y Swagger UI
paths:
  /healthz/live:
    get:
      tags: [health]
      summary: Liveness probe
      responses:
        "200":
          description: Proceso vivo
  /healthz/ready:
    get:
      tags: [health]
      summary: Readiness probe
      responses:
        "200":
          description: Proceso listo para recibir tráfico
  /healthz/startup:
    get:
      tags: [health]
      summary: Startup probe
      responses:
        "200":
          description: Inicialización completada
  /api/v1/admin/status:
    get:
      tags: [admin]
      summary: Estado general del sistema
      security:
        - BasicAuth: []
      responses:
        "200":
          description: Estado general
        "401":
          description: No autorizado
  /api/v1/admin/sessions:
    get:
      tags: [admin]
      summary: Listar sesiones activas
      security:
        - BasicAuth: []
      responses:
        "200":
          description: Sesiones activas
  /api/v1/admin/tunnels:
    get:
      tags: [admin]
      summary: Listar túneles activos
      security:
        - BasicAuth: []
      responses:
        "200":
          description: Túneles activos
  /api/v1/admin/tunnels/{id}/close:
    post:
      tags: [admin]
      summary: Cerrar túnel activo
      security:
        - BasicAuth: []
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: Túnel cerrado
        "404":
          description: Túnel no encontrado
  /api/v1/docs/swagger.json:
    get:
      tags: [docs]
      summary: Especificación OpenAPI en JSON
      responses:
        "200":
          description: OpenAPI JSON
components:
  securitySchemes:
    BasicAuth:
      type: http
      scheme: basic
YAML

echo "Documentación OpenAPI generada en docs/"
