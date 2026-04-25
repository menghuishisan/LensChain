#!/bin/bash
# preload-images.sh
# 手动刷新镜像预拉取清单并触发 DaemonSet 重新拉取（运维调试用）
# 用途：扫描 deploy/images/**/manifest.yaml，生成 image-manifest ConfigMap，再触发所有 K8s 节点重新拉取

set -euo pipefail
shopt -s globstar nullglob

NAMESPACE=${NAMESPACE:-lenschain-system}
TIMEOUT=${TIMEOUT:-30m}
REGISTRY=${REGISTRY:-registry.lianjing.com}
CONFIGMAP_NAME=${CONFIGMAP_NAME:-image-manifest}
PHASE_FILTER=${PHASE_FILTER:-}

DEPLOY_DIR="$(cd "$(dirname "$0")/.." && pwd)"
TMP_MANIFEST="$(mktemp)"

cleanup() {
  rm -f "$TMP_MANIFEST"
}
trap cleanup EXIT

echo "==> Building preload manifest from $DEPLOY_DIR/images/"
{
  echo "registry: ${REGISTRY}"
  echo "images:"
} > "$TMP_MANIFEST"

MANIFESTS=("$DEPLOY_DIR"/images/**/manifest.yaml)

if [ ${#MANIFESTS[@]} -eq 0 ]; then
  echo "ERROR: no manifest.yaml files found under $DEPLOY_DIR/images/"
  exit 1
fi

COUNT=0

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
    [ -z "$TAG" ] && continue
    {
      echo "  - project: $PROJECT"
      echo "    name: $NAME"
      echo "    tag: $TAG"
    } >> "$TMP_MANIFEST"
    COUNT=$((COUNT + 1))
  done < <(grep '^[[:space:]]\{4\}tag:' "$MANIFEST" | awk '{print $2}' | tr -d '"')
done

echo "==> Prepared $COUNT image entries"
if [ "$COUNT" -eq 0 ]; then
  echo "ERROR: preload manifest is empty"
  exit 1
fi

echo "==> Applying ConfigMap ${CONFIGMAP_NAME} in namespace ${NAMESPACE}"
kubectl create configmap "$CONFIGMAP_NAME" \
  --namespace "$NAMESPACE" \
  --from-file=manifest.yaml="$TMP_MANIFEST" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "==> Restarting image-prepuller DaemonSet"
kubectl -n "$NAMESPACE" rollout restart daemonset/image-prepuller

echo "==> Waiting for rollout to complete (timeout: $TIMEOUT)"
kubectl -n "$NAMESPACE" rollout status daemonset/image-prepuller --timeout="$TIMEOUT"

echo "==> Image preload trigger complete"
kubectl -n "$NAMESPACE" get pods -l app.kubernetes.io/name=image-prepuller -o wide
