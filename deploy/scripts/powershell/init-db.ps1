# init-db.ps1
# 初始化数据库：建库 + 执行迁移 + 种子数据
# 用途：Windows / PowerShell 环境下首次部署或清空数据库后执行
# 幂等：可重复执行不出错
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

$DB_HOST = if ($env:DB_HOST) { $env:DB_HOST } elseif ($envVars.ContainsKey("DB_HOST")) { $envVars["DB_HOST"] } else { "localhost" }
$DB_PORT = if ($env:DB_PORT) { $env:DB_PORT } elseif ($envVars.ContainsKey("DB_PORT")) { $envVars["DB_PORT"] } else { "5442" }
$DB_USER = if ($env:DB_USER) { $env:DB_USER } elseif ($envVars.ContainsKey("DB_USER")) { $envVars["DB_USER"] } else { "lenschain" }
$DB_PASSWORD = if ($env:DB_PASSWORD) { $env:DB_PASSWORD } elseif ($envVars.ContainsKey("DB_PASSWORD")) { $envVars["DB_PASSWORD"] } else { throw "DB_PASSWORD 未设置，请检查 config.env" }
$DB_NAME = if ($env:DB_NAME) { $env:DB_NAME } elseif ($envVars.ContainsKey("DB_NAME")) { $envVars["DB_NAME"] } else { "lenschain" }

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

$REDIS_DB = if ($env:REDIS_DB) { $env:REDIS_DB } else { "0" }
$REDIS_CONTAINER = if ($env:REDIS_CONTAINER) { $env:REDIS_CONTAINER } else { "lenschain-redis" }

Write-Host "==> Flushing Redis cache (container=$REDIS_CONTAINER, db=$REDIS_DB)"
& docker exec $REDIS_CONTAINER redis-cli -n $REDIS_DB FLUSHDB
if ($LASTEXITCODE -ne 0) {
    throw "Redis 缓存清理失败，请确认容器 $REDIS_CONTAINER 正在运行"
}

$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$backendDir = Join-Path $repoRoot "backend"
$deployImagesDir = Join-Path $repoRoot "deploy\images"
$goCacheDir = Join-Path $backendDir ".gocache"

# 种子数据加载顺序（详见 backend/seeds/）：
#   schema 迁移 → image_categories（sync 前置依赖）
#   → seed-manifests CLI（从 manifest.yaml 灌入 images / image_versions）
#   → 其它 seed（CTF 题目模板 / 演示数据 / 实验模板 / sim 场景 / CTF 竞赛），
#     这些 seed 通过 (image_name, version) 子查询关联 image_version_id，
#     必须排在 sync 之后才能解析。
$seedDir = Join-Path $backendDir "seeds"
$imageCategoriesSeed   = Join-Path $seedDir "000_seed_image_categories.sql"
$ctfTemplatesSeed      = Join-Path $seedDir "001_seed_ctf_challenge_templates.sql"
$demoSeed              = Join-Path $seedDir "002_seed_demo_data.sql"
$demoSupplementSeed    = Join-Path $seedDir "003_seed_demo_supplement.sql"
$imagesExperimentsSeed = Join-Path $seedDir "004_seed_images_experiments.sql"
$simScenariosSeed      = Join-Path $seedDir "005_seed_sim_scenarios.sql"
$ctfSeed               = Join-Path $seedDir "006_seed_ctf.sql"

if (-not $env:GOCACHE) {
    if (-not (Test-Path $goCacheDir)) {
        New-Item -ItemType Directory -Path $goCacheDir -Force | Out-Null
    }
    $env:GOCACHE = $goCacheDir
}

Write-Host "==> Running schema migrations (backend/migrations)"
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

function Invoke-Seed {
    param([string]$Label, [string]$FilePath)
    if (Test-Path $FilePath) {
        Write-Host "==> Seeding $Label from $FilePath"
        & psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -v ON_ERROR_STOP=1 -f $FilePath
        if ($LASTEXITCODE -ne 0) {
            throw "导入 $Label 失败"
        }
    }
    else {
        Write-Host "==> $Label seed file not found ($FilePath), skipping"
    }
}

Invoke-Seed "image categories" $imageCategoriesSeed

Write-Host "==> Syncing image manifests (deploy/images/**/manifest.yaml → images / image_versions)"
Push-Location $backendDir
try {
    & go run cmd/seed-manifests/main.go -images-dir $deployImagesDir
    if ($LASTEXITCODE -ne 0) {
        throw "镜像 manifest 同步失败"
    }
}
finally {
    Pop-Location
}

Invoke-Seed "CTF challenge templates"        $ctfTemplatesSeed
Invoke-Seed "demo data"                      $demoSeed
Invoke-Seed "demo supplement"                $demoSupplementSeed
Invoke-Seed "images & experiments templates" $imagesExperimentsSeed
Invoke-Seed "sim scenarios"                  $simScenariosSeed
Invoke-Seed "CTF competitions"               $ctfSeed

Write-Host "==> Database initialization complete"
