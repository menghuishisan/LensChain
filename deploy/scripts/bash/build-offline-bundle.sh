#!/bin/bash
# build-offline-bundle.sh
# 构建离线镜像包，供学校私有化部署使用
# 用途：从 Registry 拉取所有 Phase 1 镜像，打包为 tar + SHA256 校验文件

set -euo pipefail

REGISTRY=${REGISTRY:-registry.lianjing.com}
VERSION=${VERSION:?"Usage: build-offline-bundle.sh (requires VERSION env)"}
OUTPUT_DIR=${OUTPUT_DIR:-.}
BUNDLE_NAME="lenschain-phase1-${VERSION}"

DEPLOY_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
IMAGES=()

echo "==> Collecting Phase 1 image list from manifests"

for MANIFEST in "$DEPLOY_DIR"/images/**/manifest.yaml; do
  [ ! -f "$MANIFEST" ] && continue

  PHASE=$(grep 'prepull_phase:' "$MANIFEST" | awk '{print $2}' | tr -d '"')
  [ "$PHASE" != "1" ] && continue

  PROJECT=$(grep 'registry_project:' "$MANIFEST" | awk '{print $2}' | tr -d '"')
  NAME=$(grep '^name:' "$MANIFEST" | head -1 | awk '{print $2}' | tr -d '"')

  while IFS= read -r TAG; do
    TAG=$(echo "$TAG" | tr -d '"' | xargs)
    [ -z "$TAG" ] && continue
    FULL="${REGISTRY}/${PROJECT}/${NAME}:${TAG}"
    IMAGES+=("$FULL")
  done < <(grep '    tag:' "$MANIFEST" | awk '{print $2}')
done

echo "==> Found ${#IMAGES[@]} images to bundle"

echo "==> Pulling images"
for IMG in "${IMAGES[@]}"; do
  echo "    Pulling: $IMG"
  docker pull "$IMG"
done

echo "==> Saving to ${OUTPUT_DIR}/${BUNDLE_NAME}.tar"
docker save -o "${OUTPUT_DIR}/${BUNDLE_NAME}.tar" "${IMAGES[@]}"

echo "==> Generating SHA256 checksum"
sha256sum "${OUTPUT_DIR}/${BUNDLE_NAME}.tar" > "${OUTPUT_DIR}/${BUNDLE_NAME}.tar.sha256"

echo "==> Bundle complete"
ls -lh "${OUTPUT_DIR}/${BUNDLE_NAME}.tar" "${OUTPUT_DIR}/${BUNDLE_NAME}.tar.sha256"
