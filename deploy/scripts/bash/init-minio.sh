#!/bin/bash
# init-minio.sh
# 创建默认 MinIO Bucket + 权限策略
# 用途：首次部署或清空 MinIO 后执行
# 幂等：可重复执行不出错（--ignore-existing）

set -euo pipefail

MINIO_HOST=${MINIO_HOST:-localhost}
MINIO_PORT=${MINIO_PORT:-9010}
MINIO_USER=${MINIO_USER:-minioadmin}
MINIO_PASSWORD=${MINIO_PASSWORD:-minioadmin}

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
