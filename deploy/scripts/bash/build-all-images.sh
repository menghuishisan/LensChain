#!/usr/bin/env bash
# build-all-images.sh
# 本地构建全部项目镜像
# 扫描 deploy/images/**/Dockerfile 和 deploy/docker/*.Dockerfile，统一构建并打上项目 tag
#
# 用法：
#   ./build-all-images.sh                    # 构建全部镜像
#   PHASE_FILTER=1 ./build-all-images.sh     # 仅构建 Phase 1 镜像
#   DRY_RUN=1 ./build-all-images.sh          # 仅打印命令，不执行

set -euo pipefail

REGISTRY="${REGISTRY:-registry.lianjing.com}"
PHASE_FILTER="${PHASE_FILTER:-}"
DRY_RUN="${DRY_RUN:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
DEPLOY_DIR="$REPO_ROOT/deploy"
succeeded=0
failed=0
skipped=0

command -v docker >/dev/null 2>&1 || { echo "缺少依赖命令: docker"; exit 1; }

# ---------------------------------------------------------------------------
# 阶段 1：构建实验/比赛镜像（deploy/images/）
# ---------------------------------------------------------------------------

echo ""
echo "========================================"
echo " 阶段 1：实验/比赛镜像"
echo "========================================"
echo ""

while IFS= read -r manifest; do
    dir="$(dirname "$manifest")"
    dockerfile="$dir/Dockerfile"
    if [ ! -f "$dockerfile" ]; then
        echo "  SKIP (无 Dockerfile): $dir"
        skipped=$((skipped + 1))
        continue
    fi

    project=$(grep '^registry_project:' "$manifest" | head -1 | cut -d: -f2 | tr -d ' "')
    name=$(grep '^name:' "$manifest" | head -1 | cut -d: -f2 | tr -d ' "')
    phase=$(grep '^prepull_phase:' "$manifest" | head -1 | cut -d: -f2 | tr -d ' "' || true)

    if [ -z "$project" ] || [ -z "$name" ]; then
        echo "  SKIP (manifest 不完整): $dir"
        skipped=$((skipped + 1))
        continue
    fi

    if [ -n "$PHASE_FILTER" ] && [ "$phase" != "$PHASE_FILTER" ]; then
        skipped=$((skipped + 1))
        continue
    fi

    grep -E '^\s{4}tag:' "$manifest" | while IFS= read -r tagline; do
        tag=$(echo "$tagline" | cut -d: -f2 | tr -d ' "')
        [ -z "$tag" ] && continue
        version_arg="${tag#v}"
        image="$REGISTRY/$project/$name:$tag"

        if [ -n "$DRY_RUN" ]; then
            echo "  [DRY-RUN] docker build -t $image --build-arg VERSION=$version_arg $dir"
            succeeded=$((succeeded + 1))
            continue
        fi

        echo "  构建: $image"
        if docker build -t "$image" --build-arg VERSION="$version_arg" -f "$dockerfile" "$dir"; then
            echo "  成功: $image"
            succeeded=$((succeeded + 1))
        else
            echo "  失败: $image"
            failed=$((failed + 1))
        fi
    done
done < <(find "$DEPLOY_DIR/images" -name "manifest.yaml" -type f)

# ---------------------------------------------------------------------------
# 阶段 2：构建平台服务镜像（deploy/docker/）
# ---------------------------------------------------------------------------

echo ""
echo "========================================"
echo " 阶段 2：平台服务镜像"
echo "========================================"
echo ""

declare -A SERVICE_IMAGES=(
    ["backend.Dockerfile"]="lenschain/backend:v1.0.0|$REPO_ROOT"
    ["frontend.Dockerfile"]="lenschain/frontend:v1.0.0|$REPO_ROOT"
    ["sim-engine-core.Dockerfile"]="lenschain/sim-engine-core:v1.0.0|$REPO_ROOT/sim-engine/core"
    ["collector-agent.Dockerfile"]="lenschain/collector-agent:v1.0.0|$REPO_ROOT/sim-engine/core"
    ["image-prepuller.Dockerfile"]="lenschain/image-prepuller:v1.0.0|$DEPLOY_DIR"
    ["image-gc.Dockerfile"]="lenschain/image-gc:v1.0.0|$DEPLOY_DIR"
    ["pv-cleanup.Dockerfile"]="lenschain/pv-cleanup:v1.0.0|$DEPLOY_DIR"
    ["scenario-base.Dockerfile"]="lenschain/scenario-base:v1.0.0|$REPO_ROOT/sim-engine/scenarios"
)

for df in "${!SERVICE_IMAGES[@]}"; do
    dockerfile_path="$DEPLOY_DIR/docker/$df"
    IFS='|' read -r image_name context <<< "${SERVICE_IMAGES[$df]}"
    full_image="$REGISTRY/$image_name"

    if [ ! -f "$dockerfile_path" ]; then
        echo "  SKIP (Dockerfile 不存在): $dockerfile_path"
        skipped=$((skipped + 1))
        continue
    fi

    if [ -n "$DRY_RUN" ]; then
        echo "  [DRY-RUN] docker build -t $full_image -f $dockerfile_path $context"
        succeeded=$((succeeded + 1))
        continue
    fi

    echo "  构建: $full_image"
    if docker build -t "$full_image" -f "$dockerfile_path" "$context"; then
        echo "  成功: $full_image"
        succeeded=$((succeeded + 1))
    else
        echo "  失败: $full_image (平台服务源码可能未就绪)"
        failed=$((failed + 1))
    fi
done

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------

echo ""
echo "========================================"
echo " 构建汇总"
echo "========================================"
echo "  成功: $succeeded"
echo "  失败: $failed"
echo "  跳过: $skipped"

if [ "$failed" -gt 0 ]; then
    echo ""
    echo "提示：部分镜像构建失败是正常的："
    echo "  - 平台服务镜像（backend/frontend/sim-engine）需要源码先编译完成"
    echo "  - 自研工具镜像（xterm-server/judge-service）需要对应源码目录"
    echo "  - 部分上游镜像可能因网络原因拉取基础镜像失败，重试即可"
    exit 1
fi
