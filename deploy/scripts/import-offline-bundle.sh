#!/bin/bash
# import-offline-bundle.sh
# 导入离线镜像包到本地 Docker 并推送到学校私有 Registry
# 用途：学校私有化部署时使用

set -euo pipefail

BUNDLE_FILE=${1:?"Usage: import-offline-bundle.sh <bundle.tar> [registry_url]"}
TARGET_REGISTRY=${2:-registry.local}

if [ ! -f "$BUNDLE_FILE" ]; then
  echo "ERROR: Bundle file not found: $BUNDLE_FILE"
  exit 1
fi

SHA_FILE="${BUNDLE_FILE}.sha256"
if [ -f "$SHA_FILE" ]; then
  echo "==> Verifying SHA256 checksum"
  sha256sum -c "$SHA_FILE"
else
  echo "WARNING: No SHA256 file found, skipping verification"
fi

echo "==> Loading images from bundle"
docker load -i "$BUNDLE_FILE"

echo "==> Retagging and pushing to $TARGET_REGISTRY"
for IMG in $(docker load -i "$BUNDLE_FILE" 2>&1 | grep 'Loaded image:' | awk '{print $3}'); do
  NEW_TAG=$(echo "$IMG" | sed "s|^[^/]*/|${TARGET_REGISTRY}/|")
  echo "    $IMG -> $NEW_TAG"
  docker tag "$IMG" "$NEW_TAG"
  docker push "$NEW_TAG"
done

echo "==> Import complete"
