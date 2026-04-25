#!/bin/bash
# seed-images.sh
# 扫描 deploy/images/**/manifest.yaml，通过后端管理 API 同步到 images 表
# 用途：首次部署或镜像清单变更后执行
# 幂等：新增镜像自动插入，已有镜像更新元数据

set -euo pipefail
shopt -s globstar nullglob

BACKEND_URL=${BACKEND_URL:-http://localhost:8080/api/v1}
ADMIN_TOKEN=${ADMIN_TOKEN:?"ADMIN_TOKEN is required"}

DEPLOY_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
COUNT=0
FAILED=0

echo "==> Scanning manifest files in $DEPLOY_DIR/images/"

MANIFESTS=("$DEPLOY_DIR"/images/**/manifest.yaml)

if [ ${#MANIFESTS[@]} -eq 0 ]; then
  echo "==> No manifest.yaml files found under $DEPLOY_DIR/images/"
  exit 0
fi

for MANIFEST in "${MANIFESTS[@]}"; do

  NAME=$(basename "$(dirname "$MANIFEST")")
  echo "    Syncing: $NAME"

  HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "${BACKEND_URL}/admin/images/sync" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/yaml" \
    --data-binary @"$MANIFEST")

  if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
    COUNT=$((COUNT + 1))
  else
    echo "    FAILED ($HTTP_CODE): $NAME"
    FAILED=$((FAILED + 1))
  fi
done

echo "==> Seed complete: $COUNT synced, $FAILED failed"

if [ "$FAILED" -gt 0 ]; then
  exit 1
fi
