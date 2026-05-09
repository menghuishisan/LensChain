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

echo "==> Running schema migrations (backend/migrations)"
cd "$(dirname "$0")/../../../backend"
if [ -z "${GOCACHE:-}" ]; then
  export GOCACHE="$PWD/.gocache"
  mkdir -p "$GOCACHE"
fi
go run cmd/migrate/main.go up

# 种子数据加载顺序：
#   schema 迁移 → image_categories（sync 前置依赖）
#   → seed-manifests CLI（从 manifest.yaml 灌入 images / image_versions）
#   → 其它 seed（CTF 题目模板 / 演示数据 / 实验模板 / sim 场景 / CTF 竞赛），
#     这些 seed 通过 (image_name, version) 子查询关联 image_version_id，
#     必须排在 sync 之后才能解析。
IMAGE_CATEGORIES_SEED="seeds/000_seed_image_categories.sql"
CTF_TEMPLATES_SEED="seeds/001_seed_ctf_challenge_templates.sql"
DEMO_SEED="seeds/002_seed_demo_data.sql"
DEMO_SUPPLEMENT_SEED="seeds/003_seed_demo_supplement.sql"
IMAGES_EXPERIMENTS_SEED="seeds/004_seed_images_experiments.sql"
SIM_SCENARIOS_SEED="seeds/005_seed_sim_scenarios.sql"
CTF_SEED="seeds/006_seed_ctf.sql"

run_seed() {
  local label="$1"
  local file="$2"
  if [ -f "$file" ]; then
    echo "==> Seeding $label from $file"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -f "$file"
  else
    echo "==> $label seed file not found ($file), skipping"
  fi
}

run_seed "image categories" "$IMAGE_CATEGORIES_SEED"

echo "==> Syncing image manifests (deploy/images/**/manifest.yaml → images / image_versions)"
DEPLOY_IMAGES_DIR="$(cd "$(dirname "$0")/../.." && pwd)/images"
go run cmd/seed-manifests/main.go -images-dir "$DEPLOY_IMAGES_DIR"

run_seed "CTF challenge templates"        "$CTF_TEMPLATES_SEED"
run_seed "demo data"                      "$DEMO_SEED"
run_seed "demo supplement"                "$DEMO_SUPPLEMENT_SEED"
run_seed "images & experiments templates" "$IMAGES_EXPERIMENTS_SEED"
run_seed "sim scenarios"                  "$SIM_SCENARIOS_SEED"
run_seed "CTF competitions"               "$CTF_SEED"

echo "==> Database initialization complete"
