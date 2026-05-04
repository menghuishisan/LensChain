# build-all-images.ps1
# 本地构建全部项目镜像
# 扫描 deploy/images/**/Dockerfile 和 deploy/docker/*.Dockerfile，统一构建并打上项目 tag
#
# 用法：
#   .\build-all-images.ps1                    # 构建全部镜像
#   $env:PHASE_FILTER="1"; .\build-all-images.ps1  # 仅构建 Phase 1 镜像
#   $env:DRY_RUN="1"; .\build-all-images.ps1       # 仅打印命令，不执行
#
# 说明：
#   - 镜像按 manifest.yaml 中的 registry_project/name:tag 命名
#   - Docker Desktop K8s 共享 Docker 镜像存储，构建完成后 K8s Pod 可直接使用
#   - imagePullPolicy 需设为 IfNotPresent 或 Never（base 层默认已配置 IfNotPresent）

Set-StrictMode -Version Latest
$ErrorActionPreference = "Continue"

$Registry = if ($env:REGISTRY) { $env:REGISTRY } else { "registry.lianjing.com" }
$PhaseFilter = if ($env:PHASE_FILTER) { $env:PHASE_FILTER } else { "" }
$DryRun = $env:DRY_RUN -eq "1"
$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$deployDir = Join-Path $repoRoot "deploy"
$succeeded = 0
$failed = 0
$skipped = 0

if (-not (Get-Command "docker" -ErrorAction SilentlyContinue)) {
    throw "缺少依赖命令: docker"
}

# ---------------------------------------------------------------------------
# 阶段 1：构建实验/比赛镜像（deploy/images/）
# ---------------------------------------------------------------------------

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host " 阶段 1：实验/比赛镜像" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

$manifests = Get-ChildItem -Path (Join-Path $deployDir "images") -Filter "manifest.yaml" -Recurse -File
foreach ($manifest in $manifests) {
    $dir = $manifest.DirectoryName
    $dockerfile = Join-Path $dir "Dockerfile"
    if (-not (Test-Path $dockerfile)) {
        Write-Host "  SKIP (无 Dockerfile): $dir" -ForegroundColor DarkGray
        $skipped++
        continue
    }

    $content = Get-Content $manifest.FullName
    $projectLine = $content | Where-Object { $_ -match '^registry_project:' } | Select-Object -First 1
    $nameLine = $content | Where-Object { $_ -match '^name:' } | Select-Object -First 1
    $phaseLine = $content | Where-Object { $_ -match '^prepull_phase:' } | Select-Object -First 1

    if (-not $projectLine -or -not $nameLine) {
        Write-Host "  SKIP (manifest 不完整): $dir" -ForegroundColor DarkGray
        $skipped++
        continue
    }

    $project = ($projectLine -split ':', 2)[1].Trim().Trim('"')
    $name = ($nameLine -split ':', 2)[1].Trim().Trim('"')
    $phase = ""
    if ($phaseLine) {
        $phase = ($phaseLine -split ':', 2)[1].Trim().Trim('"')
    }

    if ($PhaseFilter -and $phase -ne $PhaseFilter) {
        $skipped++
        continue
    }

    # 提取所有版本 tag
    $tagLines = $content | Where-Object { $_ -match '^\s{4}tag:' }
    foreach ($tagLine in $tagLines) {
        $tag = ($tagLine -split ':', 2)[1].Trim().Trim('"')
        if (-not $tag) { continue }

        # 提取对应的 upstream_tag（用于 --build-arg VERSION）
        $versionArg = $tag -replace '^v', ''

        $image = "$Registry/$project/${name}:$tag"
        if ($DryRun) {
            Write-Host "  [DRY-RUN] docker build -t $image --build-arg VERSION=$versionArg $dir"
            $succeeded++
            continue
        }

        Write-Host "  构建: $image" -ForegroundColor Yellow
        & docker build -t $image --build-arg VERSION=$versionArg -f $dockerfile $dir 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  成功: $image" -ForegroundColor Green
            $succeeded++
        }
        else {
            Write-Host "  失败: $image" -ForegroundColor Red
            $failed++
        }
    }
}

# ---------------------------------------------------------------------------
# 阶段 2：构建平台服务镜像（deploy/docker/）
# ---------------------------------------------------------------------------

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host " 阶段 2：平台服务镜像" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

$serviceImages = @(
    @{ Dockerfile = "backend.Dockerfile";         Image = "lenschain/backend:v1.0.0";             Context = $repoRoot },
    @{ Dockerfile = "frontend.Dockerfile";        Image = "lenschain/frontend:v1.0.0";            Context = $repoRoot },
    @{ Dockerfile = "sim-engine-core.Dockerfile";  Image = "lenschain/sim-engine-core:v1.0.0";     Context = (Join-Path $repoRoot "sim-engine/core") },
    @{ Dockerfile = "collector-agent.Dockerfile";  Image = "lenschain/collector-agent:v1.0.0";     Context = (Join-Path $repoRoot "sim-engine/core") },
    @{ Dockerfile = "image-prepuller.Dockerfile";  Image = "lenschain/image-prepuller:v1.0.0";     Context = $deployDir },
    @{ Dockerfile = "image-gc.Dockerfile";         Image = "lenschain/image-gc:v1.0.0";            Context = $deployDir },
    @{ Dockerfile = "pv-cleanup.Dockerfile";       Image = "lenschain/pv-cleanup:v1.0.0";          Context = $deployDir },
    @{ Dockerfile = "scenario-base.Dockerfile";    Image = "lenschain/scenario-base:v1.0.0";       Context = (Join-Path $repoRoot "sim-engine/scenarios") }
)

foreach ($svc in $serviceImages) {
    $dockerfilePath = Join-Path $deployDir "docker" $svc.Dockerfile
    $fullImage = "$Registry/$($svc.Image)"

    if (-not (Test-Path $dockerfilePath)) {
        Write-Host "  SKIP (Dockerfile 不存在): $dockerfilePath" -ForegroundColor DarkGray
        $skipped++
        continue
    }

    if ($DryRun) {
        Write-Host "  [DRY-RUN] docker build -t $fullImage -f $dockerfilePath $($svc.Context)"
        $succeeded++
        continue
    }

    Write-Host "  构建: $fullImage" -ForegroundColor Yellow
    & docker build -t $fullImage -f $dockerfilePath $svc.Context 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  成功: $fullImage" -ForegroundColor Green
        $succeeded++
    }
    else {
        Write-Host "  失败: $fullImage (平台服务源码可能未就绪，后续再构建)" -ForegroundColor Red
        $failed++
    }
}

# ---------------------------------------------------------------------------
# 汇总
# ---------------------------------------------------------------------------

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host " 构建汇总" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  成功: $succeeded" -ForegroundColor Green
Write-Host "  失败: $failed" -ForegroundColor $(if ($failed -gt 0) { "Red" } else { "Green" })
Write-Host "  跳过: $skipped" -ForegroundColor DarkGray

if ($failed -gt 0) {
    Write-Host "`n提示：部分镜像构建失败是正常的：" -ForegroundColor Yellow
    Write-Host "  - 平台服务镜像（backend/frontend/sim-engine）需要源码先编译完成" -ForegroundColor Yellow
    Write-Host "  - 自研工具镜像（xterm-server/judge-service）需要对应源码目录" -ForegroundColor Yellow
    Write-Host "  - 部分上游镜像可能因网络原因拉取基础镜像失败，重试即可" -ForegroundColor Yellow
    exit 1
}
