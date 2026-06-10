### Validación de SentinelOps v2.4.1

SentinelOps v2.4.1 fue validado como software funcional de laboratorio DevSecOps.

La validación cubrió los siguientes escenarios:

- ejecución local en modo TCP,
- ejecución local en modo SSH,
- ejecución con Docker simple,
- ejecución con Docker Compose,
- prueba E2E completa,
- despliegue en Kubernetes con Helm sobre Minikube,
- API HTTPS administrativa,
- métricas Prometheus,
- autenticación de usuarios,
- comandos internos de laboratorio,
- túneles SSH locales,
- túneles SSH remotos,
- consulta y cierre de túneles por API,
- auditoría Python,
- validador Rust,
- políticas OPA/Rego.

El proyecto quedó validado como **laboratorio técnico reproducible**, no como sistema de acceso remoto de producción.

### Entorno de validación

#### Sistema base

Validación realizada en entorno Linux local con:

- Docker,
- Docker Compose v2,
- Minikube,
- Kubernetes local,
- Helm,
- kubectl,
- Go,
- Rust/Cargo,
- Python,
- OPA,
- OpenSSH client,
- netcat,
- curl,
- make.

#### Puertos usados durante la validación

Para evitar conflictos con servicios locales, se usaron puertos alternativos:

| Servicio | Puerto externo | Puerto interno |
|---|---:|---:|
| SSH Docker/Compose | `2223` | `2222` |
| TCP local/Kubernetes | `2325` | `2323` |
| Métricas Docker/Compose | `9101` | `9001` |
| Métricas Kubernetes | `9101` | `9000` |
| API HTTPS | `9444` | `9443` |
| Túnel local SSH | `9901` | variable |
| Túnel remoto SSH | `10080` | variable |

### Resultado general

#### Estado final

| Área | Resultado |
|---|---|
| Formato Go | Validado |
| `go vet` | Validado |
| Pruebas Go | Validado |
| Pruebas Rust | Validado |
| Build Rust release | Validado |
| Auditoría perfil `hardened` | Validado |
| Auditoría perfil `insecure` | Validado como perfil demostrativo |
| OPA/Rego | Validado |
| Docker build | Validado |
| Docker simple | Validado |
| Docker Compose | Validado |
| E2E completa | Validado |
| Helm template/lint | Validado |
| Kubernetes/Minikube | Validado |
| API HTTPS | Validado |
| Métricas Prometheus | Validado |
| TCP interactivo | Validado |
| SSH interactivo | Validado |
| Túnel local SSH | Validado |
| Túnel remoto SSH | Validado |
| Cierre de túneles por API | Validado |

### Validación de código

#### Formato Go

Comando:

```bash
make fmt
```

Resultado:

```text
pass
```

#### Análisis estático Go

Comando:

```bash
make vet
```

Resultado:

```text
pass
```

#### Pruebas Go

Comando:

```bash
make test
```

Resultado:

```text
pass
```

#### Pruebas por nivel agregadas en fase 2

Comandos:

```bash
make test-unit
make test-integration
make test-race
make test-coverage
```

Resultado esperado:

```text
pass
```

Notas:

- `make test-unit` ejecuta pruebas rápidas sin contenedores.
- `make test-integration` ejecuta pruebas con build tag `containers` y requiere Docker.
- En ejecución local, el target desactiva Ryuk con `TESTCONTAINERS_RYUK_DISABLED=true` para evitar fallos del contenedor reaper cuando Docker no permite ese flujo.
- La prueba con Prometheus real se ejecuta solo si `SENTINELOPS_PROMETHEUS_CONTAINER_TEST=true`; el target normal valida el endpoint `/metrics` de SentinelOps sin depender de Prometheus como contenedor obligatorio.
- `make test-race` valida acceso concurrente en paquetes internos.
- `make test-e2e-containers` requiere `SENTINELOPS_E2E_IMAGE`.


Durante la validación se detectó y corrigió un error de tipo en:

```text
internal/server/tcp_integration_test.go
```

El error era:

```text
cannot use body (variable of type []byte) as string value in return statement
```

Corrección aplicada:

```go
return string(body)
```

Después de la corrección, `go vet` y `go test` pasaron correctamente.

### Validación Rust

#### Pruebas unitarias Rust

Comando:

```bash
make rust-test
```

Resultado:

```text
pass
```

Se validaron los casos del componente:

```text
rust/input-guard
```

Incluyendo:

- comandos simples válidos,
- comandos con argumentos,
- entradas vacías,
- entradas demasiado largas,
- tokens prohibidos,
- caracteres no soportados.

#### Build release Rust

Comando:

```bash
make rust-build
```

Resultado:

```text
pass
```

Binario generado:

```text
rust/input-guard/target/release/input-guard
```
### Validación de auditoría

#### Perfil hardened

Comando:

```bash
make audit PROFILE=hardened
```

Resultado:

```text
pass
```

Interpretación:

```text
El perfil hardened cumple las reglas esperadas para el laboratorio.
```

#### Perfil insecure

Comando:

```bash
make audit PROFILE=insecure
```

Resultado esperado:

```text
fail
```

Interpretación:

```text
El perfil insecure está diseñado para demostrar hallazgos de configuración débil.El fallo es esperado y útil para fines didácticos.
```

### Validación de políticas OPA/Rego

#### Perfil hardened

Comando:

```bash
make policy PROFILE=hardened
```

Resultado:

```text
pass
```

#### Perfil insecure

Comando:

```bash
make policy PROFILE=insecure
```

Resultado:

```text
fallos esperados por perfil demostrativo
```

### Validación local en modo TCP

Valida SentinelOps ejecutándose directamente en la máquina local usando transporte TCP.

#### Comando de ejecución

```bash
make run-tcp ENV_FILE=env/dev-tcp.env \
  APP_ADDR=:2325 \
  METRICS_ADDR=:9101 \
  APP_CONTROL_API_ADDR=:9444
```

#### Prueba de conexión TCP

```bash
nc localhost 2325
```

Credenciales usadas:

```text
student
<LAB_PASSWORD_STUDENT>
```

#### Comandos internos validados

```text
help
whoami
status
profile
audit
policy
tunnels
quit
```

#### API y métricas

```bash
curl http://localhost:9101/metrics
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
```

Resultado:

```text
pass
```
### Validación local en modo SSH

Valida SentinelOps ejecutándose directamente en la máquina local usando transporte SSH.

#### Preparación de llaves

```bash
make ssh-lab-setup USER_NAME=student
make ssh-lab-setup USER_NAME=teacher
make ssh-lab-setup USER_NAME=auditor
make ssh-lab-setup USER_NAME=admin
```

Permisos aplicados:

```bash
chmod 700 data/ssh/client
chmod 600 data/ssh/client/*_ed25519
chmod 644 data/ssh/client/*.pub 2>/dev/null || true
```

#### Comando de ejecución

```bash
make run-ssh ENV_FILE=env/dev-ssh.env \
  APP_SSH_ADDR=:2223 \
  METRICS_ADDR=:9101 \
  APP_CONTROL_API_ADDR=:9444 \
  APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9101,localhost:9101
```

#### Conexión SSH

```bash
ssh -T -p 2223 -i data/ssh/client/student_ed25519 student@localhost
```

#### Comandos internos validados

```text
help
whoami
status
audit
policy
tunnels
quit
```

Resultado:

```text
pass
```

### Validación de Docker simple

Valida la imagen final ejecutando un único contenedor `sentinelops-local`.

#### Construcción de imagen

```bash
make docker-build
```

Imagen generada:

```text
sentinelops:local
```

#### Ejecución Docker simple en modo SSH

```bash
docker run --rm -d \
  --name sentinelops-local \
  -p 2223:2222 \
  -p 9101:9001 \
  -p 9444:9443 \
  -v "$PWD/data:/app/data" \
  -v "$PWD/reports:/app/reports" \
  -e APP_ENV=container \
  -e APP_PROFILE=hardened \
  -e APP_TRANSPORT=ssh \
  -e APP_SSH_ADDR=:2222 \
  -e METRICS_ADDR=:9001 \
  -e APP_CONTROL_API_ENABLED=true \
  -e APP_CONTROL_API_ADDR=:9443 \
  -e APP_CONTROL_API_USER=admin \
  -e APP_CONTROL_API_PASSWORD='<APP_CONTROL_API_PASSWORD>' \
  -e APP_SSH_LOCAL_FORWARD_ENABLED=true \
  -e APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9001,localhost:9001 \
  -e APP_SSH_LOCAL_ALLOWED_ROLES=student,teacher,auditor,admin \
  -e APP_SSH_REMOTE_FORWARD_ENABLED=false \
  -e APP_AUTH_RATE_LIMIT_ENABLED=true \
  -e APP_AUTH_RATE_LIMIT_MAX_FAILURES=5 \
  -e APP_AUTH_RATE_LIMIT_WINDOW=1m \
  -e APP_AUTH_RATE_LIMIT_LOCKOUT=1m \
  -e APP_STATE_PERSISTENCE_ENABLED=false \
  -e APP_STATE_PERSISTENCE_DIR=/app/data/state \
  -e APP_STATE_SESSIONS_PATH=/app/data/state/sessions.json \
  -e APP_STATE_TUNNELS_PATH=/app/data/state/tunnels.json \
  -e EXTERNAL_AUDIT_ENABLED=true \
  -e EXTERNAL_AUDIT_COMMAND=python3 \
  -e EXTERNAL_AUDIT_SCRIPT=/app/tools/audit/audit.py \
  -e EXTERNAL_VALIDATOR_ENABLED=true \
  -e EXTERNAL_VALIDATOR_BINARY=/app/bin/input-guard \
  -e EXTERNAL_VALIDATOR_FAIL_OPEN=false \
  -e OPA_POLICY_ENABLED=true \
  -e OPA_BINARY=/app/bin/opa \
  -e OPA_POLICY_DIR=/app/policies/kubernetes \
  sentinelops:local
```

#### Verificación

```bash
docker ps
docker logs --tail=100 sentinelops-local
```

Servicios validados:

```text
SSH en localhost:2223
Métricas en localhost:9101
API HTTPS en localhost:9444
```

#### Pruebas

```bash
ssh -T -p 2223 -i data/ssh/client/student_ed25519 student@localhost
curl http://localhost:9101/metrics
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
```

Resultado:

```text
pass
```

### Validación de Docker Compose

Valida el laboratorio completo con dos servicios:

```text
sentinelops
sentinelops-tester
```

#### Correcciones aplicadas

El servicio `tester` fue corregido para evitar error de BusyBox con `sleep infinity`.

Configuración válida:

```yaml
entrypoint: ["/bin/sh", "-lc"]
command: tail -f /dev/null
```

Los puertos fueron parametrizados:

```yaml
ports:
  - "${SSH_PORT:-2223}:2222"
  - "${METRICS_PORT:-9101}:9001"
  - "${API_PORT:-9444}:9443"
```

#### Ejecución

```bash
SSH_PORT=2223 \
METRICS_PORT=9101 \
API_PORT=9444 \
docker compose -f docker-compose.demo.yml up --build -d
```

#### Verificación

```bash
docker ps
docker logs --tail=100 sentinelops
docker logs --tail=100 sentinelops-tester
```

Resultado esperado:

```text
sentinelops          Up
sentinelops-tester   Up
```

Servicios validados:

```text
SSH:      localhost:2223
Métricas: localhost:9101
API:      localhost:9444
```

#### Pruebas externas

```bash
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
curl http://localhost:9101/metrics
ssh -T -p 2223 -i data/ssh/client/student_ed25519 student@localhost
```

#### Pruebas desde contenedor tester

```bash
docker exec -it sentinelops-tester sh
```

Dentro del contenedor:

```sh
curl -k https://sentinelops:9443/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://sentinelops:9443/api/admin/status
curl http://sentinelops:9001/metrics
nc -z sentinelops 2222
```

Resultado:

```text
pass
```

### Validación E2E completa

Valida el flujo completo de SentinelOps con Docker Compose, SSH, API, métricas, túneles y evidencias.

#### Comando

```bash
SSH_PORT=2223 \
METRICS_PORT=9101 \
API_PORT=9444 \
LOCAL_FORWARD_PORT=9901 \
REMOTE_BIND_PORT=10080 \
make e2e-full
```

El target `e2e-full` ejecuta `scripts/test-e2e-full.sh`, que prepara llaves, levanta Docker Compose, espera servicios, ejecuta comandos SSH, consulta API, valida métricas, prueba túneles y genera evidencias.

#### Pasos validados

La prueba E2E validó:

- dependencias requeridas,
- generación de llaves SSH,
- levantamiento de Docker Compose,
- disponibilidad del puerto SSH,
- disponibilidad de `/healthz`,
- disponibilidad de `/api/admin/status` con Basic Auth,
- disponibilidad de métricas,
- exportación de estado inicial,
- comandos SSH `help`, `status`, `whoami`, `audit`, `policy`,
- métricas directas,
- túnel local SSH,
- listado de túneles por API,
- cierre de túnel local por API,
- túnel remoto SSH,
- cierre de túnel remoto por API,
- exportación de sesiones finales,
- generación de acta y evidencias.

#### Evidencias generadas

Directorio:

```text
reports/e2e/latest
```

Archivos principales:

```text
acta-validacion.txt
resultados.json
run.log
api-healthz.txt
api-status.json
api-sessions.json
api-tunnels-inicial.json
api-tunnels-local.json
api-tunnels-remote.json
metrics-direct.txt
metrics-local-forward.txt
metrics-remote-forward.txt
ssh-help.txt
ssh-status.txt
ssh-whoami.txt
ssh-audit.txt
ssh-policy.txt
docker-sentinelops.log
docker-sentinelops-tester.log
docker-compose-ps.txt
```

Resultado:

```text
pass
```

### Validación de Kubernetes con Helm

#### Objetivo

Valida despliegue de SentinelOps en Kubernetes local usando Minikube y Helm.

#### Preparación de Minikube

```bash
minikube delete
minikube start --driver=docker
```

#### Construcción de imagen dentro de Minikube

```bash
eval "$(minikube docker-env)"
make docker-build
docker images | grep sentinelops
```

Nota:

```text
No se usó minikube image load como flujo principal.
Se construyó la imagen directamente dentro del Docker interno de Minikube.
```

#### Corrección aplicada para Kubernetes

Se validó que Kubernetes debe ejecutar SentinelOps en modo TCP.

Motivo:

```text
El perfil hardened usa filesystem de solo lectura.
El modo SSH intenta generar data/ssh/host_ed25519_key.
Eso produce error read-only file system si no se cambia el transporte.
```

Configuración requerida en Helm:

```yaml
APP_TRANSPORT: {{ .Values.config.transport | default "tcp" | quote }}
```

Y en `values.yaml`:

```yaml
config:
  transport: tcp
```

#### Instalación con Helm

```bash
kubectl create namespace sentinelops 2>/dev/null || true

helm upgrade --install sentinelops deploy/helm/sentinelops \
  --namespace sentinelops \
  -f deploy/helm/sentinelops/values.yaml \
  -f deploy/helm/sentinelops/values-hardened.yaml \
  --set replicaCount=1
```

#### Estado del despliegue

Comandos:

```bash
kubectl get pods -n sentinelops -o wide
kubectl get svc -n sentinelops
kubectl logs -n sentinelops deploy/sentinelops --tail=100
```

Resultado observado:

```text
Pod:      Running 1/1
Service:  ClusterIP
Ports:    2323/TCP, 9000/TCP, 9443/TCP
```

Logs esperados:

```text
servidor de métricas escuchando addr=:9000
servidor TCP escuchando addr=:2323
API de control escuchando direccion=:9443
```

#### Port-forward

```bash
kubectl port-forward -n sentinelops svc/sentinelops \
  2325:2323 \
  9101:9000 \
  9444:9443
```

#### Pruebas

En otra terminal:

```bash
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
curl http://localhost:9101/metrics
nc localhost 2325
```

Credenciales TCP:

```text
student
<LAB_PASSWORD_STUDENT>
```

Comandos internos validados:

```text
help
whoami
status
audit
policy
quit
```

Resultado:

```text
pass
```

### Incidencias encontradas y corregidas

#### Error de tipo en prueba Go

Error:

```text
cannot use body (variable of type []byte) as string value in return statement
```

Corrección:

```go
return string(body)
```

Estado:

```text
corregido
```

#### Puerto de métricas ocupado

Error:

```text
metrics: listen tcp :9001: bind: address already in use
```

Corrección:

```text
usar puertos alternativos:
METRICS_ADDR=:9101
API=:9444
SSH=:2223
```

Estado:

```text
corregido
```

#### Docker Compose tester reiniciando

Error:

```text
BusyBox sleep infinity
Usage: sleep [N]...
```

Corrección:

```yaml
command: tail -f /dev/null
```

Estado:

```text
corregido
```

#### Permisos de volumen Docker

Error:

```text
mkdir data/ssh: permission denied
```

Corrección:

```bash
chmod 777 data/ssh
chmod -R a+rwX data/controlplane data/state reports
```

Estado:

```text
corregido para laboratorio local
```

#### Clave privada SSH con permisos abiertos

Error:

```text
WARNING: UNPROTECTED PRIVATE KEY FILE
Permissions 0666 are too open
```

Corrección:

```bash
chmod 700 data/ssh/client
chmod 600 data/ssh/client/*_ed25519
chmod 644 data/ssh/client/*.pub 2>/dev/null || true
```

Estado:

```text
corregido
```

#### API administrativa sin autenticación en E2E

Problema:

```text
/api/admin/status requiere Basic Auth.
```

Corrección:

```bash
curl -ksf -u "${API_USER}:${API_PASSWORD}" "https://localhost:${API_PORT}/api/admin/status"
```

Estado:

```text
corregido
```

#### Extracción de ID de túnel local/remoto

Problema:

```text
El JSON tenía túneles, pero la función no extraía el ID correctamente.
```

Causa:

```text
uso incorrecto de stdin al pasar código Python y JSON simultáneamente.
```

Corrección:

```text
usar python3 -c para que stdin quede disponible para el JSON.
```

Estado:

```text
corregido
```

#### Kubernetes CrashLoopBackOff

Error:

```text
read-only file system
open data/ssh/host_ed25519_key
```

Corrección:

```text
Kubernetes debe usar APP_TRANSPORT=tcp.
```

Estado:

```text
corregido
```

#### minikube image load fallando

Error:

```text
GUEST_IMAGE_LOAD
unable to calculate manifest
```

Corrección:

```bash
eval "$(minikube docker-env)"
make docker-build
```

Estado:

```text
corregido mediante build directo dentro del Docker de Minikube
```
### Limitaciones conocidas

#### No es software de producción

SentinelOps no debe presentarse como reemplazo de OpenSSH ni como solución de acceso remoto general para producción.

Debe presentarse como:

```text
laboratorio DevSecOps reproducible
```

#### Credenciales demo

El proyecto usa credenciales de laboratorio:

```text
admin/<APP_CONTROL_API_PASSWORD>
student/<LAB_PASSWORD_STUDENT>
teacher/<LAB_PASSWORD_TEACHER>
auditor/<LAB_PASSWORD_AUDITOR>
```

Estas credenciales deben cambiarse en cualquier entorno no académico.

#### Permisos amplios en Docker local

Algunas pruebas Docker usan permisos amplios sobre carpetas como:

```text
data/ssh
data/controlplane
data/state
reports
```

Esto es aceptable para laboratorio local, pero no representa una política de producción.

#### Kubernetes validado en modo TCP

El chart Helm fue validado en modo TCP.

El modo SSH en Kubernetes requiere configuración adicional de volúmenes escribibles para host keys y exposición del puerto SSH.

### Limpieza posterior a validación

#### Detener servicios

```bash
eval "$(minikube docker-env -u)" 2>/dev/null || true

docker rm -f sentinelops-local sentinelops sentinelops-tester 2>/dev/null || true
docker compose -f docker-compose.demo.yml down 2>/dev/null || true

helm uninstall sentinelops -n sentinelops 2>/dev/null || true
kubectl delete namespace sentinelops 2>/dev/null || true
```

#### Limpiar artefactos generados

```bash
make cleanup

rm -rf bin
rm -rf dist
rm -rf tmp
rm -rf .tmp
rm -rf coverage.out
rm -rf coverage.html
rm -rf rust/input-guard/target
rm -rf reports/e2e
rm -rf reports/*
```

#### Eliminar secretos y datos runtime

```bash
rm -f data/ssh/client/*_ed25519
rm -f data/ssh/client/*_ed25519.pub
rm -f data/ssh/host_ed25519_key
rm -f data/ssh/host_ed25519_key.pub
rm -f data/controlplane/tls.crt
rm -f data/controlplane/tls.key
rm -f data/state/*.json
```

#### Mantener estructura de carpetas

```bash
mkdir -p data/ssh/client
mkdir -p data/ssh/authorized_keys
mkdir -p data/controlplane
mkdir -p data/state
mkdir -p reports

touch data/ssh/client/.gitkeep
touch data/ssh/authorized_keys/.gitkeep
touch data/controlplane/.gitkeep
touch data/state/.gitkeep
touch reports/.gitkeep
```

### Verificación antes de subir a GitHub

#### Simulación de commit

```bash
git status --short
git add -n .
```

#### Archivos que no deben subirse

No subir:

```text
claves privadas
certificados generados
tls.key
host keys SSH
reports/e2e
binarios compilados
rust/input-guard/target
snapshots JSON de runtime
cachés
logs
```

#### Validación final recomendada

```bash
make fmt
make vet
make test
make rust-test
make rust-build
make audit PROFILE=hardened
make policy PROFILE=hardened
make helm-lint
make helm-template PROFILE=hardened
make e2e-full
```

Si se ejecuta `make e2e-full` antes de subir, limpiar después:

```bash
rm -rf reports/e2e
touch reports/.gitkeep
```

### Veredicto final

SentinelOps v2.4.1 queda validado como **software real de laboratorio DevSecOps**.

El proyecto demuestra correctamente:

- servidor TCP,
- servidor SSH,
- autenticación por usuario, contraseña y clave pública,
- comandos internos de laboratorio,
- API HTTPS administrativa,
- métricas Prometheus,
- auditoría Python,
- validación Rust,
- policy-as-code con OPA/Rego,
- túneles SSH locales y remotos,
- control de túneles por API,
- ejecución local,
- ejecución con Docker simple,
- ejecución con Docker Compose,
- prueba E2E completa,
- despliegue Kubernetes con Helm.

Resultado final:

```text
VALIDACIÓN APROBADA 
```

### Validación de fase 3 - OpenTelemetry

#### Objetivo de validación

Comprobar que SentinelOps puede ejecutarse con trazas distribuidas opcionales sin romper el flujo normal de pruebas, seguridad y despliegue.

#### Validación local básica

```bash
make check-secrets
make fmt
make vet
make test
make rust-test
```

#### Validación de dependencias

Después de aplicar la fase 3, ejecutar:

```bash
go mod tidy
git status --short
```

Si `go.mod` o `go.sum` cambian por dependencias de OpenTelemetry, esos cambios deben versionarse en la rama de fase 3.

#### Validación con Jaeger

```bash
make generate-secrets
source .env.local
make run-jaeger
make run-ssh-telemetry
```

`OTEL_TRACES_ENABLED=false` es el valor seguro por defecto. Para Jaeger UI, `make run-ssh-telemetry` activa explícitamente `OTEL_TRACES_ENABLED=true` y usa `OTEL_EXPORTER_TYPE=otlp-grpc`.

En otra terminal, conectarse al servidor SSH o consultar la API de control:

```bash
source .env.local
curl -k https://localhost:9443/healthz
curl -k -u "$APP_CONTROL_API_USER:$APP_CONTROL_API_PASSWORD" https://localhost:9443/api/admin/status
```

Comprobar que Jaeger recibió el servicio:

```bash
curl -s http://localhost:16686/api/services
```

Luego abrir:

```text
http://localhost:16686
```

Buscar el servicio desde el panel izquierdo:

```text
Service -> sentinelops
Operation -> control_api.request
Find Traces
```

No escribir `sentinelops` en el cuadro superior derecho de Jaeger, porque ese campo espera un trace ID.

#### Validación con Docker Compose

```bash
source .env.local
make docker-observability-up
```

Verificar:

```text
http://localhost:16686
http://localhost:9090
https://localhost:9444/healthz
```

#### Limpieza local

```bash
make docker-observability-down
make stop-jaeger
```

Si se generaron artefactos de cobertura durante la validación:

```bash
rm -f coverage.out coverage.html
```

#### Criterio de aceptación

La fase 3 se considera válida cuando:

- `make test` pasa.
- `make rust-test` pasa.
- `make check-secrets` no detecta credenciales conocidas.
- SentinelOps arranca con `OTEL_TRACES_ENABLED=true`.
- Jaeger muestra spans del servicio `sentinelops`.
- La API de control devuelve `X-Correlation-ID` y `X-Trace-ID`.

### Validación de fase 4 - OPA sidecar runtime

#### Objetivo

Validar que SentinelOps pueda consultar OPA en runtime mediante HTTP, sin eliminar el modo anterior basado en el binario `opa eval`.

#### Variables relevantes

```env
OPA_POLICY_ENABLED=true
OPA_POLICY_MODE=http
OPA_POLICY_URL=http://localhost:8181
OPA_POLICY_TIMEOUT=2s
OPA_POLICY_CACHE_ENABLED=true
OPA_POLICY_CACHE_TTL=30s
```

#### Validación unitaria

```bash
make check-secrets
make fmt
make vet
make test
make rust-test
```

#### Validación Rego local

```bash
make opa-test
make opa-build
```

#### Validación con sidecar OPA

```bash
make generate-secrets
source .env.local
make run-opa-sidecar
```

El target usa `HOST_UID` y `HOST_GID` para que el usuario dentro del contenedor pueda escribir en `./data`. Si SentinelOps queda reiniciando, inspecciona:

```bash
docker compose -f docker-compose.opa.yml ps
docker compose -f docker-compose.opa.yml logs --tail=200 sentinelops
```

Verificar OPA:

```bash
curl -s http://localhost:8181/health
```

Verificar SentinelOps:

```bash
curl -k https://localhost:9445/healthz
```

Ejecutar el comando de política dentro del flujo del servidor permite validar que SentinelOps consulte el sidecar cuando `OPA_POLICY_MODE=http`.

#### Resultado esperado

```text
OPA responde por HTTP en :8181
SentinelOps arranca con OPA_POLICY_MODE=http
El modo exec sigue disponible con OPA_POLICY_MODE=exec
```

#### Limpieza

```bash
make stop-opa-sidecar
```

### Validación de fase 5 - OpenAPI y API v1

#### Objetivo

Validar que la API de control tenga endpoints versionados, health checks compatibles con Kubernetes y documentación OpenAPI accesible por HTTP.

#### Validación estática

```bash
make docs
make docs-check
make fmt
make vet
make test
make rust-test
```

#### Validación de health checks

Con SentinelOps en ejecución:

```bash
curl -k https://localhost:9443/healthz/live
curl -k https://localhost:9443/healthz/ready
curl -k https://localhost:9443/healthz/startup
```

#### Validación de documentación

```bash
curl -k https://localhost:9443/api/v1/docs/swagger.json
curl -k https://localhost:9443/api/v1/docs/swagger/
```

#### Validación de API v1

```bash
source .env.local
curl -k -u "$APP_CONTROL_API_USER:$APP_CONTROL_API_PASSWORD" \
  https://localhost:9443/api/v1/admin/status

curl -k -u "$APP_CONTROL_API_USER:$APP_CONTROL_API_PASSWORD" \
  https://localhost:9443/api/v1/admin/sessions

curl -k -u "$APP_CONTROL_API_USER:$APP_CONTROL_API_PASSWORD" \
  https://localhost:9443/api/v1/admin/tunnels
```

#### Validación automatizada local

```bash
make api-smoke
```

Si se usa OPA sidecar:

```bash
API_URL=https://localhost:9445 make api-smoke
```

#### Validación Helm

```bash
make helm-template
```

El render debe mostrar probes HTTP HTTPS contra el puerto `control-api`:

```text
/healthz/live
/healthz/ready
/healthz/startup
```

#### Resultado esperado

```text
La API v1 responde con Basic Auth.
Los health checks responden sin autenticación.
La especificación OpenAPI es JSON válido.
Los endpoints legados siguen disponibles.
```

### Fase 6 - Validación del validador Rust gRPC

#### Validación básica

El flujo normal debe seguir pasando sin levantar gRPC:

```bash
make check-secrets
make fmt
make vet
make test
make rust-test
```

#### Validación del proto

```bash
make proto-go
make proto-clean
```

`make proto-go` requiere `protoc`, `protoc-gen-go` y `protoc-gen-go-grpc`. Si no están instalados:

```bash
make proto-tools
```

#### Validación del servidor Rust gRPC

```bash
make validator-grpc-build
make validator-grpc-test
```

#### Validación con Docker Compose

```bash
source .env.local
make validator-grpc-up
```

En otra terminal:

```bash
source .env.local
curl -k https://localhost:9446/healthz/live
curl -k https://localhost:9446/api/v1/docs/swagger.json
API_URL=https://localhost:9446 make validator-grpc-smoke
```

Luego:

```bash
make validator-grpc-down
```

#### Limpieza

```bash
make proto-clean
make opa-clean
rm -f coverage.out coverage.html
```

No se deben versionar:

```text
gen/go
rust/input-guard-grpc/target
policies/bundle
```

#### Validación extendida del cliente Go gRPC

El cliente Go gRPC es opt-in y usa el build tag `grpcvalidator`.

```bash
make proto-tools
make proto-go
go test -tags grpcvalidator ./internal/security
make proto-clean
```

Si esta validación modifica `go.mod` o `go.sum`, revisa el diff antes de versionarlo. Para esta fase, el flujo obligatorio sigue usando `VALIDATOR_MODE=binary`.

### Fase 7: validación CI/CD DevSecOps

#### Objetivo

Validar localmente los controles principales que ejecuta GitHub Actions antes de abrir un PR.

#### Validación rápida

```bash
bash -n scripts/ci-check.sh
python3 -m json.tool .github/changelog-config.json >/dev/null
git diff --check
```

#### Validación principal

```bash
make check-secrets
make docs-check
make vet
make test
make rust-test
make validator-grpc-build
make validator-grpc-test
```

#### Validación por bloques

```bash
make ci-check
make ci-openapi
make ci-proto
make ci-security
```

#### Limpieza obligatoria

```bash
make ci-clean
rm -f coverage.out coverage.html
```

#### Verificación antes del commit

```bash
git status --short
git status --short --ignored | grep -E "target/|gen/go|coverage|policies/bundle|\.patch" || true
git diff --check
```

Es aceptable ver rutas con `!!` si son artefactos ignorados, por ejemplo:

```text
!! rust/input-guard/target/
!! rust/input-guard-grpc/target/
```

No deben aparecer en `git status --short` normal.
### Fase 8: validación de observabilidad runtime

#### Objetivo

Validar que SentinelOps expone evidencia operacional reproducible mediante métricas, health checks, trazas, dashboard Grafana y reportes locales de runtime.

#### Validación estática

Comandos:

    bash -n scripts/observability-smoke.sh
    bash -n scripts/runtime-evidence.sh
    bash -n scripts/generate-secrets.sh
    python3 -m json.tool deploy/observability/grafana/dashboards/sentinelops-runtime.json >/dev/null
    git diff --check

#### Validación base del proyecto

Comandos:

    make generate-secrets
    make check-secrets
    make docs-check
    make vet
    make test
    make rust-test
    make validator-grpc-build
    make validator-grpc-test

#### Levantar observabilidad

Comando:

    make observability-up

#### Smoke test de observabilidad

En otra terminal:

    source .env.local
    make observability-smoke

El smoke test revisa:

    Control Plane HTTPS
    Prometheus
    Grafana
    Jaeger
    endpoint /metrics
    endpoint /healthz/live
    endpoint /healthz/ready
    endpoint /healthz/startup

#### Generar evidencia runtime

Comandos:

    source .env.local
    make runtime-evidence

La evidencia se genera en:

    reports/runtime/<timestamp>/

#### Apagar stack observable

Comando:

    make observability-down

#### Limpieza local

Comandos:

    make observability-clean
    rm -f coverage.out coverage.html

#### Verificación antes del commit

Comandos:

    git status --short
    git status --short --ignored | grep -E "target/|gen/go|coverage|policies/bundle|reports/runtime|\.patch" || true
    git diff --check

Es aceptable ver rutas ignoradas con `!!` si corresponden a artefactos generados. No deben aparecer en `git status --short` normal.
### Fase 9: validación de persistencia PostgreSQL y Redis

#### Objetivo

Validar que SentinelOps incorpora una abstracción de almacenamiento con modo en memoria, PostgreSQL para persistencia durable y Redis para cache y rate limiting.

#### Validación estática

Comandos:

    bash -n scripts/storage-smoke.sh
    bash -n scripts/generate-secrets.sh
    git diff --check

#### Dependencias Go

Después de aplicar la fase 9 se debe ejecutar:

    go mod tidy

Esto actualiza `go.mod` y puede actualizar `go.sum`.

#### Validación base del proyecto

Comandos:

    make generate-secrets
    make check-secrets
    make vet
    make test
    make storage-test
    make rust-test
    make validator-grpc-build
    make validator-grpc-test

#### Levantar PostgreSQL y Redis

Comando:

    make storage-up

#### Smoke test de almacenamiento

En otra terminal:

    source .env.local
    make storage-smoke

El smoke test revisa:

    PostgreSQL disponible
    Redis disponible
    variables de entorno de storage
    migraciones SQL presentes
    conectividad básica de servicios

#### Apagar stack de almacenamiento

Comando:

    make storage-down

#### Limpieza local

Comando:

    make storage-clean

#### Verificación antes del commit

Comandos:

    git status --short
    git status --short --ignored | grep -E "target/|gen/go|coverage|policies/bundle|reports/runtime|\.patch" || true
    git diff --check

No se debe versionar el patch de fase 9 ni artefactos generados.
