APP_NAME=sentinelops
BINARY=bin/sentinelops
CLIENT_BINARY=bin/sentinelops-client
IMAGE=sentinelops:local
RUST_MANIFEST=rust/input-guard/Cargo.toml
RUST_BINARY=rust/input-guard/target/release/input-guard
HELM_CHART=deploy/helm/sentinelops

ENV_FILE ?=
ifneq ($(strip $(ENV_FILE)),)
include $(ENV_FILE)
export
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
APP_CONTROL_API_PASSWORD ?= admin123!
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

.PHONY: help build build-client run run-tcp run-ssh ssh-lab-setup demo docker-demo curl-examples test rust-test rust-build fmt vet clean check audit policy helm-lint helm-template helm-install bootstrap setup-dev e2e e2e-full docker-build docker-run docker-run-tcp docker-run-ssh docker-demo-up docker-demo-down docker-demo-logs docker-stop deploy-local cleanup

help:
	@echo "Targets disponibles:"
	@echo "  make build            - Compila servidor y cliente"
	@echo "  make build-client     - Compila cliente SSH"
	@echo "  make run              - Ejecuta el servidor local (por defecto TCP)"
	@echo "  make run-tcp          - Ejecuta el servidor en modo TCP"
	@echo "  make run-ssh          - Ejecuta el servidor en modo SSH"
	@echo "  make ssh-lab-setup    - Genera llave de laboratorio y authorized_keys"
	@echo "  make curl-examples    - Ejecuta ejemplos curl contra la API HTTPS"
	@echo "  make demo             - Ejecuta una demostración guiada end-to-end"
	@echo "  make test             - Ejecuta pruebas Go"
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
	@echo "  make setup-dev        - Instala dependencias de desarrollo en Ubuntu"
	@echo "  make e2e              - Ejecuta prueba end-to-end local TCP"
	@echo "  make e2e-full         - Ejecuta validación E2E Docker con evidencias"
	@echo "  make docker-build     - Construye imagen Docker"
	@echo "  make docker-run-tcp   - Ejecuta contenedor local en modo TCP"
	@echo "  make docker-run-ssh   - Ejecuta contenedor local en modo SSH"
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
	OPA_BINARY=opa \
	OPA_POLICY_DIR=policies/kubernetes \
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
	OPA_BINARY=opa \
	OPA_POLICY_DIR=policies/kubernetes \
	APP_TRANSPORT=ssh \
	APP_SSH_ADDR=$(APP_SSH_ADDR) \
	APP_SSH_LOCAL_FORWARD_ENABLED=true \
	APP_SSH_FORWARD_ALLOWLIST=$(APP_SSH_FORWARD_ALLOWLIST) \
	APP_SSH_LOCAL_ALLOWED_ROLES=$(APP_SSH_LOCAL_ALLOWED_ROLES) \
	APP_SSH_REMOTE_FORWARD_ENABLED=$(APP_SSH_REMOTE_FORWARD_ENABLED) \
	APP_SSH_REMOTE_BIND_ALLOWLIST=$(APP_SSH_REMOTE_BIND_ALLOWLIST) \
	APP_SSH_REMOTE_ALLOWED_ROLES=$(APP_SSH_REMOTE_ALLOWED_ROLES) \
	go run ./cmd/server

ssh-lab-setup:
	USER_NAME=$${USER_NAME:-student}; bash scripts/setup-ssh-lab.sh "$$USER_NAME"

curl-examples:
	API_URL=$${API_URL:-https://localhost:9443} API_USER=$${API_USER:-admin} API_PASSWORD=$${API_PASSWORD:-admin123!} bash scripts/control-api-curl-examples.sh

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

test:
	go test ./...

rust-test:
	cargo test --manifest-path $(RUST_MANIFEST)

rust-build:
	cargo build --release --manifest-path $(RUST_MANIFEST)
	chmod +x $(RUST_BINARY)

fmt:
	go fmt ./...

vet:
	go vet ./...

check: fmt vet test rust-test

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

setup-dev:
	bash scripts/setup-dev-ubuntu.sh

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
		-e APP_CONTROL_API_PASSWORD=admin123! \
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
		-e APP_CONTROL_API_PASSWORD=admin123! \
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
