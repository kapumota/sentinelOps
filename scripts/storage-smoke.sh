#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.storage.yml}"
ENV_FILE="${ENV_FILE:-.env.local}"

if [ ! -f "$ENV_FILE" ]; then
    echo "ERROR .env.local no existe. Ejecuta make generate-secrets primero."
    exit 1
fi

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

POSTGRES_USER="${POSTGRES_USER:-sentinelops}"
POSTGRES_DB="${POSTGRES_DB:-sentinelops}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"

echo "Verificando contenedores de almacenamiento"

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps postgres >/dev/null
docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" ps redis >/dev/null

echo "Verificando PostgreSQL"

docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T postgres \
    pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"

echo "Verificando Redis"

if [ -n "$REDIS_PASSWORD" ]; then
    redis_result="$(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T \
        -e REDISCLI_AUTH="$REDIS_PASSWORD" redis redis-cli ping | tr -d '\r\n')"
else
    redis_result="$(docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" exec -T \
        redis redis-cli ping | tr -d '\r\n')"
fi

if [ "$redis_result" != "PONG" ]; then
    echo "ERROR Redis no respondió PONG"
    echo "Respuesta recibida: $redis_result"
    exit 1
fi

echo "Verificando migraciones SQL"

test -f deploy/postgres/init/001_init_storage.sql
test -f migrations/000001_storage.up.sql
test -f migrations/000001_storage.down.sql

echo "Verificando variables de storage"

grep -q "^STORE_TYPE=" "$ENV_FILE"
grep -q "^POSTGRES_HOST=" "$ENV_FILE"
grep -q "^REDIS_ADDR=" "$ENV_FILE"

echo "Storage smoke OK"
