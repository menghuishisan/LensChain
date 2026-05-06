#!/bin/bash
# init-minio.sh
# 创建默认 MinIO Bucket + 权限策略
# 用途：首次部署或清空 MinIO 后执行
# 幂等：可重复执行不出错（--ignore-existing）
# 密码来源：统一从 deploy/config.env 读取，不硬编码

set -euo pipefail

# ---------------------------------------------------------------------------
# 加载 config.env
# ---------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${1:-${SCRIPT_DIR}/../../config.env}"

if [ ! -f "${ENV_FILE}" ]; then
    echo "[init-minio] 错误：找不到配置文件: ${ENV_FILE}"
    echo "请先将 config.example.env 复制为 config.env 并填入真实值"
    exit 1
fi

while IFS='=' read -r key value; do
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | xargs)
    [[ -z "$key" || "$key" == \#* ]] && continue
    export "$key=$value"
done < "$ENV_FILE"

MINIO_HOST=${MINIO_HOST:-localhost}
MINIO_PORT=${MINIO_PORT:-9010}
MINIO_USER=${MINIO_ROOT_USER:-${MINIO_USER:-}}
MINIO_PASSWORD=${MINIO_ROOT_PASSWORD:-${MINIO_PASSWORD:-}}

if [ -z "$MINIO_USER" ] || [ -z "$MINIO_PASSWORD" ]; then
    echo "[init-minio] 错误：MINIO_ROOT_USER / MINIO_ROOT_PASSWORD 未设置，请检查 config.env"
    exit 1
fi

echo "==> Configuring MinIO client alias"
mc alias set local "http://${MINIO_HOST}:${MINIO_PORT}" "$MINIO_USER" "$MINIO_PASSWORD"

BUCKETS=("attachments" "reports" "snapshots" "images" "exports" "backups")

for BUCKET in "${BUCKETS[@]}"; do
  echo "==> Creating bucket: lenschain-$BUCKET"
  mc mb --ignore-existing "local/lenschain-$BUCKET"
  mc anonymous set none "local/lenschain-$BUCKET"
done

echo "==> MinIO initialization complete"
echo "    Buckets created: ${BUCKETS[*]}"
