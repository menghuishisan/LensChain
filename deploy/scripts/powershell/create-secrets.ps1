# deploy/scripts/powershell/create-secrets.ps1
# 从 deploy/config.env 读取密码，创建 K8s Secret
# 用法：.\deploy\scripts\powershell\create-secrets.ps1
# 说明：所有密码统一从 config.env 获取，不硬编码

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

Write-Host "[create-secrets] 加载配置文件: $EnvFile" -ForegroundColor Cyan

$envVars = @{}
Get-Content $EnvFile | ForEach-Object {
    $line = $_.Trim()
    if ($line -and -not $line.StartsWith("#")) {
        $parts = $line -split "=", 2
        if ($parts.Length -eq 2) {
            $envVars[$parts[0].Trim()] = $parts[1].Trim()
        }
    }
}

# ---------------------------------------------------------------------------
# 校验必填字段
# ---------------------------------------------------------------------------

$requiredKeys = @(
    "BACKEND_DATABASE_PASSWORD",
    "BACKEND_JWT_ACCESS_SECRET",
    "BACKEND_JWT_REFRESH_SECRET",
    "BACKEND_REDIS_PASSWORD",
    "BACKEND_SNAPSHOT_ENCRYPTION_KEY",
    "POSTGRES_PASSWORD",
    "REDIS_PASSWORD",
    "MINIO_ROOT_USER",
    "MINIO_ROOT_PASSWORD"
)

$missing = @()
foreach ($key in $requiredKeys) {
    if (-not $envVars.ContainsKey($key) -or $envVars[$key] -eq "CHANGE_ME" -or [string]::IsNullOrWhiteSpace($envVars[$key])) {
        $missing += $key
    }
}

if ($missing.Count -gt 0) {
    Write-Error "以下字段未填写或仍为 CHANGE_ME:`n$($missing -join "`n")"
    exit 1
}

# ---------------------------------------------------------------------------
# 读取命名空间
# ---------------------------------------------------------------------------

$namespace = if ($envVars.ContainsKey("K8S_NAMESPACE")) { $envVars["K8S_NAMESPACE"] } else { "lenschain" }

Write-Host "[create-secrets] 目标命名空间: $namespace" -ForegroundColor Cyan

# ---------------------------------------------------------------------------
# 创建 Secret
# ---------------------------------------------------------------------------

# 1. backend-secret
Write-Host "[create-secrets] 创建 backend-secret ..." -ForegroundColor Yellow
kubectl -n $namespace create secret generic backend-secret `
    --from-literal=database-password="$($envVars['BACKEND_DATABASE_PASSWORD'])" `
    --from-literal=jwt-access-secret="$($envVars['BACKEND_JWT_ACCESS_SECRET'])" `
    --from-literal=jwt-refresh-secret="$($envVars['BACKEND_JWT_REFRESH_SECRET'])" `
    --from-literal=redis-password="$($envVars['BACKEND_REDIS_PASSWORD'])" `
    --from-literal=snapshot-encryption-key="$($envVars['BACKEND_SNAPSHOT_ENCRYPTION_KEY'])" `
    --dry-run=client -o yaml | kubectl apply -f -

# 2. postgres-secret
Write-Host "[create-secrets] 创建 postgres-secret ..." -ForegroundColor Yellow
kubectl -n $namespace create secret generic postgres-secret `
    --from-literal=password="$($envVars['POSTGRES_PASSWORD'])" `
    --dry-run=client -o yaml | kubectl apply -f -

# 3. redis-secret
Write-Host "[create-secrets] 创建 redis-secret ..." -ForegroundColor Yellow
kubectl -n $namespace create secret generic redis-secret `
    --from-literal=password="$($envVars['REDIS_PASSWORD'])" `
    --dry-run=client -o yaml | kubectl apply -f -

# 4. minio-secret
Write-Host "[create-secrets] 创建 minio-secret ..." -ForegroundColor Yellow
kubectl -n $namespace create secret generic minio-secret `
    --from-literal=root-user="$($envVars['MINIO_ROOT_USER'])" `
    --from-literal=root-password="$($envVars['MINIO_ROOT_PASSWORD'])" `
    --dry-run=client -o yaml | kubectl apply -f -

# 5. registry-pull-secret（仅当配置了 REGISTRY_USER 和 REGISTRY_PASSWORD 时创建）
if ($envVars.ContainsKey("REGISTRY_USER") -and $envVars["REGISTRY_USER"] -ne "CHANGE_ME" `
    -and $envVars.ContainsKey("REGISTRY_PASSWORD") -and $envVars["REGISTRY_PASSWORD"] -ne "CHANGE_ME") {
    $server = if ($envVars.ContainsKey("REGISTRY_SERVER")) { $envVars["REGISTRY_SERVER"] } else { "registry.lianjing.com" }
    Write-Host "[create-secrets] 创建 registry-pull-secret ..." -ForegroundColor Yellow
    kubectl -n $namespace create secret docker-registry registry-pull-secret `
        --docker-server="$server" `
        --docker-username="$($envVars['REGISTRY_USER'])" `
        --docker-password="$($envVars['REGISTRY_PASSWORD'])" `
        --dry-run=client -o yaml | kubectl apply -f -
} else {
    Write-Host "[create-secrets] 跳过 registry-pull-secret（REGISTRY_USER/REGISTRY_PASSWORD 未配置）" -ForegroundColor DarkGray
}

Write-Host "[create-secrets] 全部完成" -ForegroundColor Green
