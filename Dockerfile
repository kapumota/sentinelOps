# syntax=docker/dockerfile:1

FROM rust:1.79-alpine AS rust-builder
WORKDIR /build/rust/input-guard
COPY rust/input-guard /build/rust/input-guard
RUN cargo build --release && chmod +x target/release/input-guard

FROM golang:1.25-alpine AS go-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sentinelops ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/sentinelops-client ./cmd/client

FROM openpolicyagent/opa:1.17.1-static AS opa-builder

FROM alpine:3.20
LABEL org.opencontainers.image.title="sentinelops" \
      org.opencontainers.image.version="2.4.1" \
      org.opencontainers.image.description="SentinelOps secure remote access lab"
RUN apk add --no-cache python3 openssh-client curl
ARG APP_UID=1000
ARG APP_GID=1000
# El UID/GID se alinea con el usuario local para que los bind mounts sean escribibles.
RUN addgroup -S -g ${APP_GID} appgroup && adduser -S -D -H -u ${APP_UID} -G appgroup appuser

WORKDIR /app
RUN mkdir -p /app/bin /app/reports /app/data/controlplane /app/data/state /app/data/ssh/authorized_keys /app/data/ssh/client

COPY --from=go-builder /out/sentinelops /app/sentinelops
COPY --from=go-builder /out/sentinelops-client /app/bin/sentinelops-client
COPY --from=rust-builder /build/rust/input-guard/target/release/input-guard /app/bin/input-guard
COPY --from=opa-builder /opa /app/bin/opa
COPY tools /app/tools
COPY scripts /app/scripts
COPY policies /app/policies
COPY deploy /app/deploy
COPY env /app/env
COPY README.md /app/README.md

RUN chmod +x /app/bin/input-guard /app/bin/sentinelops-client /app/scripts/docker-entrypoint.sh && chown -R appuser:appgroup /app

USER appuser

EXPOSE 2323
EXPOSE 2222
EXPOSE 9001
EXPOSE 9443

ENV APP_NAME=sentinelops
ENV APP_VERSION=2.4.1
ENV APP_ENV=container
ENV APP_PROFILE=hardened
ENV APP_TRANSPORT=ssh
ENV APP_ADDR=:2323
ENV APP_SSH_ADDR=:2222
ENV METRICS_ADDR=:9001
ENV APP_CONTROL_API_ENABLED=true
ENV APP_CONTROL_API_ADDR=:9443
ENV APP_CONTROL_API_USER=admin
ENV APP_CONTROL_API_CERT_PATH=/app/data/controlplane/tls.crt
ENV APP_CONTROL_API_KEY_PATH=/app/data/controlplane/tls.key
ENV LOG_LEVEL=info
ENV APP_AUTH_ENABLED=true
ENV APP_AUTH_MAX_ATTEMPTS=3
ENV APP_AUTH_RATE_LIMIT_ENABLED=true
ENV APP_AUTH_RATE_LIMIT_MAX_FAILURES=5
ENV APP_AUTH_RATE_LIMIT_WINDOW=1m
ENV APP_AUTH_RATE_LIMIT_LOCKOUT=1m
ENV APP_STATE_PERSISTENCE_ENABLED=false
ENV APP_STATE_PERSISTENCE_DIR=/app/data/state
ENV APP_STATE_SESSIONS_PATH=/app/data/state/sessions.json
ENV APP_STATE_TUNNELS_PATH=/app/data/state/tunnels.json
ENV APP_PROJECT_ROOT=/app
ENV APP_SSH_LOCAL_FORWARD_ENABLED=true
ENV APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9001,localhost:9001
ENV APP_SSH_LOCAL_ALLOWED_ROLES=student,teacher,auditor,admin
ENV APP_SSH_REMOTE_FORWARD_ENABLED=false
ENV APP_SSH_REMOTE_BIND_ALLOWLIST=127.0.0.1:10080,127.0.0.1:10443
ENV APP_SSH_REMOTE_ALLOWED_ROLES=teacher,auditor,admin
ENV EXTERNAL_AUDIT_ENABLED=true
ENV EXTERNAL_AUDIT_COMMAND=python3
ENV EXTERNAL_AUDIT_SCRIPT=/app/tools/audit/audit.py
ENV EXTERNAL_VALIDATOR_ENABLED=true
ENV EXTERNAL_VALIDATOR_BINARY=/app/bin/input-guard
ENV EXTERNAL_VALIDATOR_FAIL_OPEN=false
ENV VALIDATOR_MODE=binary
ENV VALIDATOR_GRPC_ADDR=localhost:50051
ENV VALIDATOR_GRPC_TIMEOUT=2s
ENV VALIDATOR_GRPC_FAIL_OPEN=false
ENV OPA_POLICY_ENABLED=true
ENV OPA_POLICY_MODE=exec
ENV OPA_BINARY=/app/bin/opa
ENV OPA_POLICY_DIR=/app/policies/kubernetes
ENV OPA_POLICY_URL=http://localhost:8181
ENV OPA_POLICY_TIMEOUT=2s
ENV OPA_POLICY_CACHE_ENABLED=true
ENV OPA_POLICY_CACHE_TTL=30s

ENTRYPOINT ["/app/scripts/docker-entrypoint.sh"]
CMD ["/app/sentinelops"]
