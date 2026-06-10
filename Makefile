APP_NAME=sentinelops
BINARY=bin/sentinelops
CLIENT_BINARY=bin/sentinelops-client
IMAGE=sentinelops:local
RUST_MANIFEST=rust/input-guard/Cargo.toml
RUST_BINARY=rust/input-guard/target/release/input-guard
HELM_CHART=deploy/helm/sentinelops

ENV_FILE ?= .env.local
ifneq ($(strip $(ENV_FILE)),)
ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export
endif
endif

APP_ENV ?= dev
PROFILE ?= hardened
APP_TRANSPORT ?= tcp
APP_ADDR ?= :2324
APP_SSH_ADDR ?= :2222
METRICS_ADDR ?= :9001
APP_CONTROL_API_ENABLED ?= true
APP_CONTROL_API_ADDR ?= :9443
APP_CONTROL_API_USER ?= admin
APP_CONTROL_API_PASSWORD ?=
OTEL_TRACES_ENABLED ?= false
OTEL_EXPORTER_TYPE ?= stdout
OTEL_EXPORTER_ENDPOINT ?= localhost:4317
OTEL_EXPORTER_INSECURE ?= true
OTEL_SAMPLE_RATE ?= 1.0
OPA_POLICY_MODE ?= exec
OPA_POLICY_URL ?= http://localhost:8181
OPA_POLICY_TIMEOUT ?= 2s
OPA_POLICY_CACHE_ENABLED ?= true
OPA_POLICY_CACHE_TTL ?= 30s
LAB_PASSWORD_STUDENT ?=
LAB_PASSWORD_TEACHER ?=
LAB_PASSWORD_AUDITOR ?=
LAB_PASSWORD_ADMIN ?=
APP_AUTH_RATE_LIMIT_ENABLED ?= true
APP_AUTH_RATE_LIMIT_MAX_FAILURES ?= 5
APP_AUTH_RATE_LIMIT_WINDOW ?= 1m
APP_AUTH_RATE_LIMIT_LOCKOUT ?= 1m
APP_STATE_PERSISTENCE_ENABLED ?= false
APP_STATE_PERSISTENCE_DIR ?= data/state
APP_STATE_SESSIONS_PATH ?= data/state/sessions.json
APP_STATE_TUNNELS_PATH ?= data/state/tunnels.json
APP_SSH_FORWARD_ALLOWLIST ?= 127.0.0.1:9001,localhost:9001
APP_SSH_LOCAL_ALLOWED_ROLES ?= student,teacher,auditor,admin
APP_SSH_REMOTE_FORWARD_ENABLED ?= false
APP_SSH_REMOTE_BIND_ALLOWLIST ?= 127.0.0.1:10080,127.0.0.1:10443
APP_SSH_REMOTE_ALLOWED_ROLES ?= teacher,auditor,admin

.PHONY: help build build-client run run-tcp run-ssh run-ssh-telemetry run-jaeger stop-jaeger run-with-telemetry run-opa-sidecar stop-opa-sidecar opa-build opa clean opa-test opa-run opa-ci ssh-lab-setup demo docker-demo curl-examples test test-unit test-integration test-race test-coverage test-all test-e2e-containers rust-test rust-build fmt vet clean check audit policy helm-lint helm-template helm-install bootstrap setup-dev install-dev-tools generate-secrets check-secrets e2e e2e-full docker-build docker-run docker-run-tcp docker-run-ssh docker-demo-up docker-demo-down docker-demo-logs docker-observability-up docker-observability-down docker-observability-logs docker-stop deploy-local cleanup

help:
	@echo "Targets disponibles:"
	@echo "  make build            - Compila servidor y cliente"
	@echo "  make build-client     - Compila cliente SSH"
	@echo "  make run              - Ejecuta el servidor local (por defecto TCP)"
	@echo "  make run-tcp          - Ejecuta el servidor en modo TCP"
	@echo "  make run-ssh          - Ejecuta el servidor en modo SSH"
	@echo "  make run-ssh-telemetry - Ejecuta SSH con OpenTelemetry local"
	@echo "  make run-jaeger       - Levanta Jaeger local"
	@echo "  make run-with-telemetry - Levanta SentinelOps, Jaeger y Prometheus"
	@echo "  make run-opa-sidecar - Levanta SentinelOps con OPA sidecar HTTP"
	@echo "  make opa-test         - Ejecuta pruebas Rego"
	@echo "  make ssh-lab-setup    - Genera llave de laboratorio y authorized_keys"
	@echo "  make generate-secrets - Genera .env.local con credenciales aleatorias"
	@echo "  make curl-examples    - Ejecuta ejemplos curl contra la API HTTPS"
	@echo "  make demo             - Ejecuta una demostración guiada end-to-end"
	@echo "  make test             - Ejecuta pruebas Go"
	@echo "  make test-unit        - Ejecuta pruebas unitarias rápidas"
	@echo "  make test-integration - Ejecuta pruebas de integración con testcontainers"
	@echo "  make test-race        - Ejecuta pruebas Go con detector de carreras"
	@echo "  make test-coverage    - Genera reporte de cobertura Go"
	@echo "  make test-all         - Ejecuta pruebas unitarias, integración y race detector"
	@echo "  make test-e2e-containers - Ejecuta E2E con imagen Docker y testcontainers"
	@echo "  make rust-test        - Ejecuta pruebas Rust"
	@echo "  make rust-build       - Compila el binario Rust"
	@echo "  make fmt              - Formatea el código Go"
	@echo "  make vet              - Ejecuta go vet"
	@echo "  make check            - Ejecuta validaciones completas"
	@echo "  make audit            - Ejecuta auditoría externa Python"
	@echo "  make policy           - Evalúa políticas Rego con OPA"
	@echo "  make helm-lint        - Valida el chart Helm"
	@echo "  make helm-template    - Renderiza el chart Helm"
	@echo "  make helm-install     - Despliega el chart Helm"
	@echo "  make bootstrap        - Prepara entorno local"
	@echo "  make setup-dev        - Genera secretos y llaves para desarrollo"
	@echo "  make install-dev-tools - Instala dependencias de desarrollo en Ubuntu"
	@echo "  make e2e              - Ejecuta prueba end-to-end local TCP"
	@echo "  make e2e-full         - Ejecuta validación E2E Docker con evidencias"
	@echo "  make docker-build     - Construye imagen Docker"
	@echo "  make docker-run-tcp   - Ejecuta contenedor local en modo TCP"
	@echo "  make docker-run-ssh   - Ejecuta contenedor local en modo SSH"
	@echo "  make docker-observability-up - Levanta stack con Jaeger"
	@echo "  make deploy-local     - Despliega contenedor en background"
	@echo "  make docker-stop      - Detiene contenedor local"
	@echo "  make cleanup          - Limpia reportes y contenedor"
	@echo "  make ... ENV_FILE=env/dev-ssh.env  - Carga variables desde archivo de entorno"

build: rust-build
	@mkdir -p bin
	go build -o $(BINARY) ./cmd/server
	go build -o $(CLIENT_BINARY) ./cmd/client

build-client:
	@mkdir -p bin
	go build -o $(CLIENT_BINARY) ./cmd/client

run: run-tcp

run-tcp: rust-build
	APP_ENV=$(APP_ENV) \
	APP_PROFILE=$(PROFILE) \
	APP_ADDR=$(APP_ADDR) \
	METRICS_ADDR=$(METRICS_ADDR) \
	APP_CONTROL_API_ENABLED=$(APP_CONTROL_API_ENABLED) \
	APP_CONTROL_API_ADDR=$(APP_CONTROL_API_ADDR) \
	APP_CONTROL_API_USER=$(APP_CONTROL_API_USER) \
	APP_CONTROL_API_PASSWORD=$(APP_CONTROL_API_PASSWORD) \
	APP_AUTH_RATE_LIMIT_ENABLED=$(APP_AUTH_RATE_LIMIT_ENABLED) \
	APP_AUTH_RATE_LIMIT_MAX_FAILURES=$(APP_AUTH_RATE_LIMIT_MAX_FAILURES) \
	APP_AUTH_RATE_LIMIT_WINDOW=$(APP_AUTH_RATE_LIMIT_WINDOW) \
	APP_AUTH_RATE_LIMIT_LOCKOUT=$(APP_AUTH_RATE_LIMIT_LOCKOUT) \
	APP_STATE_PERSISTENCE_ENABLED=$(APP_STATE_PERSISTENCE_ENABLED) \
	APP_STATE_PERSISTENCE_DIR=$(APP_STATE_PERSISTENCE_DIR) \
	APP_STATE_SESSIONS_PATH=$(APP_STATE_SESSIONS_PATH) \
	APP_STATE_TUNNELS_PATH=$(APP_STATE_TUNNELS_PATH) \
	EXTERNAL_VALIDATOR_ENABLED=true \
	EXTERNAL_VALIDATOR_BINARY=$(RUST_BINARY) \
	EXTERNAL_VALIDATOR_FAIL_OPEN=false \
	OPA_POLICY_ENABLED=true \
	OPA_POLICY_MODE=$(OPA_POLICY_MODE) \
	OPA_POLICY_URL=$(OPA_POLICY_URL) \
	OPA_POLICY_TIMEOUT=$(OPA_POLICY_TIMEOUT) \
	OPA_POLICY_CACHE_ENABLED=$(OPA_POLICY_CACHE_ENABLED) \
	OPA_POLICY_CACHE_TTL=$(OPA_POLICY_CACHE_TTL) \
	OPA_BINARY=opa \
	OPA_POLICY_DIR=policies/kubernetes \
	OTEL_TRACES_ENABLED=$(OTEL_TRACES_ENABLED) \
	OTEL_EXPORTER_TYPE=$(OTEL_EXPORTER_TYPE) \
	OTEL_EXPORTER_ENDPOINT=$(OTEL_EXPORTER_ENDPOINT) \
	OTEL_EXPORTER_INSECURE=$(OTEL_EXPORTER_INSECURE) \
	OTEL_SAMPLE_RATE=$(OTEL_SAMPLE_RATE) \
	go run ./cmd/server

run-ssh: rust-build
	APP_ENV=$(APP_ENV) \
	APP_PROFILE=$(PROFILE) \
	METRICS_ADDR=$(METRICS_ADDR) \
	APP_CONTROL_API_ENABLED=$(APP_CONTROL_API_ENABLED) \
	APP_CONTROL_API_ADDR=$(APP_CONTROL_API_ADDR) \
	APP_CONTROL_API_USER=$(APP_CONTROL_API_USER) \
	APP_CONTROL_API_PASSWORD=$(APP_CONTROL_API_PASSWORD) \
	APP_AUTH_RATE_LIMIT_ENABLED=$(APP_AUTH_RATE_LIMIT_ENABLED) \
	APP_AUTH_RATE_LIMIT_MAX_FAILURES=$(APP_AUTH_RATE_LIMIT_MAX_FAILURES) \
	APP_AUTH_RATE_LIMIT_WINDOW=$(APP_AUTH_RATE_LIMIT_WINDOW) \
	APP_AUTH_RATE_LIMIT_LOCKOUT=$(APP_AUTH_RATE_LIMIT_LOCKOUT) \
	APP_STATE_PERSISTENCE_ENABLED=$(APP_STATE_PERSISTENCE_ENABLED) \
	APP_STATE_PERSISTENCE_DIR=$(APP_STATE_PERSISTENCE_DIR) \
	APP_STATE_SESSIONS_PATH=$(APP_STATE_SESSIONS_PATH) \
	APP_STATE_TUNNELS_PATH=$(APP_STATE_TUNNELS_PATH) \
	EXTERNAL_VALIDATOR_ENABLED=true \
	EXTERNAL_VALIDATOR_BINARY=$(RUST_BINARY) \
	EXTERNAL_VALIDATOR_FAIL_OPEN=false \
	OPA_POLICY_ENABLED=true \
	OPA_POLICY_MODE=$(OPA_POLICY_MODE) \
	OPA_POLICY_URL=$(OPA_POLICY_URL) \
	OPA_POLICY_TIMEOUT=$(OPA_POLICY_TIMEOUT) \
	OPA_POLICY_CACHE_ENABLED=$(OPA_POLICY_CACHE_ENABLED) \
	OPA_POLICY_CACHE_TTL=$(OPA_POLICY_CACHE_TTL) \
	OPA_BINARY=opa \
	OPA_POLICY_DIR=policies/kubernetes \
	OTEL_TRACES_ENABLED=$(OTEL_TRACES_ENABLED) \
	OTEL_EXPORTER_TYPE=$(OTEL_EXPORTER_TYPE) \
	OTEL_EXPORTER_ENDPOINT=$(OTEL_EXPORTER_ENDPOINT) \
	OTEL_EXPORTER_INSECURE=$(OTEL_EXPORTER_INSECURE) \
	OTEL_SAMPLE_RATE=$(OTEL_SAMPLE_RATE) \
	APP_TRANSPORT=ssh \
	APP_SSH_ADDR=$(APP_SSH_ADDR) \
	APP_SSH_LOCAL_FORWARD_ENABLED=true \
	APP_SSH_FORWARD_ALLOWLIST=$(APP_SSH_FORWARD_ALLOWLIST) \
	APP_SSH_LOCAL_ALLOWED_ROLES=$(APP_SSH_LOCAL_ALLOWED_ROLES) \
	APP_SSH_REMOTE_FORWARD_ENABLED=$(APP_SSH_REMOTE_FORWARD_ENABLED) \
	APP_SSH_REMOTE_BIND_ALLOWLIST=$(APP_SSH_REMOTE_BIND_ALLOWLIST) \
	APP_SSH_REMOTE_ALLOWED_ROLES=$(APP_SSH_REMOTE_ALLOWED_ROLES) \
	OTEL_TRACES_ENABLED=$(OTEL_TRACES_ENABLED) \
	OTEL_EXPORTER_TYPE=$(OTEL_EXPORTER_TYPE) \
	OTEL_EXPORTER_ENDPOINT=$(OTEL_EXPORTER_ENDPOINT) \
	OTEL_EXPORTER_INSECURE=$(OTEL_EXPORTER_INSECURE) \
	OTEL_SAMPLE_RATE=$(OTEL_SAMPLE_RATE) \
	go run ./cmd/server


run-jaeger:
	docker rm -f sentinelops-jaeger >/dev/null 2>&1 || true
	docker run -d --name sentinelops-jaeger \
		-p 16686:16686 \
		-p 4317:4317 \
		-p 4318:4318 \
		-e COLLECTOR_OTLP_ENABLED=true \
		jaegertracing/all-in-one:1.50
	@echo "Jaeger disponible en http://localhost:16686"

stop-jaeger:
	docker rm -f sentinelops-jaeger >/dev/null 2>&1 || true

run-ssh-telemetry:
	$(MAKE) run-ssh \
		OTEL_TRACES_ENABLED=true \
		OTEL_EXPORTER_TYPE=otlp-grpc \
		OTEL_EXPORTER_ENDPOINT=localhost:4317 \
		OTEL_EXPORTER_INSECURE=true \
		OTEL_SAMPLE_RATE=1.0

run-with-telemetry: docker-observability-up

run-opa-sidecar:
	HOST_UID=$$(id -u) HOST_GID=$$(id -g) docker compose -f docker-compose.opa.yml up --build -d
	@echo "OPA sidecar: http://localhost:8181"
	@echo "SentinelOps API: https://localhost:9445/healthz"

stop-opa-sidecar:
	HOST_UID=$$(id -u) HOST_GID=$$(id -g) docker compose -f docker-compose.opa.yml down

opa-test:
	opa test -v policies/kubernetes

opa-build:
	@mkdir -p policies/bundle
	opa build -b policies/kubernetes -o policies/bundle/sentinelops.tar.gz
	@echo "Bundle OPA generado en policies/bundle/sentinelops.tar.gz"

opa-run: opa-build
	opa run --server --addr=localhost:8181 --log-level=info --bundle=policies/bundle/sentinelops.tar.gz

opa-ci: opa-test opa-build
	@echo "Políticas OPA validadas"

opa-clean:
	rm -rf policies/bundle

ssh-lab-setup:
	USER_NAME=$${USER_NAME:-student}; bash scripts/setup-ssh-lab.sh "$$USER_NAME"

curl-examples:
	API_URL=$${API_URL:-https://localhost:9443} API_USER=$${API_USER:-admin} API_PASSWORD=$${API_PASSWORD:-$${APP_CONTROL_API_PASSWORD:-}} bash scripts/control-api-curl-examples.sh

demo:
	bash ./demo.sh

docker-demo:
	bash scripts/docker-demo.sh

docker-demo-up:
	docker compose -f docker-compose.demo.yml up --build -d

docker-demo-down:
	docker compose -f docker-compose.demo.yml down

docker-demo-logs:
	docker logs sentinelops

docker-observability-up:
	docker compose -f docker-compose.observability.yml up --build -d
	@echo "Jaeger UI: http://localhost:16686"
	@echo "Prometheus: http://localhost:9090"

docker-observability-down:
	docker compose -f docker-compose.observability.yml down

docker-observability-logs:
	docker compose -f docker-compose.observability.yml logs -f

test:
	go test ./...

test-unit:
	go test -short -v ./internal/... ./cmd/...

test-integration:
	TESTCONTAINERS_RYUK_DISABLED=true go test -tags=containers -v -run Integration ./internal/...
	cd tests/integration && TESTCONTAINERS_RYUK_DISABLED=true go test -tags=containers -v -timeout 3m .

test-race:
	go test -race -v ./internal/...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Reporte de cobertura generado en coverage.html"

test-all: test-unit test-integration test-race
	@echo "Todas las pruebas configuradas pasaron."

test-e2e-containers:
	cd tests/integration && TESTCONTAINERS_RYUK_DISABLED=true go test -tags=containers -v -run E2E -timeout 5m .

rust-test:
	cargo test --manifest-path $(RUST_MANIFEST)

rust-build:
	cargo build --release --manifest-path $(RUST_MANIFEST)
	chmod +x $(RUST_BINARY)

fmt:
	go fmt ./...

vet:
	go vet ./...

check: fmt vet test rust-test check-secrets

audit:
	PROFILE=$${PROFILE:-hardened}; python3 tools/audit/audit.py --profile "$$PROFILE" --project-root .

policy:
	PROFILE=$${PROFILE:-hardened}; bash scripts/policy-check.sh "$$PROFILE"

helm-lint:
	helm lint $(HELM_CHART)

helm-template:
	PROFILE=$${PROFILE:-hardened}; helm template sentinelops $(HELM_CHART) \
		-f $(HELM_CHART)/values.yaml \
		-f $(HELM_CHART)/values-$$PROFILE.yaml

helm-install:
	PROFILE=$${PROFILE:-hardened}; bash scripts/helm-deploy.sh "$$PROFILE"

bootstrap:
	bash scripts/bootstrap.sh

setup-dev: generate-secrets ssh-lab-setup
	@echo "Entorno de desarrollo preparado."
	@echo "Ejecuta: source .env.local && make run-ssh"

install-dev-tools:
	bash scripts/setup-dev-ubuntu.sh

generate-secrets:
	bash scripts/generate-secrets.sh

check-secrets:
	bash scripts/check-no-hardcoded-secrets.sh

e2e:
	bash scripts/test-e2e.sh

e2e-full:
	bash scripts/test-e2e-full.sh

clean:
	rm -rf bin
	rm -rf rust/input-guard/target

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-run-tcp

docker-run-tcp:
	docker run --rm -it \
		-p 2324:2323 \
		-p 9001:9001 \
		-p 9443:9443 \
		-e APP_ENV=container \
		-e APP_PROFILE=hardened \
		-e APP_ADDR=:2323 \
		-e METRICS_ADDR=:9001 \
		-e APP_CONTROL_API_ENABLED=true \
		-e APP_CONTROL_API_ADDR=:9443 \
		-e APP_CONTROL_API_USER=admin \
		-e APP_CONTROL_API_PASSWORD=$${APP_CONTROL_API_PASSWORD:-} \
		-e LAB_PASSWORD_STUDENT=$${LAB_PASSWORD_STUDENT:-} \
		-e LAB_PASSWORD_TEACHER=$${LAB_PASSWORD_TEACHER:-} \
		-e LAB_PASSWORD_AUDITOR=$${LAB_PASSWORD_AUDITOR:-} \
		-e LAB_PASSWORD_ADMIN=$${LAB_PASSWORD_ADMIN:-} \
		-e APP_AUTH_RATE_LIMIT_ENABLED=true \
		-e APP_AUTH_RATE_LIMIT_MAX_FAILURES=5 \
		-e APP_AUTH_RATE_LIMIT_WINDOW=1m \
		-e APP_AUTH_RATE_LIMIT_LOCKOUT=1m \
		-e APP_STATE_PERSISTENCE_ENABLED=false \
		-e APP_STATE_PERSISTENCE_DIR=/data/state \
		-e APP_STATE_SESSIONS_PATH=/data/state/sessions.json \
		-e APP_STATE_TUNNELS_PATH=/data/state/tunnels.json \
		-e EXTERNAL_AUDIT_ENABLED=true \
		-e EXTERNAL_AUDIT_COMMAND=python3 \
		-e EXTERNAL_AUDIT_SCRIPT=/app/tools/audit/audit.py \
		-e EXTERNAL_VALIDATOR_ENABLED=true \
		-e EXTERNAL_VALIDATOR_BINARY=/app/bin/input-guard \
		-e EXTERNAL_VALIDATOR_FAIL_OPEN=false \
		-e OPA_POLICY_ENABLED=true \
		-e OPA_POLICY_MODE=$${OPA_POLICY_MODE:-exec} \
		-e OPA_POLICY_URL=$${OPA_POLICY_URL:-http://localhost:8181} \
		-e OPA_POLICY_TIMEOUT=$${OPA_POLICY_TIMEOUT:-2s} \
		-e OPA_POLICY_CACHE_ENABLED=$${OPA_POLICY_CACHE_ENABLED:-true} \
		-e OPA_POLICY_CACHE_TTL=$${OPA_POLICY_CACHE_TTL:-30s} \
		-e OPA_BINARY=/app/bin/opa \
		-e OPA_POLICY_DIR=/app/policies/kubernetes \
		--name sentinelops-local \
		$(IMAGE)

docker-run-ssh:
	docker run --rm -it \
		-p 2222:2222 \
		-p 9001:9001 \
		-p 9443:9443 \
		-e APP_ENV=container \
		-e APP_PROFILE=hardened \
		-e APP_TRANSPORT=ssh \
		-e APP_SSH_ADDR=:2222 \
		-e METRICS_ADDR=:9001 \
		-e APP_CONTROL_API_ENABLED=true \
		-e APP_CONTROL_API_ADDR=:9443 \
		-e APP_CONTROL_API_USER=admin \
		-e APP_CONTROL_API_PASSWORD=$${APP_CONTROL_API_PASSWORD:-} \
		-e LAB_PASSWORD_STUDENT=$${LAB_PASSWORD_STUDENT:-} \
		-e LAB_PASSWORD_TEACHER=$${LAB_PASSWORD_TEACHER:-} \
		-e LAB_PASSWORD_AUDITOR=$${LAB_PASSWORD_AUDITOR:-} \
		-e LAB_PASSWORD_ADMIN=$${LAB_PASSWORD_ADMIN:-} \
		-e APP_SSH_LOCAL_FORWARD_ENABLED=true \
		-e APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9001,localhost:9001 \
		-e APP_SSH_LOCAL_ALLOWED_ROLES=student,teacher,auditor,admin \
		-e APP_SSH_REMOTE_FORWARD_ENABLED=false \
		-e APP_SSH_REMOTE_BIND_ALLOWLIST=127.0.0.1:10080,127.0.0.1:10443 \
		-e APP_SSH_REMOTE_ALLOWED_ROLES=teacher,auditor,admin \
		-e APP_AUTH_RATE_LIMIT_ENABLED=true \
		-e APP_AUTH_RATE_LIMIT_MAX_FAILURES=5 \
		-e APP_AUTH_RATE_LIMIT_WINDOW=1m \
		-e APP_AUTH_RATE_LIMIT_LOCKOUT=1m \
		-e APP_STATE_PERSISTENCE_ENABLED=false \
		-e APP_STATE_PERSISTENCE_DIR=/data/state \
		-e APP_STATE_SESSIONS_PATH=/data/state/sessions.json \
		-e APP_STATE_TUNNELS_PATH=/data/state/tunnels.json \
		-e EXTERNAL_AUDIT_ENABLED=true \
		-e EXTERNAL_AUDIT_COMMAND=python3 \
		-e EXTERNAL_AUDIT_SCRIPT=/app/tools/audit/audit.py \
		-e EXTERNAL_VALIDATOR_ENABLED=true \
		-e EXTERNAL_VALIDATOR_BINARY=/app/bin/input-guard \
		-e EXTERNAL_VALIDATOR_FAIL_OPEN=false \
		-e OPA_POLICY_ENABLED=true \
		-e OPA_POLICY_MODE=$${OPA_POLICY_MODE:-exec} \
		-e OPA_POLICY_URL=$${OPA_POLICY_URL:-http://localhost:8181} \
		-e OPA_POLICY_TIMEOUT=$${OPA_POLICY_TIMEOUT:-2s} \
		-e OPA_POLICY_CACHE_ENABLED=$${OPA_POLICY_CACHE_ENABLED:-true} \
		-e OPA_POLICY_CACHE_TTL=$${OPA_POLICY_CACHE_TTL:-30s} \
		-e OPA_BINARY=/app/bin/opa \
		-e OPA_POLICY_DIR=/app/policies/kubernetes \
		--name sentinelops-local \
		$(IMAGE)

deploy-local:
	PROFILE=$${PROFILE:-hardened}; TRANSPORT=$${TRANSPORT:-ssh}; bash scripts/deploy-local.sh "$$PROFILE" "$$TRANSPORT"

docker-stop:
	-docker stop sentinelops-local

cleanup:
	bash scripts/cleanup.sh
	rm -f .env.local

.PHONY: docs docs-check api-smoke

docs:
	bash scripts/generate-openapi.sh

docs-check:
	python3 -m json.tool docs/swagger.json >/dev/null
	python3 -m json.tool internal/controlplane/httpapi/openapi.json >/dev/null
	@echo "Especificación OpenAPI válida."

api-smoke:
	@API_URL=$${API_URL:-https://localhost:9443}; \
	API_USER=$${API_USER:-$${APP_CONTROL_API_USER:-admin}}; \
	API_PASSWORD=$${API_PASSWORD:-$${APP_CONTROL_API_PASSWORD:-}}; \
	curl -ksf "$$API_URL/healthz/live" >/dev/null; \
	curl -ksf "$$API_URL/healthz/ready" >/dev/null; \
	curl -ksf "$$API_URL/healthz/startup" >/dev/null; \
	curl -ksf "$$API_URL/api/v1/docs/swagger.json" >/dev/null; \
	curl -ksf -u "$$API_USER:$$API_PASSWORD" "$$API_URL/api/v1/admin/status" >/dev/null; \
	echo "API v1 verificada en $$API_URL"

.PHONY: proto-tools proto-go proto-clean validator-grpc-build validator-grpc-test validator-grpc-docker validator-grpc-up validator-grpc-down validator-grpc-logs validator-grpc-smoke

proto-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

proto-go:
	bash scripts/generate-proto.sh

proto-clean:
	rm -rf gen/go

validator-grpc-build:
	cargo build --manifest-path rust/input-guard-grpc/Cargo.toml --release

validator-grpc-test:
	cargo test --manifest-path rust/input-guard-grpc/Cargo.toml

validator-grpc-docker:
	docker build -f rust/input-guard-grpc/Dockerfile -t sentinelops/input-guard-grpc:local .

validator-grpc-up:
	HOST_UID=$$(id -u) HOST_GID=$$(id -g) docker compose -f docker-compose.grpc.yml up --build

validator-grpc-down:
	docker compose -f docker-compose.grpc.yml down --remove-orphans

validator-grpc-logs:
	docker compose -f docker-compose.grpc.yml logs --tail=200 -f

validator-grpc-smoke:
	@API_URL=$${API_URL:-https://localhost:9446}; \
	API_USER=$${API_USER:-$${APP_CONTROL_API_USER:-admin}}; \
	API_PASSWORD=$${API_PASSWORD:-$${APP_CONTROL_API_PASSWORD:-}}; \
	curl -ksf "$$API_URL/healthz/live" >/dev/null; \
	curl -ksf "$$API_URL/api/v1/docs/swagger.json" >/dev/null; \
	curl -ksf -u "$$API_USER:$$API_PASSWORD" "$$API_URL/api/v1/admin/status" >/dev/null; \
	echo "Stack gRPC verificado en $$API_URL"

.PHONY: ci-check ci-openapi ci-proto ci-security ci-clean release-tag

ci-check:
	bash scripts/ci-check.sh

ci-openapi:
	$(MAKE) docs
	$(MAKE) docs-check

ci-proto:
	$(MAKE) proto-go
	$(MAKE) proto-clean

ci-security:
	$(MAKE) check-secrets
	@if command -v gosec >/dev/null 2>&1; then gosec ./...; else echo "gosec no está instalado"; fi
	@if command -v trivy >/dev/null 2>&1; then trivy fs --exit-code 0 .; else echo "trivy no está instalado"; fi

ci-clean:
	$(MAKE) proto-clean
	$(MAKE) opa-clean
	rm -f coverage.out coverage.html gosec.sarif trivy-fs.sarif
	rm -rf reports/release

release-tag:
	@read -r -p "Versión, ejemplo 0.7.0: " version; \
	git tag -a "v$$version" -m "release v$$version"; \
	git push origin "v$$version"; \
	echo "Tag v$$version creado"
