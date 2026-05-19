### SentinelOps

**SentinelOps es software real de laboratorio**, diseñado para enseñanza, evaluación técnica y experimentación controlada en DevSecOps. Su objetivo es mostrar cómo un servicio remoto simple puede evolucionar hacia una solución con:

- transporte SSH cifrado,
- autenticación por contraseña y clave pública,
- validación defensiva de entradas,
- auditoría automatizada,
- túneles SSH controlados por política,
- métricas Prometheus,
- API HTTPS administrativa,
- policy-as-code con OPA/Rego,
- despliegue reproducible con Docker, Helm y Kubernetes.

SentinelOps **no pretende reemplazar OpenSSH ni ofrecer acceso remoto general para producción**. Está orientado a laboratorio, defensa académica y validación técnica.

#### Credenciales de laboratorio

Las credenciales por defecto (`admin` / `admin123!`) son únicamente para entorno académico/local. Si el proyecto se reutiliza fuera del laboratorio, deben reemplazarse por variables de entorno o secretos gestionados por la plataforma de despliegue.

### Capacidades principales

#### Transporte y acceso remoto
- servidor TCP heredado para comparación,
- servidor SSH con cifrado de canal real,
- autenticación por contraseña,
- autenticación por clave pública,
- `authorized_keys` por usuario,
- host key persistente,
- cliente SSH mínimo en Go,
- compatibilidad con el cliente `ssh` del sistema.

#### Seguridad y validación
- validación rápida en Go,
- validador externo en Rust,
- auditoría externa en Python,
- policy-as-code con OPA/Rego,
- perfiles `hardened` e `insecure`.

#### Observabilidad y administración
- métricas Prometheus,
- sesiones activas en memoria,
- túneles activos en memoria,
- API HTTPS administrativa con TLS y Basic Auth,
- listado y cierre de túneles desde shell y vía API,
- snapshots JSON opcionales de sesiones y túneles,
- rate limiting de login por usuario y origen.

#### Fundamento técnico

SentinelOps aprovecha capacidades reales del ecosistema Go:

- `crypto/tls` para la API HTTPS,
- `crypto/ed25519` para host keys y material criptográfico,
- `crypto/x509` para certificados,
- `crypto/rand` para generación segura,
- `golang.org/x/crypto/ssh` para transporte SSH,
- **goroutines** para manejar múltiples sesiones y túneles concurrentes.

Además integra:
- **Python** para auditoría,
- **Rust** para validación defensiva,
- **Rego/OPA** para validación de despliegues.

#### Arquitectura

```text
Cliente SSH / Cliente Go / Cliente TCP
                │
                ▼
          SentinelOps Server
   ├── transporte TCP o SSH
   ├── autenticación y sesiones
   ├── shell de laboratorio
   ├── auditoría y policy-as-code
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
| `internal/session` | sesiones activas y registro |
| `internal/forwarding` | política, registro y control de túneles |
| `internal/controlplane/httpapi` | API HTTPS administrativa |
| `internal/metrics` | endpoint Prometheus |
| `tools/audit` | auditoría Python |
| `rust/input-guard` | validador externo |
| `policies/kubernetes` | reglas Rego |

##### Requisitos

Instala como mínimo:
- **Go 1.25+**
- Rust / Cargo
- Python 3.11+
- OPA
- Helm 3
- Docker
- Make
- `ssh`, `ssh-keygen`, `nc`, `curl`
- opcionalmente Minikube

##### Ubuntu / Debian

```bash
sudo apt update
sudo apt install -y make curl docker.io netcat-openbsd openssh-client
```

#### Entornos soportados

SentinelOps incorpora una estrategia explícita de entornos para que el mismo software pueda ejecutarse de forma repetible en desarrollo local, contenedor y Kubernetes.

### Archivos incluidos
- `env/dev-tcp.env`
- `env/dev-ssh.env`
- `env/container-ssh.env`
- `env/minikube-hardened.env`

##### Uso con el Makefile

```bash
make run-ssh ENV_FILE=env/dev-ssh.env
make run-tcp ENV_FILE=env/dev-tcp.env
```

##### Uso manual

```bash
set -a
source env/dev-ssh.env
set +a
make run-ssh
```

##### Qué representa cada entorno

| Archivo | Uso recomendado |
|---|---|
| `env/dev-tcp.env` | desarrollo local en modo TCP |
| `env/dev-ssh.env` | desarrollo local en modo SSH |
| `env/container-ssh.env` | referencia para ejecución Docker/Compose |
| `env/minikube-hardened.env` | referencia para despliegue endurecido en Kubernetes |

