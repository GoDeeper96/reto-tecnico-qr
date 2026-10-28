# deploy.ps1
# Despliega el reto completo en AWS (cuenta EYATECH): pruebas, stack de CDK y frontend en Amplify.
# Uso (desde la raíz del repo, con Docker Desktop abierto):
#   .\deploy.ps1
#   .\deploy.ps1 -SoloFrontend      # vuelve a publicar solo el frontend
# Requiere: AWS CLI con el perfil indicado, AWS CDK CLI, Python con aws-cdk-lib, Node 22 y Docker.

param(
    [string]$Perfil = "eyatech",
    [string]$Region = "us-east-1",
    [switch]$SoloFrontend
)

$ErrorActionPreference = "Stop"
$Raiz = $PSScriptRoot

function Paso($texto) { Write-Host "`n$texto" -ForegroundColor Yellow }
function Comprobar($descripcion) { if ($LASTEXITCODE -ne 0) { throw "$descripcion falló (código $LASTEXITCODE)." } }

Write-Host "`n=== Reto Matrices — despliegue en AWS ===" -ForegroundColor Cyan
$Cuenta = aws sts get-caller-identity --query Account --output text --profile $Perfil
Comprobar "Resolver la cuenta del perfil '$Perfil'"
Write-Host "Perfil: $Perfil | Cuenta: $Cuenta | Región: $Region" -ForegroundColor Gray

$Salidas = Join-Path $Raiz "cdk-outputs.json"

if (-not $SoloFrontend) {
    # ── 1. Pruebas (las de Go corren dentro del build de su imagen Docker) ──
    Paso "[1/5] Pruebas de la API en Node..."
    Push-Location (Join-Path $Raiz "api-node")
    npm ci --silent; Comprobar "npm ci (api-node)"
    npm test; Comprobar "Pruebas de api-node"
    Pop-Location

    # ── 2. Stack de CDK: imágenes en ECR, Lambdas, API Gateway, secretos y app de Amplify ──
    Paso "[2/5] Desplegando el stack RetoMatrices (construye las imágenes y corre las pruebas de Go)..."
    Push-Location (Join-Path $Raiz "infra")
    cdk deploy RetoMatrices `
        --profile $Perfil `
        --context account=$Cuenta `
        --context region=$Region `
        --require-approval never `
        --outputs-file $Salidas
    Comprobar "cdk deploy"
    Pop-Location
}

$o = (Get-Content $Salidas -Raw | ConvertFrom-Json).RetoMatrices
$UrlGo = $o.ApiGoUrl.TrimEnd("/")
$UrlNode = $o.ApiNodeUrl.TrimEnd("/")
$ApiKey = aws secretsmanager get-secret-value --secret-id $o.ApiKeySecretArn --query SecretString --output text --profile $Perfil --region $Region
Comprobar "Leer la API key"

# ── 3. Frontend: construir y empaquetar en un solo paso (así el zip nunca lleva un build viejo) ──
# La API key NO va en el build: quien la recibe por correo la ingresa en la página.
Paso "[3/5] Construyendo el frontend..."
$Front = Join-Path $Raiz "frontend"
Push-Location $Front
@"
VITE_API_GO_URL=$UrlGo
VITE_API_NODE_URL=$UrlNode
"@ | Set-Content -Encoding ascii .env.production
npm ci --silent; Comprobar "npm ci (frontend)"
npx vite build --mode production; Comprobar "vite build"
if (Select-String -Path "dist\assets\*.js" -SimpleMatch -Pattern $ApiKey -Quiet) {
    throw "El build del frontend contiene la API key; no se publica."
}

# Zip con separadores '/': Compress-Archive de PowerShell 5.1 usa '\' y Amplify no encuentra los archivos.
$Zip = Join-Path $Front "dist.zip"
if (Test-Path $Zip) { Remove-Item $Zip }
Add-Type -AssemblyName System.IO.Compression, System.IO.Compression.FileSystem
$Dist = (Resolve-Path "dist").Path
$archivo = [System.IO.Compression.ZipFile]::Open($Zip, "Create")
Get-ChildItem $Dist -Recurse -File | ForEach-Object {
    $relativa = $_.FullName.Substring($Dist.Length + 1).Replace("\", "/")
    [void][System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile($archivo, $_.FullName, $relativa)
}
$archivo.Dispose()
$Bundle = [regex]::Match((Get-Content "dist/index.html" -Raw), '/assets/index-[^"]+\.js').Value
Pop-Location

# ── 4. Publicar en Amplify ──
Paso "[4/5] Publicando en Amplify (app $($o.AmplifyAppId))..."
$despliegue = aws amplify create-deployment --app-id $o.AmplifyAppId --branch-name $o.AmplifyBranch --profile $Perfil --region $Region | ConvertFrom-Json
Comprobar "amplify create-deployment"
Invoke-WebRequest -Uri $despliegue.zipUploadUrl -Method Put -InFile $Zip -ContentType "application/zip" -UseBasicParsing | Out-Null
aws amplify start-deployment --app-id $o.AmplifyAppId --branch-name $o.AmplifyBranch --job-id $despliegue.jobId --profile $Perfil --region $Region | Out-Null
Comprobar "amplify start-deployment"
do {
    Start-Sleep -Seconds 5
    $estado = aws amplify get-job --app-id $o.AmplifyAppId --branch-name $o.AmplifyBranch --job-id $despliegue.jobId `
        --query "job.summary.status" --output text --profile $Perfil --region $Region
    Write-Host "  job $($despliegue.jobId): $estado" -ForegroundColor Gray
} while ($estado -in @("PENDING", "PROVISIONING", "RUNNING"))
if ($estado -ne "SUCCEED") { throw "El despliegue de Amplify terminó en $estado." }

# Prueba de que el sitio sirve ESTE build y no uno anterior.
$Sirve = [regex]::Match((Invoke-WebRequest -Uri $o.FrontendUrl -UseBasicParsing).Content, '/assets/index-[^"]+\.js').Value
if ($Sirve -ne $Bundle) { throw "El sitio sirve $Sirve, pero el build es $Bundle." }
Write-Host "  ✅ El sitio sirve el bundle $Bundle" -ForegroundColor Green

# ── 5. Prueba de humo contra AWS ──
Paso "[5/5] Prueba de humo contra las APIs desplegadas..."
node (Join-Path $Raiz "scripts/prueba-humo.mjs") $UrlGo $UrlNode $ApiKey
Comprobar "Prueba de humo"

Write-Host "`n=== Despliegue completo ===" -ForegroundColor Green
Write-Host "Frontend : $($o.FrontendUrl)"
Write-Host "API Go   : $UrlGo"
Write-Host "API Node : $UrlNode"
