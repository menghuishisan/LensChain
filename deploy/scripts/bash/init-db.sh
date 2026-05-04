#!/bin/bash
# init-db.sh
# 初始化数据库：建库 + 执行迁移 + 种子数据
# 用途：首次部署或清空数据库后执行
# 幂等：可重复执行不出错

set -euo pipefail

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5442}
DB_USER=${DB_USER:-lenschain}
DB_PASSWORD=${DB_PASSWORD:-lenschain}
DB_NAME=${DB_NAME:-lenschain}

export PGPASSWORD="$DB_PASSWORD"
export LENSCHAIN_DATABASE_HOST="$DB_HOST"
export LENSCHAIN_DATABASE_PORT="$DB_PORT"
export LENSCHAIN_DATABASE_USER="$DB_USER"
export LENSCHAIN_DATABASE_PASSWORD="$DB_PASSWORD"
export LENSCHAIN_DATABASE_DBNAME="$DB_NAME"
export LENSCHAIN_DATABASE_SSLMODE="${LENSCHAIN_DATABASE_SSLMODE:-disable}"

echo "==> Checking database connection"
until pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" 2>/dev/null; do
  echo "    Waiting for PostgreSQL..."
  sleep 2
done

echo "==> Recreating database"
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tAc \
  "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | grep -q 1; then
  echo "    Existing database found, terminating connections"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 -c \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();"
  echo "    Dropping database $DB_NAME"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 -c \
    "DROP DATABASE \"$DB_NAME\""
fi

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -v ON_ERROR_STOP=1 -c \
  "CREATE DATABASE \"$DB_NAME\""

REDIS_DB=${REDIS_DB:-0}
REDIS_CONTAINER=${REDIS_CONTAINER:-lenschain-redis}

echo "==> Flushing Redis cache (container=$REDIS_CONTAINER, db=$REDIS_DB)"
docker exec "$REDIS_CONTAINER" redis-cli -n "$REDIS_DB" FLUSHDB || { echo "Redis 缓存清理失败，请确认容器 $REDIS_CONTAINER 正在运行"; exit 1; }

echo "==> Running migrations"
cd "$(dirname "$0")/../../../backend"
if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="$PWD/.gocache"
  mkdir -p "$GOCACHE"
fi
go run cmd/migrate/main.go up

DEMO_SEED_FILE="migrations/010_seed_demo_data.up.sql"
DEMO_SUPPLEMENT_FILE="migrations/011_seed_demo_supplement.up.sql"

if [ -f "$DEMO_SEED_FILE" ]; then
  echo "==> Seeding demo data from $DEMO_SEED_FILE"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$DEMO_SEED_FILE"
else
  echo "==> Demo seed file not found, skipping"
fi

if [ -f "$DEMO_SUPPLEMENT_FILE" ]; then
  echo "==> Seeding supplement data from $DEMO_SUPPLEMENT_FILE"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$DEMO_SUPPLEMENT_FILE"
else
  echo "==> Supplement seed file not found, skipping"
fi

echo "==> Database initialization complete"
