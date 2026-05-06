# init-minio.ps1
# 创建默认 MinIO Bucket + 权限策略
# 密码来源：统一从 deploy/config.env 读取，不硬编码

param(
    [string]$EnvFile = "$PSScriptRoot\..\..\config.env"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# 加载 config.env
# ---------------------------------------------------------------------------

if (-not (Test-Path $EnvFile)) {
    Write-Error "找不到配置文件: $EnvFile`n请先将 config.example.env 复制为 config.env 并填入真实值"
    exit 1
}

$envVars = @{}
Get-Content $EnvFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -and -not $line.StartsWith('#')) {
        $parts = $line -split '=', 2
        if ($parts.Count -eq 2) {
            $envVars[$parts[0].Trim()] = $parts[1].Trim()
        }
    }
}

$MinioHost = if ($env:MINIO_HOST) { $env:MINIO_HOST } else { "localhost" }
$MinioPort = if ($env:MINIO_PORT) { $env:MINIO_PORT } else { "9010" }
$MinioUser = if ($env:MINIO_ROOT_USER) { $env:MINIO_ROOT_USER } elseif ($envVars.ContainsKey("MINIO_ROOT_USER")) { $envVars["MINIO_ROOT_USER"] } else { throw "MINIO_ROOT_USER 未设置，请检查 config.env" }
$MinioPassword = if ($env:MINIO_ROOT_PASSWORD) { $env:MINIO_ROOT_PASSWORD } elseif ($envVars.ContainsKey("MINIO_ROOT_PASSWORD")) { $envVars["MINIO_ROOT_PASSWORD"] } else { throw "MINIO_ROOT_PASSWORD 未设置，请检查 config.env" }
$buckets = @("attachments", "reports", "snapshots", "images", "exports", "backups")

function Require-Command {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "缺少依赖命令: $Name"
    }
}

Require-Command -Name "mc"

Write-Host "==> Configuring MinIO client alias"
& mc alias set local "http://${MinioHost}:${MinioPort}" $MinioUser $MinioPassword
if ($LASTEXITCODE -ne 0) {
    throw "配置 MinIO 客户端失败"
}

foreach ($bucket in $buckets) {
    Write-Host "==> Creating bucket: lenschain-$bucket"
    & mc mb --ignore-existing "local/lenschain-$bucket"
    if ($LASTEXITCODE -ne 0) {
        throw "创建 Bucket 失败: lenschain-$bucket"
    }
    & mc anonymous set none "local/lenschain-$bucket"
    if ($LASTEXITCODE -ne 0) {
        throw "设置 Bucket 权限失败: lenschain-$bucket"
    }
}

Write-Host "==> MinIO initialization complete"
