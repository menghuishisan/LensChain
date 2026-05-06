# build-all-images.ps1
# 本地构建全部项目镜像
# 扫描 deploy/images/**/Dockerfile 和 deploy/docker/*.Dockerfile，统一构建并打上项目 tag
#
# 用法：
#   .\build-all-images.ps1                    # 构建全部镜像
#   $env:PHASE_FILTER="1"; .\build-all-images.ps1  # 仅构建 Phase 1 镜像
#   $env:DRY_RUN="1"; .\build-all-images.ps1       # 仅打印命令，不执行
#
# 构建策略：
#   - 阶段 0：扫描所有 Dockerfile，提取 FROM 基础镜像并全部预拉取
#     → 任何基础镜像拉取失败则立即停止
#   - 阶段 1：构建实验/比赛镜像（deploy/images/）
#     → 任何镜像构建失败则立即停止，清理悬挂镜像后退出
#   - 阶段 2：构建平台服务镜像（deploy/docker/）
#     → 同上，失败即停
#   - 已构建成功的镜像自动跳过（通过 docker image inspect 检测）

Set-StrictMode -Version Latest
$ErrorActionPreference = "Continue"

$Registry = if ($env:REGISTRY) { $env:REGISTRY } else { "registry.lianjing.com" }
$PhaseFilter = if ($env:PHASE_FILTER) { $env:PHASE_FILTER } else { "" }
$DryRun = $env:DRY_RUN -eq "1"
$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$deployDir = Join-Path $repoRoot "deploy"
$succeeded = 0
$skipped = 0

if (-not (Get-Command "docker" -ErrorAction SilentlyContinue)) {
    throw "缺少依赖命令: docker"
}

# ---------------------------------------------------------------------------
# 工具函数：docker build（一次构建，失败即停，不重试）
# 使用 Start-Process 获取真实退出码，避免 PowerShell 2>&1 导致 $LASTEXITCODE 不可靠
# 构建后用 docker image inspect 二次验证镜像是否真实存在
# ---------------------------------------------------------------------------
function Build-Image {
    param([string]$ImageTag, [string[]]$BuildCmd)
    $proc = Start-Process -FilePath "docker" -ArgumentList $BuildCmd -NoNewWindow -Wait -PassThru
    if ($proc.ExitCode -ne 0) {
        return $false
    }
    # 二次验证：确认镜像真实存在
    docker image inspect $ImageTag *> $null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "    警告：docker build 返回成功但镜像不存在，视为失败" -ForegroundColor Red
        return $false
    }
    return $true
}

# ---------------------------------------------------------------------------
# 工具函数：清理构建失败产生的悬挂镜像和构建缓存
# ---------------------------------------------------------------------------
function Cleanup-Dangling {
    Write-Host "`n  清理悬挂镜像..." -ForegroundColor Yellow
    $dangling = docker images -f "dangling=true" -q 2>$null
    if ($dangling -and $LASTEXITCODE -eq 0) {
        $dangling | ForEach-Object { docker rmi $_ *> $null }
        Write-Host "  已清理悬挂镜像" -ForegroundColor Green
    } else {
        Write-Host "  无悬挂镜像需要清理" -ForegroundColor DarkGray
    }
    Write-Host "  清理构建缓存..." -ForegroundColor Yellow
    docker builder prune -f *> $null
    Write-Host "  构建缓存已清理" -ForegroundColor Green
}

# ---------------------------------------------------------------------------
# 工具函数：失败时打印汇总并退出
# ---------------------------------------------------------------------------
function Exit-OnFailure {
    param([string]$FailedImage)
    Write-Host "`n========================================" -ForegroundColor Red
    Write-Host " 构建中断" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    Write-Host "  失败镜像: $FailedImage" -ForegroundColor Red
    Write-Host "  已成功: $succeeded" -ForegroundColor Green
    Write-Host "  已跳过: $skipped" -ForegroundColor DarkGray
    Cleanup-Dangling
    Write-Host "`n  请修复问题后重新运行脚本。已成功构建的镜像不会重复构建（Docker 缓存生效）。" -ForegroundColor Yellow
    exit 1
}

# ---------------------------------------------------------------------------
# 阶段 0：预拉取所有基础镜像（从 Dockerfile 的 FROM 指令提取）
# ---------------------------------------------------------------------------

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host " 阶段 0：预拉取基础镜像" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

$allDockerfiles = @()
$allDockerfiles += Get-ChildItem -Path (Join-Path $deployDir "images") -Filter "Dockerfile" -Recurse -File
$allDockerfiles += Get-ChildItem -Path (Join-Path $deployDir "docker") -Filter "*.Dockerfile" -File

$baseImages = @{}
foreach ($df in $allDockerfiles) {
    $lines = Get-Content $df.FullName
    foreach ($line in $lines) {
        if ($line -match '^\s*FROM\s+(.+?)(\s+AS\s+.+)?$') {
            $img = $Matches[1].Trim()
            # 跳过 ARG 引用的动态镜像（如 ${VERSION}）和构建阶段引用
            if ($img -notmatch '\$\{' -and $img -notmatch '^(builder|deps|foundry-builder|node-builder)$') {
                $baseImages[$img] = $true
            }
        }
    }
}

$baseList = $baseImages.Keys | Sort-Object
Write-Host "  共发现 $($baseList.Count) 个基础镜像需要预拉取`n" -ForegroundColor White

if (-not $DryRun) {
    $pullOk = 0
    foreach ($img in $baseList) {
        # 检查本地是否已有
        docker image inspect $img *> $null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  [已缓存] $img" -ForegroundColor DarkGray
            $pullOk++
            continue
        }
        Write-Host "  拉取: $img" -ForegroundColor Yellow
        docker pull $img
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  成功: $img" -ForegroundColor Green
            $pullOk++
        } else {
            Write-Host "`n  致命错误：基础镜像 $img 拉取失败" -ForegroundColor Red
            Write-Host "  请检查网络/代理设置后重新运行脚本" -ForegroundColor Red
            Cleanup-Dangling
            exit 1
        }
    }
    Write-Host "`n  预拉取全部完成：$pullOk 个基础镜像就绪" -ForegroundColor Green
} else {
    foreach ($img in $baseList) {
        Write-Host "  [DRY-RUN] docker pull $img"
    }
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

    # 提取所有版本 tag 及其对应的 upstream_tag（强制转为数组避免单元素时 .Count 失效）
    [array]$tagLines = $content | Where-Object { $_ -match '^\s{4}tag:' }
    [array]$upstreamLines = $content | Where-Object { $_ -match '^\s{4}upstream_tag:' }
    for ($i = 0; $i -lt $tagLines.Count; $i++) {
        $tag = ($tagLines[$i] -split ':', 2)[1].Trim().Trim('"')
        if (-not $tag) { continue }

        # 使用 upstream_tag 作为 VERSION（上游镜像的真实 tag）
        $versionArg = $tag -replace '^v', ''
        if ($i -lt $upstreamLines.Count) {
            $upstreamTag = ($upstreamLines[$i] -split ':', 2)[1].Trim().Trim('"')
            if ($upstreamTag) { $versionArg = $upstreamTag }
        }

        $image = "$Registry/$project/${name}:$tag"
        if ($DryRun) {
            Write-Host "  [DRY-RUN] docker build -t $image --build-arg VERSION=$versionArg $dir"
            $succeeded++
            continue
        }

        # 跳过已构建的镜像
        docker image inspect $image *> $null
        if ($LASTEXITCODE -eq 0) {
            Write-Host "  [已构建] $image" -ForegroundColor DarkGray
            $skipped++
            continue
        }

        Write-Host "  构建: $image" -ForegroundColor Yellow
        $buildCmd = @("build", "-t", $image, "--build-arg", "VERSION=$versionArg", "-f", $dockerfile, $dir)
        if (Build-Image -ImageTag $image -BuildCmd $buildCmd) {
            Write-Host "  成功: $image" -ForegroundColor Green
            $succeeded++
        }
        else {
            Exit-OnFailure -FailedImage $image
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
    @{ Dockerfile = "sim-engine-core.Dockerfile";  Image = "lenschain/sim-engine-core:v1.0.0";     Context = (Join-Path $repoRoot "sim-engine") },
    @{ Dockerfile = "collector-agent.Dockerfile";  Image = "lenschain/collector-agent-ethereum:v1.0.0";   Context = (Join-Path $repoRoot "sim-engine"); ExtraArgs = @("ADAPTER=ethereum") },
    @{ Dockerfile = "collector-agent.Dockerfile";  Image = "lenschain/collector-agent-fabric:v1.0.0";    Context = (Join-Path $repoRoot "sim-engine"); ExtraArgs = @("ADAPTER=fabric") },
    @{ Dockerfile = "collector-agent.Dockerfile";  Image = "lenschain/collector-agent-chainmaker:v1.0.0"; Context = (Join-Path $repoRoot "sim-engine"); ExtraArgs = @("ADAPTER=chainmaker") },
    @{ Dockerfile = "collector-agent.Dockerfile";  Image = "lenschain/collector-agent-fisco:v1.0.0";     Context = (Join-Path $repoRoot "sim-engine"); ExtraArgs = @("ADAPTER=fisco") },
    @{ Dockerfile = "image-prepuller.Dockerfile";  Image = "lenschain/image-prepuller:v1.0.0";     Context = $repoRoot },
    @{ Dockerfile = "image-gc.Dockerfile";         Image = "lenschain/image-gc:v1.0.0";            Context = $repoRoot },
    @{ Dockerfile = "pv-cleanup.Dockerfile";       Image = "lenschain/pv-cleanup:v1.0.0";          Context = $repoRoot },
    @{ Dockerfile = "scenario-base.Dockerfile";    Image = "lenschain/scenario-base:v1.0.0";       Context = (Join-Path (Join-Path $repoRoot "sim-engine") "scenarios") }
)

foreach ($svc in $serviceImages) {
    $dockerfilePath = Join-Path (Join-Path $deployDir "docker") $svc.Dockerfile
    $fullImage = "$Registry/$($svc.Image)"

    if (-not (Test-Path $dockerfilePath)) {
        Write-Host "  SKIP (Dockerfile 不存在): $dockerfilePath" -ForegroundColor DarkGray
        $skipped++
        continue
    }

    $extraBuildArgs = @()
    if ($svc.ContainsKey('ExtraArgs') -and $svc.ExtraArgs) {
        foreach ($arg in $svc.ExtraArgs) {
            $extraBuildArgs += "--build-arg"
            $extraBuildArgs += $arg
        }
    }

    if ($DryRun) {
        Write-Host "  [DRY-RUN] docker build -t $fullImage $($extraBuildArgs -join ' ') -f $dockerfilePath $($svc.Context)"
        $succeeded++
        continue
    }

    # 跳过已构建的镜像
    docker image inspect $fullImage *> $null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "  [已构建] $fullImage" -ForegroundColor DarkGray
        $skipped++
        continue
    }

    Write-Host "  构建: $fullImage" -ForegroundColor Yellow
    $buildCmd = @("build", "-t", $fullImage) + $extraBuildArgs + @("-f", $dockerfilePath, $svc.Context)
    if (Build-Image -ImageTag $fullImage -BuildCmd $buildCmd) {
        Write-Host "  成功: $fullImage" -ForegroundColor Green
        $succeeded++
    }
    else {
        Exit-OnFailure -FailedImage $fullImage
    }
}

# ---------------------------------------------------------------------------
# 全部成功 - 汇总
# ---------------------------------------------------------------------------

Write-Host "`n========================================" -ForegroundColor Green
Write-Host " 构建全部完成" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "  成功: $succeeded" -ForegroundColor Green
Write-Host "  跳过: $skipped" -ForegroundColor DarkGray

# 最终清理悬挂镜像（正常构建也可能产生）
Cleanup-Dangling
