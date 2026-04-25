# docker-prepull.ps1
# 本地 Docker 镜像预拉取脚本
# 用途：扫描 deploy/images/**/manifest.yaml，并使用 docker pull 拉取镜像到当前开发机

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Registry = if ($env:REGISTRY) { $env:REGISTRY } else { "registry.lianjing.com" }
$PhaseFilter = if ($env:PHASE_FILTER) { $env:PHASE_FILTER } else { "" }
$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$deployDir = Join-Path $repoRoot "deploy"
$count = 0
$failed = 0

function Require-Command {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "缺少依赖命令: $Name"
    }
}

Require-Command -Name "docker"

Write-Host "==> Collecting images from $deployDir/images/"
$manifests = Get-ChildItem -Path (Join-Path $deployDir "images") -Filter "manifest.yaml" -Recurse -File
if ($manifests.Count -eq 0) {
    throw "no manifest.yaml files found under $deployDir/images/"
}

foreach ($manifest in $manifests) {
    $content = Get-Content $manifest.FullName
    $projectLine = $content | Where-Object { $_ -match '^registry_project:' } | Select-Object -First 1
    $nameLine = $content | Where-Object { $_ -match '^name:' } | Select-Object -First 1
    $phaseLine = $content | Where-Object { $_ -match '^prepull_phase:' } | Select-Object -First 1

    if (-not $projectLine -or -not $nameLine) {
        Write-Host "WARN: skip invalid manifest $($manifest.FullName)"
        continue
    }

    $project = ($projectLine -split ':', 2)[1].Trim().Trim('"')
    $name = ($nameLine -split ':', 2)[1].Trim().Trim('"')
    $phase = ""
    if ($phaseLine) {
        $phase = ($phaseLine -split ':', 2)[1].Trim().Trim('"')
    }

    if ($PhaseFilter -and $phase -ne $PhaseFilter) {
        continue
    }

    $tagLines = $content | Where-Object { $_ -match '^\s{4}tag:' }
    foreach ($tagLine in $tagLines) {
        $tag = ($tagLine -split ':', 2)[1].Trim().Trim('"')
        if (-not $tag) {
            continue
        }

        $image = "$Registry/$project/$name`:$tag"
        Write-Host "==> Pulling $image"
        & docker pull $image
        if ($LASTEXITCODE -eq 0) {
            $count++
        }
        else {
            Write-Host "FAILED: $image"
            $failed++
        }
    }
}

Write-Host "==> Pull complete: $count succeeded, $failed failed"
if ($failed -gt 0) {
    exit 1
}
