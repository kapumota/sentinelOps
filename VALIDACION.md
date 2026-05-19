### Validación local de SentinelOps v2.4.1

Validaciones ejecutadas en este entorno:

- `gofmt -w cmd internal`: completado.
- `go test ./internal/auth ./internal/config ./internal/persistence ./internal/session ./internal/forwarding`: `pass` usando `GOTOOLCHAIN=local` y un `go.mod` temporal con `go 1.23.0` solo para paquetes sin dependencias externas.
- `go test ./internal/controlplane/httpapi`: `pass` usando `GOTOOLCHAIN=local` y un `go.mod` temporal con `go 1.23.0`; esta prueba levanta HTTPS real con certificado autofirmado temporal.
- `python3 tools/audit/audit.py --profile hardened --project-root .`: `pass`, 0 findings.
- `python3 tools/audit/audit.py --profile insecure --project-root .`: `fail`, 2 findings esperados por perfil demostrativo.
- `bash -n scripts/*.sh demo.sh`: completado.

Validaciones no ejecutadas por limitaciones del entorno actual:

- `go test ./...`: el proyecto requiere Go 1.25.0 y el entorno local solo tiene Go 1.23.2 sin acceso a `proxy.golang.org` para descargar toolchain/módulos.
- Pruebas de integración que importan `github.com/prometheus/client_golang` o `golang.org/x/crypto/ssh`: no se pudieron compilar aquí porque las dependencias externas no están descargadas y no hay acceso al proxy de Go.
- `cargo test`: Rust/Cargo no está instalado en el entorno local.
- `helm lint` / render Helm: Helm no está instalado en el entorno local.
- Docker E2E: Docker/Compose no se ejecutó aquí.

Recomendación antes de entregar:

```bash
go mod tidy
go mod verify
gofmt -w cmd internal
go vet ./...
go test ./... -v
cargo test --manifest-path rust/input-guard/Cargo.toml
python3 tools/audit/audit.py --profile hardened --project-root .
bash scripts/ci-policy-check.sh hardened pass
bash scripts/ci-helm-validate.sh hardened
docker build -t sentinelops:2.4.1 .
bash scripts/test-e2e-full.sh
```
