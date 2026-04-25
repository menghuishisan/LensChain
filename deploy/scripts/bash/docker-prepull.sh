#!/bin/bash
# docker-prepull.sh
# 本地 Docker 镜像预拉取脚本
# 用途：扫描 deploy/images/**/manifest.yaml，并使用 docker pull 拉取镜像到当前开发机

set -euo pipefail
shopt -s globstar nullglob

REGISTRY=${REGISTRY:-registry.lianjing.com}
PHASE_FILTER=${PHASE_FILTER:-}

DEPLOY_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
COUNT=0
FAILED=0

echo "==> Collecting images from $DEPLOY_DIR/images/"

MANIFESTS=("$DEPLOY_DIR"/images/**/manifest.yaml)
if [ ${#MANIFESTS[@]} -eq 0 ]; then
  echo "ERROR: no manifest.yaml files found under $DEPLOY_DIR/images/"
  exit 1
fi

for MANIFEST in "${MANIFESTS[@]}"; do
  PROJECT=$(grep '^registry_project:' "$MANIFEST" | awk '{print $2}' | tr -d '"')
  NAME=$(grep '^name:' "$MANIFEST" | head -1 | awk '{print $2}' | tr -d '"')
  PHASE=$(grep '^prepull_phase:' "$MANIFEST" | awk '{print $2}' | tr -d '"')

  if [ -z "$PROJECT" ] || [ -z "$NAME" ]; then
    echo "WARN: skip invalid manifest $MANIFEST"
    continue
  fi

  if [ -n "$PHASE_FILTER" ] && [ "$PHASE" != "$PHASE_FILTER" ]; then
    continue
  fi

  while IFS= read -r TAG; do
    TAG=$(echo "$TAG" | tr -d '"' | xargs)
    [ -z "$TAG" ] && continue

    IMAGE="${REGISTRY}/${PROJECT}/${NAME}:${TAG}"
    echo "==> Pulling $IMAGE"
    if docker pull "$IMAGE"; then
      COUNT=$((COUNT + 1))
    else
      echo "FAILED: $IMAGE"
      FAILED=$((FAILED + 1))
    fi
  done < <(grep '^[[:space:]]\{4\}tag:' "$MANIFEST" | awk '{print $2}')
done

echo "==> Pull complete: $COUNT succeeded, $FAILED failed"

if [ "$FAILED" -gt 0 ]; then
  exit 1
fi
