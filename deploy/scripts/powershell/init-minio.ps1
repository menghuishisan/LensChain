# init-minio.ps1
# 创建默认 MinIO Bucket + 权限策略

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$MinioHost = if ($env:MINIO_HOST) { $env:MINIO_HOST } else { "localhost" }
$MinioPort = if ($env:MINIO_PORT) { $env:MINIO_PORT } else { "9010" }
$MinioUser = if ($env:MINIO_USER) { $env:MINIO_USER } else { "minioadmin" }
$MinioPassword = if ($env:MINIO_PASSWORD) { $env:MINIO_PASSWORD } else { "minioadmin" }
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
