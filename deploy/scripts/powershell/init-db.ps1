# init-db.ps1
# 初始化数据库：建库 + 执行迁移 + 种子数据
# 用途：Windows / PowerShell 环境下首次部署或清空数据库后执行
# 幂等：可重复执行不出错

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } else { "5442" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } else { "lenschain" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } else { "lenschain" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } else { "lenschain" }

$env:PGPASSWORD = $DB_PASSWORD
$env:LENSCHAIN_DATABASE_HOST = $DB_HOST
$env:LENSCHAIN_DATABASE_PORT = $DB_PORT
$env:LENSCHAIN_DATABASE_USER = $DB_USER
$env:LENSCHAIN_DATABASE_PASSWORD = $DB_PASSWORD
$env:LENSCHAIN_DATABASE_DBNAME = $DB_NAME
$env:LENSCHAIN_DATABASE_SSLMODE = if ($env:LENSCHAIN_DATABASE_SSLMODE) { $env:LENSCHAIN_DATABASE_SSLMODE } else { "disable" }

function Require-Command {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "缺少依赖命令: $Name"
    }
}

Require-Command -Name "pg_isready"
Require-Command -Name "psql"
Require-Command -Name "go"

Write-Host "==> Checking database connection"
while ($true) {
    & pg_isready -h $DB_HOST -p $DB_PORT -U $DB_USER 2>$null | Out-Null
    if ($LASTEXITCODE -eq 0) {
        break
    }

    Write-Host "    Waiting for PostgreSQL..."
    Start-Sleep -Seconds 2
}

Write-Host "==> Recreating database"
$dbExistsOutput = (& psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '$DB_NAME'" | Out-String)
$dbExists = $dbExistsOutput.Trim()
if ($dbExists -eq "1") {
    Write-Host "    Existing database found, terminating connections"
    & psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -v ON_ERROR_STOP=1 -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$DB_NAME' AND pid <> pg_backend_pid();"
    if ($LASTEXITCODE -ne 0) {
        throw "终止数据库连接失败"
    }

    Write-Host "    Dropping database $DB_NAME"
    & psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -v ON_ERROR_STOP=1 -c "DROP DATABASE `"$DB_NAME`""
    if ($LASTEXITCODE -ne 0) {
        throw "删除数据库失败"
    }
}

& psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE `"$DB_NAME`""
if ($LASTEXITCODE -ne 0) {
    throw "创建数据库失败"
}

$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$backendDir = Join-Path $repoRoot "backend"
$seedFile = Join-Path $backendDir "migrations/010_seed_demo_data.up.sql"
$goCacheDir = Join-Path $backendDir ".gocache"

if (-not $env:GOCACHE) {
    if (-not (Test-Path $goCacheDir)) {
        New-Item -ItemType Directory -Path $goCacheDir -Force | Out-Null
    }
    $env:GOCACHE = $goCacheDir
}

Write-Host "==> Running migrations"
Push-Location $backendDir
try {
    & go run cmd/migrate/main.go up
    if ($LASTEXITCODE -ne 0) {
        throw "执行迁移失败"
    }
}
finally {
    Pop-Location
}

if (Test-Path $seedFile) {
    Write-Host "==> Seeding demo data from migrations/010_seed_demo_data.up.sql"
    & psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f $seedFile
    if ($LASTEXITCODE -ne 0) {
        throw "导入 demo 数据失败"
    }
}
else {
    Write-Host "==> Demo seed file not found, skipping"
}

Write-Host "==> Database initialization complete"
