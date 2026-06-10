### SentinelOps

**SentinelOps es software real de laboratorio** diseñado para enseñanza, evaluación técnica y experimentación controlada en DevSecOps.

El proyecto muestra cómo un servicio remoto simple puede evolucionar hacia una solución de laboratorio con transporte seguro, autenticación, auditoría, métricas, API administrativa, validación externa, policy as code y despliegue reproducible. 

SentinelOps no pretende reemplazar OpenSSH ni ofrecer acceso remoto general para producción.

### Estado actual del proyecto

#### Validación general

El proyecto fue validado en los siguientes escenarios:

- ejecución local en modo TCP,
- ejecución local en modo SSH,
- ejecución con Docker simple,
- ejecución con Docker Compose,
- prueba E2E completa,
- despliegue en Kubernetes con Helm sobre Minikube,
- API HTTPS administrativa,
- endpoint de métricas Prometheus,
- autenticación de usuarios,
- túneles SSH locales,
- túneles SSH remotos,
- consulta y cierre de túneles por API,
- auditoría Python,
- validación Rust,
- políticas OPA y Rego.

#### Escenarios comprobados

| Escenario | Estado |
|---|---|
| Local TCP | Validado |
| Local SSH | Validado |
| Docker simple | Validado |
| Docker Compose | Validado |
| E2E completa | Validado |
| Kubernetes con Helm | Validado |
| API HTTPS | Validado |
| Métricas Prometheus | Validado |
| Túnel local SSH | Validado |
| Túnel remoto SSH | Validado |
| Auditoría Python | Validado |
| Validador Rust | Validado |
| OPA y Rego | Validado |

### Credenciales de laboratorio

#### Usuarios disponibles

SentinelOps incluye usuarios de laboratorio para pruebas controladas.

| Usuario | Contraseña | Rol |
|---|---|---|
| `student` | `<LAB_PASSWORD_STUDENT>` | estudiante |
| `teacher` | `<LAB_PASSWORD_TEACHER>` | docente |
| `auditor` | `<LAB_PASSWORD_AUDITOR>` | auditor |
| `admin` | `<LAB_PASSWORD_ADMIN>` | administrador |

#### Generar credenciales locales

Las credenciales se generan dinámicamente y no se versionan.

```bash
make generate-secrets
```

El comando crea `.env.local` con permisos restrictivos. También se puede preparar el entorno local completo con:

```bash
make setup-dev
```

#### Usar las credenciales en desarrollo

```bash
source .env.local
make run-ssh
```

Para revisar las contraseñas activas del entorno local:

```bash
grep 'PASSWORD' .env.local
```

#### API de control

La API HTTPS administrativa usa `APP_CONTROL_API_USER` y `APP_CONTROL_API_PASSWORD`.

```bash
curl -k -u "${APP_CONTROL_API_USER}:${APP_CONTROL_API_PASSWORD}" https://localhost:9443/api/admin/status
```

#### Advertencia de seguridad

Las credenciales se generan para entorno académico y local.

No deben usarse en producción.

Si el proyecto se reutiliza fuera del laboratorio, las credenciales deben reemplazarse por:

- variables de entorno,
- archivos `.env` no versionados,
- Docker secrets,
- Kubernetes Secrets,
- un sistema externo de gestión de secretos.

### Capacidades principales

#### Transporte y acceso remoto

SentinelOps soporta dos modos de transporte.

| Modo | Uso |
|---|---|
| TCP | modo heredado, útil para comparación y Kubernetes |
| SSH | modo principal para laboratorio de acceso remoto seguro |

Incluye:

- servidor TCP interactivo,
- servidor SSH con cifrado real,
- autenticación por contraseña,
- autenticación por clave pública,
- `authorized_keys` por usuario,
- host key persistente,
- cliente SSH mínimo en Go,
- compatibilidad con el cliente `ssh` del sistema.

#### Seguridad y validación

El proyecto integra varias capas defensivas:

- validación rápida en Go,
- validador externo en Rust,
- auditoría externa en Python,
- policy as code con OPA y Rego,
- perfiles `hardened` e `insecure`,
- rate limiting de login por usuario y origen.
- control de túneles SSH por rol y allowlist.

#### Observabilidad y administración

Incluye:

- endpoint de métricas Prometheus,
- API HTTPS administrativa,
- TLS con certificado local o autogenerado,
- Basic Auth para endpoints administrativos,
- sesiones activas en memoria,
- túneles activos en memoria,
- cierre de túneles por API,
- snapshots JSON opcionales de sesiones y túneles.

#### Túneles SSH

SentinelOps permite probar:

- túneles locales SSH,
- túneles remotos SSH,
- listado de túneles activos,
- cierre de túneles por API,
- restricciones por rol,
- restricciones por destino permitido.

### Fundamento técnico

#### Go

SentinelOps usa Go como lenguaje principal.

Aprovecha:

- `crypto/tls` para API HTTPS,
- `crypto/ed25519` para host keys,
- `crypto/x509` para certificados,
- `crypto/rand` para generación segura,
- `golang.org/x/crypto/ssh` para transporte SSH,
- goroutines para manejar sesiones concurrentes,
- registros internos para sesiones y túneles.

#### Rust

El componente Rust funciona como validador externo defensivo.

Se encuentra en:

```text
rust/input-guard
```

Se ejecuta con:

```bash
cargo test --manifest-path rust/input-guard/Cargo.toml
cargo build --release --manifest-path rust/input-guard/Cargo.toml
```

#### Python

La auditoría externa está en:

```text
tools/audit/audit.py
```

Se ejecuta con:

```bash
python3 tools/audit/audit.py --profile hardened --project-root .
python3 tools/audit/audit.py --profile insecure --project-root .
```

#### OPA y Rego

Las políticas están en:

```text
policies/kubernetes
```

Se ejecutan con:

```bash
make policy PROFILE=hardened
make policy PROFILE=insecure
```

### Arquitectura

#### Vista general

```text
Cliente TCP
Cliente SSH
Cliente Go
        │
        ▼
SentinelOps Server
├── transporte TCP o SSH
├── autenticación
├── sesiones
├── shell de laboratorio
├── auditoría Python
├── validación Rust
├── policy as code con OPA y Rego
├── túneles SSH locales y remotos
├── métricas Prometheus
└── API HTTPS de control
```

#### Componentes principales

| Componente | Rol |
|---|---|
| `cmd/server` | servidor principal |
| `cmd/client` | cliente SSH mínimo en Go |
| `internal/server` | transporte TCP heredado |
| `internal/transport/sshserver` | servidor SSH, shell y forwarding |
| `internal/session` | sesiones activas |
| `internal/forwarding` | política, registro y control de túneles |
| `internal/controlplane/httpapi` | API HTTPS administrativa |
| `internal/metrics` | métricas Prometheus |
| `internal/security` | validación defensiva |
| `internal/auth` | autenticación y rate limiting |
| `tools/audit` | auditoría Python |
| `rust/input-guard` | validador externo Rust |
| `policies/kubernetes` | reglas Rego |
| `deploy/helm/sentinelops` | chart Helm |
| `scripts` | automatización local, Docker, E2E y despliegue |

### Requisitos

#### Herramientas necesarias

Instala como mínimo lo siguiente:

- Go 1.25 o superior
- Rust y Cargo
- Python 3.11 o superior
- OPA
- Helm 3
- Docker
- Docker Compose v2
- Make
- OpenSSH client
- `ssh-keygen`
- `nc`
- `curl`
- Minikube para Kubernetes local
- `kubectl`

#### Prepara el entorno en Ubuntu o Debian

Si agregaste el script de instalación:

```bash
make setup-dev
```

O directamente:

```bash
bash scripts/setup-dev-ubuntu.sh
```

Verifica las herramientas:

```bash
go version
cargo --version
rustc --version
python3 --version
opa version
helm version
docker version
docker compose version
kubectl version --client
```

### Puertos usados

#### Puertos por defecto del proyecto

| Servicio | Puerto |
|---|---|
| TCP local | `2324` |
| SSH local | `2222` |
| Métricas local y Docker | `9001` |
| API HTTPS local y Docker | `9443` |

#### Puertos recomendados para experimentos

Para evitar conflictos, se recomienda usar:

| Servicio | Puerto externo recomendado |
|---|---|
| TCP | `2325` |
| SSH | `2223` |
| Métricas | `9101` |
| API HTTPS | `9444` |
| Túnel local | `9901` |
| Túnel remoto | `10080` |

#### Diferencia entre Docker y Kubernetes

En Docker y Docker Compose:

```text
localhost:2223 -> contenedor:2222
localhost:9101 -> contenedor:9001
localhost:9444 -> contenedor:9443
```

En Kubernetes:

```text
localhost:2325 -> service:2323
localhost:9101 -> service:9000
localhost:9444 -> service:9443
```

### Limpieza antes de ejecutar

#### Limpieza general

Desde la raíz del proyecto escribe:

```bash
make cleanup
```

#### Limpieza de Docker

```bash
docker rm -f sentinelops-local 2>/dev/null || true
docker compose -f docker-compose.demo.yml down 2>/dev/null || true
docker rm -f sentinelops sentinelops-tester 2>/dev/null || true
```

#### Limpieza de Kubernetes

```bash
helm uninstall sentinelops -n sentinelops 2>/dev/null || true
kubectl delete namespace sentinelops 2>/dev/null || true
```

#### Permisos recomendados para claves SSH

```bash
chmod 700 data/ssh/client 2>/dev/null || true
chmod 600 data/ssh/client/*_ed25519 2>/dev/null || true
chmod 644 data/ssh/client/*.pub 2>/dev/null || true
chmod -R a+rwX data/controlplane data/state reports 2>/dev/null || true
```

### Validación del código

#### Formato, análisis y pruebas Go

```bash
make fmt
make vet
make test
```

#### Pruebas por nivel

La fase 2 separa las pruebas en niveles para mantener ciclos rápidos de desarrollo y pruebas de integración reproducibles.

```bash
make test-unit
make test-integration
make test-race
make test-coverage
```

`make test-integration` usa `testcontainers-go` y requiere Docker activo. Las pruebas de integración están protegidas con el build tag `containers`, por lo que `make test` sigue ejecutando la suite rápida por defecto.

#### Prometheus con contenedor real

La prueba que levanta un contenedor Prometheus completo es opcional para evitar fallos por red, descarga de imagen o restricciones locales de Docker. Para ejecutarla explícitamente:

```bash
cd tests/integration
SENTINELOPS_PROMETHEUS_CONTAINER_TEST=true \
TESTCONTAINERS_RYUK_DISABLED=true \
go test -tags=containers -v -run TestMetricsIntegrationWithPrometheus -timeout 3m .
cd ../..
```

El target `make test-integration` mantiene como obligatoria la validación del endpoint `/metrics` propio de SentinelOps.

#### Ryuk en entorno local

El target `make test-integration` desactiva Ryuk con `TESTCONTAINERS_RYUK_DISABLED=true` para evitar fallos locales cuando Docker no permite iniciar el contenedor reaper. Las pruebas llaman a `Terminate` sobre cada contenedor creado, por lo que la limpieza normal sigue ocurriendo al cerrar cada test.

Si se desea probar con Ryuk explícitamente, puede ejecutarse el comando manualmente sin esa variable de entorno o configurando Docker para permitir el contenedor reaper.

Para el E2E de imagen completa se debe indicar una imagen local o publicada:

```bash
docker build -t sentinelops:e2e .
SENTINELOPS_E2E_IMAGE=sentinelops:e2e make test-e2e-containers
```

#### Pruebas Rust

```bash
make rust-test
make rust-build
```

#### Validación completa básica

```bash
make check
```

#### Auditoría

```bash
make audit PROFILE=hardened
make audit PROFILE=insecure
```

#### Políticas OPA y Rego

```bash
make policy PROFILE=hardened
make policy PROFILE=insecure
```

#### Helm

```bash
make helm-lint
make helm-template PROFILE=hardened
```

### Ejecución local en modo TCP

#### Qué hace

Ejecuta SentinelOps directamente en tu máquina usando el transporte TCP.

Este modo es útil para:

- validar autenticación básica,
- probar comandos internos,
- verificar API y métricas,
- comparar TCP contra SSH,
- ejecutar Kubernetes en modo TCP.

#### Terminal 1

Inicia el servidor TCP:

```bash
make run-tcp ENV_FILE=env/dev-tcp.env \
  APP_ADDR=:2325 \
  METRICS_ADDR=:9101 \
  APP_CONTROL_API_ADDR=:9444
```

Deja esta terminal abierta.

#### Terminal 2

Conéctate por TCP:

```bash
nc localhost 2325
```

Cuando pida credenciales escribe:

```text
student
<LAB_PASSWORD_STUDENT>
```

Comandos útiles:

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

#### Terminal 3

Prueba API y métricas:

```bash
curl http://localhost:9101/metrics
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
```

### Ejecución local en modo SSH

#### Qué hace

Ejecuta SentinelOps directamente en tu máquina como servidor SSH.

Este modo permite probar:

- cifrado SSH,
- autenticación por clave pública,
- shell de laboratorio,
- comandos internos,
- túneles SSH locales y remotos,
- API administrativa,
- métricas.

#### Prepara las llaves de laboratorio

```bash
make ssh-lab-setup USER_NAME=student
make ssh-lab-setup USER_NAME=teacher
make ssh-lab-setup USER_NAME=auditor
make ssh-lab-setup USER_NAME=admin
```

Corrige permisos:

```bash
chmod 700 data/ssh/client
chmod 600 data/ssh/client/*_ed25519
chmod 644 data/ssh/client/*.pub 2>/dev/null || true
```

#### Terminal 1

Inicia el servidor SSH:

```bash
make run-ssh ENV_FILE=env/dev-ssh.env \
  APP_SSH_ADDR=:2223 \
  METRICS_ADDR=:9101 \
  APP_CONTROL_API_ADDR=:9444 \
  APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9101,localhost:9101
```

Deja esta terminal abierta.

#### Terminal 2

Conéctate por SSH:

```bash
ssh -T -p 2223 -i data/ssh/client/student_ed25519 student@localhost
```

Dentro de SentinelOps:

```text
help
whoami
status
audit
policy
tunnels
quit
```

#### Terminal 3

Prueba API y métricas:

```bash
curl http://localhost:9101/metrics
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
```

### Túnel local SSH

#### Qué hace

Un túnel local permite exponer un puerto local que viaja por SSH hacia un destino permitido.

```text
localhost:9901
        │
        ▼
túnel SSH por SentinelOps
        │
        ▼
métricas permitidas
```

#### Con servidor local SSH

Si el servidor local usa métricas en `9101`:

```bash
ssh -N -T -p 2223 \
  -i data/ssh/client/student_ed25519 \
  -L 9901:127.0.0.1:9101 \
  student@localhost
```

En otra terminal:

```bash
curl http://localhost:9901/metrics
```

#### Con Docker o Docker Compose

Si el destino está dentro del contenedor, usa el puerto interno `9001`:

```bash
ssh -N -T -p 2223 \
  -i data/ssh/client/student_ed25519 \
  -L 9901:127.0.0.1:9001 \
  student@localhost
```

En otra terminal:

```bash
curl http://localhost:9901/metrics
```

### Docker simple

#### Qué hace

Docker simple ejecuta un solo contenedor llamado:

```text
sentinelops-local
```

Es útil para:

- probar la imagen final,
- ejecutar SSH sin Docker Compose,
- probar API y métricas en contenedor,
- probar túneles manualmente,
- inspeccionar logs directamente.

#### Vuelve al Docker normal si usaste Minikube

Si antes ejecutaste:

```bash
eval "$(minikube docker-env)"
```

vuelve al Docker normal:

```bash
eval "$(minikube docker-env -u)"
```
#### Construye la imagen

```bash
make docker-build
```

Verifica la imagen:

```bash
docker images | grep sentinelops
```

Debe aparecer:

```text
sentinelops   local
```

#### Ejecuta Docker simple en modo SSH

```bash
docker rm -f sentinelops-local 2>/dev/null || true

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

#### Verifica el contenedor

```bash
docker ps
docker logs --tail=100 sentinelops-local
```

Debe mostrar:

```text
servidor de métricas escuchando
servidor SSH escuchando
API de control escuchando
```

#### Prueba SSH

```bash
ssh -T -p 2223 -i data/ssh/client/student_ed25519 student@localhost
```

#### Prueba API y métricas

```bash
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
curl http://localhost:9101/metrics
```

#### Detiene Docker simple

```bash
docker rm -f sentinelops-local
```

### Docker Compose

#### Qué hace

Docker Compose levanta un entorno de laboratorio con varios contenedores.

Normalmente incluye:

```text
sentinelops
sentinelops-tester
```

El servicio `sentinelops` ejecuta la aplicación principal.

El servicio `sentinelops-tester` contiene herramientas como:

- `curl`
- `ssh`
- `nc`
- `python3`

Sirve para pruebas E2E y validación reproducible.

#### Verifica  docker-compose.demo.yml

El servicio `sentinelops` debe publicar puertos con variables:

```yaml
ports:
  - "${SSH_PORT:-2223}:2222"
  - "${METRICS_PORT:-9101}:9001"
  - "${API_PORT:-9444}:9443"
```

El servicio `tester` debe usar:

```yaml
entrypoint: ["/bin/sh", "-lc"]
command: tail -f /dev/null
```

No usar:

```yaml
command: sleep infinity
```

porque en Alpine y BusyBox puede fallar.

#### Levanta Docker Compose

```bash
SSH_PORT=2223 \
METRICS_PORT=9101 \
API_PORT=9444 \
docker compose -f docker-compose.demo.yml up --build -d
```

#### Verifica el estado

```bash
docker ps
```

Debe mostrar:

```text
sentinelops          Up
sentinelops-tester   Up
```

#### Observa los logs resultantes

```bash
docker logs -f sentinelops
```

En otra terminal:

```bash
docker logs -f sentinelops-tester
```

#### Prueba desde tu máquina

```bash
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
curl http://localhost:9101/metrics
ssh -T -p 2223 -i data/ssh/client/student_ed25519 student@localhost
```

#### Prueba desde el contenedor tester

```bash
docker exec -it sentinelops-tester sh
```

Dentro, escribe los siguiente:

```sh
curl -k https://sentinelops:9443/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://sentinelops:9443/api/admin/status
curl http://sentinelops:9001/metrics
nc -z sentinelops 2222
```

#### Baja Docker Compose

```bash
docker compose -f docker-compose.demo.yml down
```

O:

```bash
make docker-demo-down
```

### E2E completa

#### Qué hace

La prueba E2E completa valida el flujo principal de SentinelOps:

- dependencias,
- llaves SSH,
- Docker Compose,
- SSH,
- API HTTPS,
- métricas,
- comandos internos,
- túnel local,
- túnel remoto,
- consulta de túneles por API,
- cierre de túneles por API,
- generación de evidencias.

#### Ejecuta

```bash
SSH_PORT=2223 \
METRICS_PORT=9101 \
API_PORT=9444 \
LOCAL_FORWARD_PORT=9901 \
REMOTE_BIND_PORT=10080 \
make e2e-full
```

#### Evidencias

Al finalizar se generan evidencias en:

```text
reports/e2e/latest
```

Ver resultados:

```bash
ls -la reports/e2e/latest
cat reports/e2e/latest/acta-validacion.txt
cat reports/e2e/latest/resultados.json
```

### Kubernetes con Minikube y Helm

#### Qué hace

El despliegue Kubernetes ejecuta SentinelOps dentro de un cluster local Minikube usando Helm.

En Kubernetes se recomienda usar SentinelOps en modo TCP.

Esto evita problemas con `readOnlyRootFilesystem` y generación de host keys SSH.

#### Prepara Minikube

```bash
minikube delete
minikube start --driver=docker
```

#### Usa Docker interno de Minikube

```bash
eval "$(minikube docker-env)"
```

Construye la imagen dentro de Minikube:

```bash
make docker-build
docker images | grep sentinelops
```

No uses `minikube image load` en este flujo.

#### Configuración Helm recomendada

En:

```text
deploy/helm/sentinelops/templates/configmap.yaml
```

debe existir:

```yaml
APP_TRANSPORT: {{ .Values.config.transport | default "tcp" | quote }}
```

En:

```text
deploy/helm/sentinelops/values.yaml
```

dentro de `config`:

```yaml
transport: tcp
```

#### Instala Helm

```bash
kubectl create namespace sentinelops 2>/dev/null || true

helm upgrade --install sentinelops deploy/helm/sentinelops \
  --namespace sentinelops \
  -f deploy/helm/sentinelops/values.yaml \
  -f deploy/helm/sentinelops/values-hardened.yaml \
  --set replicaCount=1
```

#### Verifica Kubernetes

```bash
kubectl get pods -n sentinelops -o wide
kubectl get svc -n sentinelops
kubectl logs -n sentinelops deploy/sentinelops --tail=100
```

Debe aparecer:

```text
READY 1/1
STATUS Running
```

Los logs esperados:

```text
servidor de métricas escuchando addr=:9000
servidor TCP escuchando addr=:2323
API de control escuchando direccion=:9443
```

#### Port forward

En una terminal:

```bash
kubectl port-forward -n sentinelops svc/sentinelops \
  2325:2323 \
  9101:9000 \
  9444:9443
```

Deja esa terminal abierta.

En otra terminal:

```bash
curl -k https://localhost:9444/healthz
curl -k -u 'admin:<APP_CONTROL_API_PASSWORD>' https://localhost:9444/api/admin/status
curl http://localhost:9101/metrics
nc localhost 2325
```

En `nc`:

```text
student
<LAB_PASSWORD_STUDENT>
```

Luego:

```text
help
whoami
status
audit
policy
quit
```

#### Limpia Kubernetes

```bash
helm uninstall sentinelops -n sentinelops
kubectl delete namespace sentinelops
```

Vuelve al Docker normal:

```bash
eval "$(minikube docker-env -u)"
```

### Problemas conocidos y solución

#### Puerto ocupado

Error típico:

```text
bind: address already in use
```

Diagnóstico:

```bash
sudo ss -ltnp | grep -E ':(2222|2223|2324|2325|9001|9101|9443|9444)\b' || true
```

Solución:

```bash
docker rm -f sentinelops-local sentinelops sentinelops-tester 2>/dev/null || true
docker compose -f docker-compose.demo.yml down 2>/dev/null || true
```

#### Clave privada SSH con permisos abiertos

Error:

```text
WARNING: UNPROTECTED PRIVATE KEY FILE
Permissions 0666 are too open
```

Solución:

```bash
chmod 700 data/ssh/client
chmod 600 data/ssh/client/*_ed25519
chmod 644 data/ssh/client/*.pub 2>/dev/null || true
```

#### Docker Compose reinicia contenedores

Diagnóstico:

```bash
docker ps
docker logs --tail=100 sentinelops
docker logs --tail=100 sentinelops-tester
```

Errores comunes:

```text
permission denied
sleep infinity
```

Soluciones:

```bash
chmod 777 data/ssh
chmod -R a+rwX data/controlplane data/state reports
```

Y en `docker-compose.demo.yml`:

```yaml
command: tail -f /dev/null
```

#### Kubernetes CrashLoopBackOff

Diagnóstico:

```bash
kubectl get pods -n sentinelops
kubectl logs -n sentinelops deploy/sentinelops --previous --tail=200
```

Error posible:

```text
read-only file system
open data/ssh/host_ed25519_key
```

Solución recomendada:

- ejecutar Kubernetes en modo TCP
- configurar `APP_TRANSPORT=tcp`
- usar `transport: tcp` en Helm values

#### minikube image load falla

Evita:

```bash
minikube image load sentinelops:local
```

Usa:

```bash
eval "$(minikube docker-env)"
make docker-build
```

### Limpieza antes de subir a GitHub

#### Detiene los servicios

```bash
eval "$(minikube docker-env -u)" 2>/dev/null || true

docker rm -f sentinelops-local sentinelops sentinelops-tester 2>/dev/null || true
docker compose -f docker-compose.demo.yml down 2>/dev/null || true

helm uninstall sentinelops -n sentinelops 2>/dev/null || true
kubectl delete namespace sentinelops 2>/dev/null || true
```

#### Limpia los artefactos generados

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

#### Elimina secretos y datos runtime

```bash
rm -f data/ssh/client/*_ed25519
rm -f data/ssh/client/*_ed25519.pub
rm -f data/ssh/host_ed25519_key
rm -f data/ssh/host_ed25519_key.pub
rm -f data/controlplane/tls.crt
rm -f data/controlplane/tls.key
rm -f data/state/*.json
```

#### Mantiene la estructura de carpetas

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

#### Observa qué entrará al commit

```bash
git status --short
git add -n .
```

No subas:

- claves privadas
- certificados generados
- reportes E2E
- binarios
- cachés
- `rust/input-guard/target`
- datos de estado runtime

### Comandos rápidos

#### Local TCP

```bash
make run-tcp ENV_FILE=env/dev-tcp.env \
  APP_ADDR=:2325 \
  METRICS_ADDR=:9101 \
  APP_CONTROL_API_ADDR=:9444
```

#### Local SSH

```bash
make run-ssh ENV_FILE=env/dev-ssh.env \
  APP_SSH_ADDR=:2223 \
  METRICS_ADDR=:9101 \
  APP_CONTROL_API_ADDR=:9444 \
  APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9101,localhost:9101
```

#### Docker simple

```bash
make docker-build
docker run --rm -d \
  --name sentinelops-local \
  -p 2223:2222 \
  -p 9101:9001 \
  -p 9444:9443 \
  sentinelops:local
```

#### Docker Compose

```bash
SSH_PORT=2223 METRICS_PORT=9101 API_PORT=9444 \
docker compose -f docker-compose.demo.yml up --build -d
```

#### E2E completa

```bash
SSH_PORT=2223 METRICS_PORT=9101 API_PORT=9444 \
LOCAL_FORWARD_PORT=9901 REMOTE_BIND_PORT=10080 \
make e2e-full
```

#### Kubernetes

```bash
minikube start --driver=docker
eval "$(minikube docker-env)"
make docker-build

helm upgrade --install sentinelops deploy/helm/sentinelops \
  --namespace sentinelops \
  --create-namespace \
  -f deploy/helm/sentinelops/values.yaml \
  -f deploy/helm/sentinelops/values-hardened.yaml \
  --set replicaCount=1
```

El proyecto demuestra:

- desarrollo de servidor en Go,
- transporte TCP y SSH,
- seguridad por autenticación, rate limiting y validación,
- integración de Rust, Python y OPA,
- observabilidad con métricas,
- control administrativo por API HTTPS,
- uso de Docker y Docker Compose,
- despliegue Kubernetes con Helm,
- validación E2E con evidencias.


Es **laboratorio técnico reproducible para DevSecOps, seguridad defensiva, observabilidad, policy as code y despliegue cloud native.**

### Fase 3 - Observabilidad con OpenTelemetry

#### Objetivo

La fase 3 agrega trazas distribuidas opcionales con OpenTelemetry para seguir el flujo de una solicitud entre transporte TCP, transporte SSH, API de control, validación de entradas, comandos y túneles.

#### Variables de entorno

OpenTelemetry queda deshabilitado por defecto para que Jaeger no sea una dependencia obligatoria durante pruebas unitarias, demos simples o ejecución local básica.

```bash
OTEL_TRACES_ENABLED=false
OTEL_EXPORTER_TYPE=stdout
OTEL_EXPORTER_ENDPOINT=localhost:4317
OTEL_EXPORTER_INSECURE=true
OTEL_SAMPLE_RATE=1.0
```

Para ver el servicio `sentinelops` en Jaeger UI, las trazas deben activarse explícitamente:

```bash
OTEL_TRACES_ENABLED=true
OTEL_EXPORTER_TYPE=otlp-grpc
OTEL_EXPORTER_ENDPOINT=localhost:4317
OTEL_EXPORTER_INSECURE=true
OTEL_SAMPLE_RATE=1.0
```

Exportadores soportados:

- `stdout`: imprime spans en consola para desarrollo local.
- `otlp-grpc`: envía trazas por OTLP gRPC.
- `otlp-http`: envía trazas por OTLP HTTP.
- `jaeger`: alias compatible con Jaeger usando OTLP gRPC.

#### Ejecutar Jaeger local

```bash
make run-jaeger
```

La interfaz queda disponible en:

```text
http://localhost:16686
```

#### Ejecutar SentinelOps con trazas

```bash
source .env.local
make run-ssh-telemetry
```

El target `make run-ssh-telemetry` fuerza `OTEL_TRACES_ENABLED=true` y `OTEL_EXPORTER_TYPE=otlp-grpc`, incluso si `.env.local` conserva `OTEL_TRACES_ENABLED=false`.

Después de generar tráfico contra la API de control, Jaeger debe listar el servicio:

```bash
curl -s http://localhost:16686/api/services
```

En la interfaz web se debe usar el panel izquierdo:

```text
Service -> sentinelops
Operation -> control_api.request
Find Traces
```

No se debe escribir `sentinelops` en el cuadro superior derecho de Jaeger, porque ese cuadro espera un identificador de traza.

#### Stack completo con observabilidad

```bash
source .env.local
docker compose -f docker-compose.observability.yml up --build -d
```

También se puede usar:

```bash
make docker-observability-up
```

Servicios expuestos:

```text
Jaeger UI: http://localhost:16686
Prometheus: http://localhost:9090
SentinelOps SSH: localhost:2223
SentinelOps métricas: http://localhost:9101/metrics
SentinelOps API: https://localhost:9444/healthz
```

#### Spans principales

```text
tcp.session
ssh.session
auth.authenticate
security.validate_input
command.execute
ssh.forwarding
control_api.request
```

#### Correlation IDs

La API de control acepta y devuelve el header:

```text
X-Correlation-ID
```

También devuelve:

```text
X-Trace-ID
```

Esto permite relacionar una solicitud HTTP con su traza correspondiente en Jaeger.

### Fase 4 - OPA como sidecar runtime

La fase 4 permite evaluar políticas OPA en runtime mediante HTTP contra un sidecar OPA. El modo anterior con binario local se mantiene para CI, auditorías offline y laboratorios sin Docker.

#### Modos de política

```text
OPA_POLICY_MODE=exec   usa el binario OPA local
OPA_POLICY_MODE=http   consulta un OPA sidecar por HTTP
```

El modo por defecto sigue siendo:

```env
OPA_POLICY_MODE=exec
```

Esto evita que OPA como servicio externo sea obligatorio durante pruebas unitarias o ejecución local básica.

#### Ejecutar OPA sidecar local

Primero genera secretos locales si no existen:

```bash
make generate-secrets
```

Luego levanta SentinelOps con OPA como sidecar simulado en Docker Compose:

```bash
source .env.local
make run-opa-sidecar
```

El target pasa `HOST_UID` y `HOST_GID` al build y al runtime del contenedor. Esto permite que SentinelOps escriba claves SSH, certificados TLS y estado dentro de `./data` cuando se usa bind mount local.

Si el contenedor `sentinelops-opa-demo` queda en `Restarting`, revisa permisos y logs:

```bash
docker compose -f docker-compose.opa.yml ps
docker compose -f docker-compose.opa.yml logs --tail=200 sentinelops
```

Servicios expuestos:

```text
OPA sidecar: http://localhost:8181
SentinelOps SSH: localhost:2224
SentinelOps métricas: http://localhost:9102/metrics
SentinelOps API: https://localhost:9445/healthz
```

#### Consultar OPA directamente

```bash
curl -s http://localhost:8181/health
```

Validar una decisión de política:

```bash
curl -s -X POST http://localhost:8181/v1/data/kubernetes/security/deny \
  -H 'Content-Type: application/json' \
  -d '{"input":{"kind":"Deployment","spec":{"template":{"spec":{"containers":[{"name":"server","image":"sentinelops:latest","securityContext":{"privileged":true,"runAsNonRoot":false,"allowPrivilegeEscalation":true,"readOnlyRootFilesystem":false}}]}}}}}'
```

#### Ejecutar SentinelOps local contra OPA HTTP

Si OPA ya está corriendo en `localhost:8181`, también puedes ejecutar SentinelOps fuera de Docker:

```bash
source .env.local
OPA_POLICY_MODE=http \
OPA_POLICY_URL=http://localhost:8181 \
make run-ssh
```

#### Volver al modo binario

```bash
OPA_POLICY_MODE=exec make run-ssh
```

#### Limpieza

```bash
make stop-opa-sidecar
```

### Fase 5 - OpenAPI y versionado de API

La fase 5 agrega una superficie HTTP versionada para la API administrativa y documentación OpenAPI sin reemplazar los endpoints legados.

#### Endpoints de health check

Los probes quedan separados para Kubernetes y validación local:

| Método | Endpoint | Uso | Auth |
|---|---|---|---|
| `GET` | `/healthz` | Compatibilidad legado | No |
| `GET` | `/healthz/live` | Liveness probe | No |
| `GET` | `/healthz/ready` | Readiness probe | No |
| `GET` | `/healthz/startup` | Startup probe | No |

#### API administrativa v1

| Método | Endpoint | Descripción | Auth |
|---|---|---|---|
| `GET` | `/api/v1/admin/status` | Estado general del sistema | Basic |
| `GET` | `/api/v1/admin/sessions` | Sesiones activas | Basic |
| `GET` | `/api/v1/admin/tunnels` | Túneles activos | Basic |
| `POST` | `/api/v1/admin/tunnels/{id}/close` | Cerrar túnel activo | Basic |

Los endpoints legados bajo `/api/admin/...` se mantienen para compatibilidad.

#### Documentación OpenAPI

La especificación OpenAPI 3.0 está disponible en:

```text
/api/v1/docs/swagger.json
/api/v1/docs/openapi.json
```

La documentación ligera se expone en:

```text
/api/v1/docs/swagger/
```

También se versionan archivos estáticos para revisión en repositorio:

```text
docs/openapi.json
docs/swagger.json
docs/openapi.yaml
```

#### Regenerar documentación

```bash
make docs
make docs-check
```

#### Probar API v1 local

Con SentinelOps ejecutándose y `.env.local` cargado:

```bash
source .env.local
curl -k https://localhost:9443/healthz/live
curl -k https://localhost:9443/healthz/ready
curl -k https://localhost:9443/healthz/startup
curl -k https://localhost:9443/api/v1/docs/swagger.json
curl -k -u "$APP_CONTROL_API_USER:$APP_CONTROL_API_PASSWORD" \
  https://localhost:9443/api/v1/admin/status
```

También se puede usar:

```bash
make api-smoke
```

Para el stack con OPA sidecar, usa el puerto externo `9445`:

```bash
API_URL=https://localhost:9445 make api-smoke
```

### Fase 6 - Validador Rust gRPC

#### Objetivo

La fase 6 agrega una ruta de evolución para ejecutar `input-guard` como servicio gRPC. El modo por defecto sigue siendo `binary`, por lo que `make test`, `make rust-test` y el flujo local existente no dependen de gRPC.

#### Arquitectura

```text
SentinelOps Go -> gRPC -> input-guard-grpc Rust
```

El servicio gRPC escucha por defecto en:

```text
0.0.0.0:50051
```

La API Go conserva el validador estático y el validador externo por binario como fallback. El modo gRPC se activa de forma explícita con:

```env
VALIDATOR_MODE=grpc
VALIDATOR_GRPC_ADDR=localhost:50051
VALIDATOR_GRPC_TIMEOUT=2s
VALIDATOR_GRPC_FAIL_OPEN=false
```

#### Proto

El contrato se define en:

```text
proto/validator/v1/validator.proto
```

El servicio principal es:

```text
validator.v1.Validator
```

con estos métodos:

```text
ValidateInput
Health
```

#### Construir el servidor Rust gRPC

```bash
make validator-grpc-build
```

#### Probar el servidor Rust gRPC

```bash
make validator-grpc-test
```

#### Levantar stack Docker con gRPC

```bash
source .env.local
make validator-grpc-up
```

En otra terminal:

```bash
source .env.local
API_URL=https://localhost:9446 make validator-grpc-smoke
```

Luego detener:

```bash
make validator-grpc-down
```

#### Generar código Go desde proto

La generación de cliente Go es opcional y requiere herramientas locales:

```bash
make proto-tools
make proto-go
```

El código generado queda en:

```text
gen/go
```

Ese directorio se considera artefacto generado y no se versiona.

#### Compatibilidad

Esta fase no elimina el binario local:

```text
rust/input-guard/target/release/input-guard
```

El modo `binary` sigue siendo el recomendado para pruebas rápidas y CI básico. El modo `grpc` queda disponible para validar el patrón sidecar en Docker y Kubernetes.

#### Cliente Go gRPC opcional

El cliente Go para gRPC queda protegido con el build tag `grpcvalidator`. Esto evita que el flujo normal descargue dependencias gRPC o requiera código generado.

Para compilar el cliente gRPC en una validación extendida:

```bash
make proto-tools
make proto-go
go test -tags grpcvalidator ./internal/security
```

Si Go solicita módulos adicionales, ejecuta:

```bash
go mod tidy
```

Solo versiona `go.mod` y `go.sum` si decides hacer obligatorio el modo gRPC en una fase posterior. En esta fase, `gen/go` sigue siendo artefacto generado.

### Fase 7: CI/CD DevSecOps

#### Objetivo

La fase 7 consolida la validación automática del proyecto. La pipeline cubre calidad Go, calidad Rust, contrato OpenAPI, contrato gRPC, políticas OPA, Helm, seguridad básica y construcción de imágenes Docker.

#### Workflows agregados

| Workflow     | Archivo                              | Uso                                      |
| ------------ | ------------------------------------ | ---------------------------------------- |
| CI DevSecOps | `.github/workflows/ci-devsecops.yml` | Validación principal de PR y `main`      |
| CodeQL       | `.github/workflows/codeql.yml`       | Análisis estático para Go                |
| Release      | `.github/workflows/release.yml`      | Creación de releases al publicar tags    |
| Dependabot   | `.github/dependabot.yml`             | Actualización controlada de dependencias |

#### Comandos locales

```bash
make ci-check
make ci-openapi
make ci-proto
make ci-security
make ci-clean
```

#### Artefactos no versionados

Estos archivos y carpetas se generan localmente, pero no deben subirse al repositorio:

```text
gen/go/
policies/bundle/
rust/input-guard/target/
rust/input-guard-grpc/target/
coverage.out
coverage.html
```

#### Release de fase 7

Después de mergear la fase 7 en `main`, se puede crear el tag:

```bash
git tag -a v0.7.0-fase7-ci-cd-devsecops -m "fase 7: CI/CD DevSecOps"
git push origin v0.7.0-fase7-ci-cd-devsecops
```

### Fase 8: observabilidad operacional y evidencia runtime

#### Objetivo

La fase 8 consolida la operación observable de SentinelOps. Agrega dashboards, configuración reproducible de Prometheus y Grafana, scripts de smoke operacional, evidencia de runtime y runbooks.

#### Componentes agregados

| Componente | Archivo |
|---|---|
| Prometheus runtime | `deploy/observability/prometheus.yml` |
| Grafana provisioning | `deploy/observability/grafana/provisioning/` |
| Dashboard runtime | `deploy/observability/grafana/dashboards/sentinelops-runtime.json` |
| Smoke de observabilidad | `scripts/observability-smoke.sh` |
| Evidencia runtime | `scripts/runtime-evidence.sh` |
| Runbook | `docs/runbooks/observabilidad-runtime.md` |
| Guía de evidencia | `docs/evidence/runtime-evidence.md` |

#### Comandos

```bash
make observability-up
make observability-smoke
make runtime-evidence
make observability-down
```

#### Endpoints locales

| Servicio | URL |
|---|---|
| SentinelOps metrics | `http://localhost:9101/metrics` |
| SentinelOps Control Plane | `https://localhost:9444` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` |
| Jaeger | `http://localhost:16686` |

#### Evidencia generada

```text
reports/runtime/<timestamp>/
```

El reporte incluye métricas, health checks, estado autenticado si hay contraseña configurada, targets de Prometheus, estado de Grafana, servicios de Jaeger y estado de contenedores.
### Fase 9: persistencia PostgreSQL y Redis

#### Objetivo

La fase 9 agrega una abstracción de almacenamiento para sesiones, túneles, auditoría y rate limiting. El modo por defecto sigue siendo `memory`, pero el proyecto queda preparado para usar PostgreSQL como almacenamiento durable y Redis como cache operativo.

#### Componentes agregados

| Componente | Archivo |
|---|---|
| Interfaz Store | `internal/store/store.go` |
| Store en memoria | `internal/store/memory.go` |
| Store PostgreSQL | `internal/store/postgres.go` |
| Store Redis | `internal/store/redis.go` |
| Stack local | `docker-compose.storage.yml` |
| Migraciones | `migrations/` |
| Runbook | `docs/runbooks/persistencia-storage.md` |

#### Comandos

    make generate-secrets
    make storage-up
    source .env.local
    make storage-smoke
    make storage-test
    make storage-down

#### Limpieza

    make storage-clean
