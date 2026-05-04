#!/usr/bin/env bash
# deploy/scripts/bash/create-secrets.sh
# 从 deploy/config.env 读取密码，创建 K8s Secret
# 用法：./deploy/scripts/bash/create-secrets.sh
# 说明：所有密码统一从 config.env 获取，不硬编码

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${1:-${SCRIPT_DIR}/../../config.env}"

# ---------------------------------------------------------------------------
# 加载 config.env
# ---------------------------------------------------------------------------

if [ ! -f "${ENV_FILE}" ]; then
    echo "[create-secrets] 错误：找不到配置文件: ${ENV_FILE}"
    echo "请先将 config.example.env 复制为 config.env 并填入真实值"
    exit 1
fi

echo "[create-secrets] 加载配置文件: ${ENV_FILE}"

# 加载环境变量（跳过注释和空行）
set -a
# shellcheck disable=SC1090
source <(grep -v '^\s*#' "${ENV_FILE}" | grep -v '^\s*$')
set +a

# ---------------------------------------------------------------------------
# 校验必填字段
# ---------------------------------------------------------------------------

REQUIRED_KEYS=(
    BACKEND_DATABASE_PASSWORD
    BACKEND_JWT_ACCESS_SECRET
    BACKEND_JWT_REFRESH_SECRET
    BACKEND_REDIS_PASSWORD
    BACKEND_SNAPSHOT_ENCRYPTION_KEY
    POSTGRES_PASSWORD
    REDIS_PASSWORD
    MINIO_ROOT_USER
    MINIO_ROOT_PASSWORD
)

MISSING=()
for key in "${REQUIRED_KEYS[@]}"; do
    val="${!key:-}"
    if [ -z "${val}" ] || [ "${val}" = "CHANGE_ME" ]; then
        MISSING+=("${key}")
    fi
done

if [ ${#MISSING[@]} -gt 0 ]; then
    echo "[create-secrets] 错误：以下字段未填写或仍为 CHANGE_ME:"
    printf '  - %s\n' "${MISSING[@]}"
    exit 1
fi

# ---------------------------------------------------------------------------
# 读取命名空间
# ---------------------------------------------------------------------------

NAMESPACE="${K8S_NAMESPACE:-lenschain}"
echo "[create-secrets] 目标命名空间: ${NAMESPACE}"

# ---------------------------------------------------------------------------
# 创建 Secret
# ---------------------------------------------------------------------------

# 1. backend-secret
echo "[create-secrets] 创建 backend-secret ..."
kubectl -n "${NAMESPACE}" create secret generic backend-secret \
    --from-literal=database-password="${BACKEND_DATABASE_PASSWORD}" \
    --from-literal=jwt-access-secret="${BACKEND_JWT_ACCESS_SECRET}" \
    --from-literal=jwt-refresh-secret="${BACKEND_JWT_REFRESH_SECRET}" \
    --from-literal=redis-password="${BACKEND_REDIS_PASSWORD}" \
    --from-literal=snapshot-encryption-key="${BACKEND_SNAPSHOT_ENCRYPTION_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -

# 2. postgres-secret
echo "[create-secrets] 创建 postgres-secret ..."
kubectl -n "${NAMESPACE}" create secret generic postgres-secret \
    --from-literal=password="${POSTGRES_PASSWORD}" \
    --dry-run=client -o yaml | kubectl apply -f -

# 3. redis-secret
echo "[create-secrets] 创建 redis-secret ..."
kubectl -n "${NAMESPACE}" create secret generic redis-secret \
    --from-literal=password="${REDIS_PASSWORD}" \
    --dry-run=client -o yaml | kubectl apply -f -

# 4. minio-secret
echo "[create-secrets] 创建 minio-secret ..."
kubectl -n "${NAMESPACE}" create secret generic minio-secret \
    --from-literal=root-user="${MINIO_ROOT_USER}" \
    --from-literal=root-password="${MINIO_ROOT_PASSWORD}" \
    --dry-run=client -o yaml | kubectl apply -f -

# 5. registry-pull-secret（仅当配置了用户名密码时创建）
if [ -n "${REGISTRY_USER:-}" ] && [ "${REGISTRY_USER}" != "CHANGE_ME" ] \
   && [ -n "${REGISTRY_PASSWORD:-}" ] && [ "${REGISTRY_PASSWORD}" != "CHANGE_ME" ]; then
    SERVER="${REGISTRY_SERVER:-registry.lianjing.com}"
    echo "[create-secrets] 创建 registry-pull-secret ..."
    kubectl -n "${NAMESPACE}" create secret docker-registry registry-pull-secret \
        --docker-server="${SERVER}" \
        --docker-username="${REGISTRY_USER}" \
        --docker-password="${REGISTRY_PASSWORD}" \
        --dry-run=client -o yaml | kubectl apply -f -
else
    echo "[create-secrets] 跳过 registry-pull-secret（REGISTRY_USER/REGISTRY_PASSWORD 未配置）"
fi

echo "[create-secrets] 全部完成"
