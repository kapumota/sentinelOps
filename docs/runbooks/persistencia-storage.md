### Persistencia storage

#### Objetivo

La fase 9 agrega una abstracción de almacenamiento para SentinelOps. El modo por defecto sigue siendo `memory` para laboratorio, pero se agregan PostgreSQL y Redis para escenarios operativos.

#### Modos soportados

| Modo | Uso |
|---|---|
| `memory` | Laboratorio local, tests unitarios y desarrollo rápido |
| `postgres` | Persistencia durable de sesiones, túneles, auditoría y rate limits |
| `redis` | Cache, TTL, rate limiting y auditoría de alta velocidad |

#### Variables principales

    STORE_TYPE=memory
    POSTGRES_HOST=localhost
    POSTGRES_PORT=5432
    POSTGRES_DB=sentinelops
    POSTGRES_USER=sentinelops
    POSTGRES_PASSWORD=
    POSTGRES_SSLMODE=disable
    POSTGRES_POOL_SIZE=10
    REDIS_ADDR=localhost:6379
    REDIS_PASSWORD=
    REDIS_DB=0
    REDIS_POOL_SIZE=10

#### Levantar PostgreSQL y Redis

    make generate-secrets
    make storage-up
    source .env.local
    make storage-smoke

#### Apagar servicios

    make storage-down

#### Borrar volúmenes locales

    make storage-clean

#### Migraciones

Las migraciones están en `migrations/`. El stack Docker también carga `deploy/postgres/init/001_init_storage.sql` durante la inicialización del volumen.

#### Recomendación operativa

Usar `memory` para laboratorio y `postgres` para estado durable. Redis debe usarse para cache y rate limiting cuando se quiera baja latencia.
