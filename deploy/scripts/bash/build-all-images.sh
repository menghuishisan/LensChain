#!/usr/bin/env bash
# build-all-images.sh
# 本地构建全部项目镜像
# 扫描 deploy/images/**/Dockerfile 和 deploy/docker/*.Dockerfile，统一构建并打上项目 tag
#
# 用法：
#   ./build-all-images.sh                    # 构建全部镜像
#   PHASE_FILTER=1 ./build-all-images.sh     # 仅构建 Phase 1 镜像
#   DRY_RUN=1 ./build-all-images.sh          # 仅打印命令，不执行
#
# 构建策略：
#   - 阶段 0：扫描所有 Dockerfile，提取 FROM 基础镜像并全部预拉取
#     → 任何基础镜像拉取失败则立即停止
#   - 阶段 1：构建实验/比赛镜像（deploy/images/）
#     → 任何镜像构建失败则立即停止，清理悬挂镜像后退出
#   - 阶段 2：构建平台服务镜像（deploy/docker/）
#     → 同上，失败即停
#   - 已构建成功的镜像自动跳过

set -euo pipefail

REGISTRY="${REGISTRY:-registry.lianjing.com}"
PHASE_FILTER="${PHASE_FILTER:-}"
DRY_RUN="${DRY_RUN:-}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
DEPLOY_DIR="$REPO_ROOT/deploy"
succeeded=0
skipped=0

command -v docker >/dev/null 2>&1 || { echo "缺少依赖命令: docker"; exit 1; }

# ---------------------------------------------------------------------------
# 工具函数：docker build（一次构建，失败即停，不重试）
# 构建后用 docker image inspect 二次验证镜像是否真实存在
# ---------------------------------------------------------------------------
build_image() {
    local image_tag="$1"
    shift
    if docker build "$@"; then
        if docker image inspect "$image_tag" > /dev/null 2>&1; then
            return 0
        fi
        echo "    警告：docker build 返回成功但镜像不存在，视为失败"
    fi
    return 1
}

# ---------------------------------------------------------------------------
# 工具函数：清理悬挂镜像和构建缓存
# ---------------------------------------------------------------------------
cleanup_dangling() {
    echo ""
    echo "  清理悬挂镜像..."
    local dangling
    dangling=$(docker images -f "dangling=true" -q 2>/dev/null || true)
    if [ -n "$dangling" ]; then
        echo "$dangling" | xargs docker rmi 2>/dev/null || true
        echo "  已清理悬挂镜像"
    else
        echo "  无悬挂镜像需要清理"
    fi
    echo "  清理构建缓存..."
    docker builder prune -f > /dev/null 2>&1 || true
    echo "  构建缓存已清理"
}

# ---------------------------------------------------------------------------
# 工具函数：失败时打印汇总并退出
# ---------------------------------------------------------------------------
exit_on_failure() {
    local failed_image="$1"
    echo ""
    echo "========================================"
    echo " 构建中断"
    echo "========================================"
    echo "  失败镜像: $failed_image"
    echo "  已成功: $succeeded"
    echo "  已跳过: $skipped"
    cleanup_dangling
    echo ""
    echo "  请修复问题后重新运行脚本。已成功构建的镜像不会重复构建（Docker 缓存生效）。"
    exit 1
}

# ---------------------------------------------------------------------------
# 阶段 0：预拉取所有基础镜像（从 Dockerfile 的 FROM 指令提取）
# ---------------------------------------------------------------------------

echo ""
echo "========================================"
echo " 阶段 0：预拉取基础镜像"
echo "========================================"
echo ""

# 收集所有 Dockerfile 中的 FROM 基础镜像（去重、排除动态变量和构建阶段引用）
base_images=()
while IFS= read -r line; do
    img=$(echo "$line" | sed -E 's/^\s*FROM\s+//; s/\s+AS\s+.*$//i' | tr -d ' ')
    # 跳过包含 ${...} 的动态镜像和构建阶段引用
    case "$img" in
        *'${'*|builder|deps|foundry-builder|node-builder) continue ;;
    esac
    base_images+=("$img")
done < <(grep -rihE '^\s*FROM\s+' "$DEPLOY_DIR/images" "$DEPLOY_DIR/docker" --include="Dockerfile" --include="*.Dockerfile" 2>/dev/null)

# 去重并排序
mapfile -t base_images < <(printf "%s\n" "${base_images[@]}" | sort -u)

echo "  共发现 ${#base_images[@]} 个基础镜像需要预拉取"
echo ""

if [ -z "$DRY_RUN" ]; then
    pull_ok=0
    for img in "${base_images[@]}"; do
        if docker image inspect "$img" > /dev/null 2>&1; then
            echo "  [已缓存] $img"
            pull_ok=$((pull_ok + 1))
            continue
        fi
        echo "  拉取: $img"
        if docker pull "$img"; then
            echo "  成功: $img"
            pull_ok=$((pull_ok + 1))
        else
            echo ""
            echo "  致命错误：基础镜像 $img 拉取失败"
            echo "  请检查网络/代理设置后重新运行脚本"
            cleanup_dangling
            exit 1
        fi
    done
    echo ""
    echo "  预拉取全部完成：$pull_ok 个基础镜像就绪"
else
    for img in "${base_images[@]}"; do
        echo "  [DRY-RUN] docker pull $img"
    done
fi

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

    # 提取所有版本 tag 及其对应的 upstream_tag
    mapfile -t tag_arr < <(grep -E '^\s{4}tag:' "$manifest")
    mapfile -t upstream_arr < <(grep -E '^\s{4}upstream_tag:' "$manifest")
    for i in "${!tag_arr[@]}"; do
        tag=$(echo "${tag_arr[$i]}" | cut -d: -f2 | tr -d ' "')
        [ -z "$tag" ] && continue

        # 使用 upstream_tag 作为 VERSION（上游镜像的真实 tag）
        version_arg="${tag#v}"
        if [ "$i" -lt "${#upstream_arr[@]}" ]; then
            upstream_tag=$(echo "${upstream_arr[$i]}" | cut -d: -f2 | tr -d ' "')
            [ -n "$upstream_tag" ] && version_arg="$upstream_tag"
        fi
        image="$REGISTRY/$project/$name:$tag"

        if [ -n "$DRY_RUN" ]; then
            echo "  [DRY-RUN] docker build -t $image --build-arg VERSION=$version_arg $dir"
            succeeded=$((succeeded + 1))
            continue
        fi

        # 跳过已构建的镜像
        if docker image inspect "$image" > /dev/null 2>&1; then
            echo "  [已构建] $image"
            skipped=$((skipped + 1))
            continue
        fi

        echo "  构建: $image"
        if build_image "$image" -t "$image" --build-arg VERSION="$version_arg" -f "$dockerfile" "$dir"; then
            echo "  成功: $image"
            succeeded=$((succeeded + 1))
        else
            exit_on_failure "$image"
        fi
    done
done < <(find "$DEPLOY_DIR/images" -name "manifest.yaml" -type f | sort)

# ---------------------------------------------------------------------------
# 阶段 2：构建平台服务镜像（deploy/docker/）
# ---------------------------------------------------------------------------

echo ""
echo "========================================"
echo " 阶段 2：平台服务镜像"
echo "========================================"
echo ""

# 格式: "Dockerfile|image_name|context|extra_build_args"
SERVICE_IMAGES=(
    "backend.Dockerfile|lenschain/backend:v1.0.0|$REPO_ROOT|"
    "frontend.Dockerfile|lenschain/frontend:v1.0.0|$REPO_ROOT|"
    "sim-engine-core.Dockerfile|lenschain/sim-engine-core:v1.0.0|$REPO_ROOT/sim-engine|"
    "collector-agent.Dockerfile|lenschain/collector-agent-ethereum:v1.0.0|$REPO_ROOT/sim-engine|--build-arg ADAPTER=ethereum"
    "collector-agent.Dockerfile|lenschain/collector-agent-fabric:v1.0.0|$REPO_ROOT/sim-engine|--build-arg ADAPTER=fabric"
    "collector-agent.Dockerfile|lenschain/collector-agent-chainmaker:v1.0.0|$REPO_ROOT/sim-engine|--build-arg ADAPTER=chainmaker"
    "collector-agent.Dockerfile|lenschain/collector-agent-fisco:v1.0.0|$REPO_ROOT/sim-engine|--build-arg ADAPTER=fisco"
    "image-prepuller.Dockerfile|lenschain/image-prepuller:v1.0.0|$REPO_ROOT|"
    "image-gc.Dockerfile|lenschain/image-gc:v1.0.0|$REPO_ROOT|"
    "pv-cleanup.Dockerfile|lenschain/pv-cleanup:v1.0.0|$REPO_ROOT|"
    "scenario-base.Dockerfile|lenschain/scenario-base:v1.0.0|$REPO_ROOT/sim-engine/scenarios|"
)

for entry in "${SERVICE_IMAGES[@]}"; do
    IFS='|' read -r df image_name context extra_args <<< "$entry"
    dockerfile_path="$DEPLOY_DIR/docker/$df"
    full_image="$REGISTRY/$image_name"

    if [ ! -f "$dockerfile_path" ]; then
        echo "  SKIP (Dockerfile 不存在): $dockerfile_path"
        skipped=$((skipped + 1))
        continue
    fi

    if [ -n "$DRY_RUN" ]; then
        echo "  [DRY-RUN] docker build -t $full_image $extra_args -f $dockerfile_path $context"
        succeeded=$((succeeded + 1))
        continue
    fi

    # 跳过已构建的镜像
    if docker image inspect "$full_image" > /dev/null 2>&1; then
        echo "  [已构建] $full_image"
        skipped=$((skipped + 1))
        continue
    fi

    echo "  构建: $full_image"
    # shellcheck disable=SC2086
    if build_image "$full_image" -t "$full_image" $extra_args -f "$dockerfile_path" "$context"; then
        echo "  成功: $full_image"
        succeeded=$((succeeded + 1))
    else
        exit_on_failure "$full_image"
    fi
done

# ---------------------------------------------------------------------------
# 全部成功 - 汇总
# ---------------------------------------------------------------------------

echo ""
echo "========================================"
echo " 构建全部完成"
echo "========================================"
echo "  成功: $succeeded"
echo "  跳过: $skipped"

# 最终清理悬挂镜像（正常构建也可能产生）
cleanup_dangling
