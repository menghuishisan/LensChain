#!/bin/bash
# init-db.sh
# 初始化数据库：建库 + 执行迁移 + 种子数据
# 用途：首次部署或清空数据库后执行
# 幂等：可重复执行不出错
# 密码来源：统一从 deploy/config.env 读取，不硬编码

set -euo pipefail

# ---------------------------------------------------------------------------
# 加载 config.env
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${1:-${SCRIPT_DIR}/../../config.env}"

if [ ! -f "${ENV_FILE}" ]; then
    echo "[init-db] 错误：找不到配置文件: ${ENV_FILE}"
    echo "请先将 config.example.env 复制为 config.env 并填入真实值"
    exit 1
fi

# 解析 config.env（跳过注释和空行）
while IFS='=' read -r key value; do
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | xargs)
    [[ -z "$key" || "$key" == \#* ]] && continue
    export "$key=$value"
done < "$ENV_FILE"

DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5442}
DB_USER=${DB_USER:-lenschain}
DB_NAME=${DB_NAME:-lenschain}

if [ -z "${DB_PASSWORD:-}" ]; then
    echo "[init-db] 错误：DB_PASSWORD 未设置，请检查 config.env"
    exit 1
fi

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
IMAGES_SEED_FILE="migrations/012_seed_images_experiments.up.sql"
SIM_SCENARIOS_SEED_FILE="migrations/013_seed_sim_scenarios.up.sql"
CTF_SEED_FILE="migrations/014_seed_ctf.up.sql"

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

if [ -f "$IMAGES_SEED_FILE" ]; then
  echo "==> Seeding images & experiments data from $IMAGES_SEED_FILE"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$IMAGES_SEED_FILE"
else
  echo "==> Images & experiments seed file not found, skipping"
fi

if [ -f "$SIM_SCENARIOS_SEED_FILE" ]; then
  echo "==> Seeding sim scenarios data from $SIM_SCENARIOS_SEED_FILE"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$SIM_SCENARIOS_SEED_FILE"
else
  echo "==> Sim scenarios seed file not found, skipping"
fi

if [ -f "$CTF_SEED_FILE" ]; then
  echo "==> Seeding CTF data from $CTF_SEED_FILE"
  psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f "$CTF_SEED_FILE"
else
  echo "==> CTF seed file not found, skipping"
fi

echo "==> Database initialization complete"
