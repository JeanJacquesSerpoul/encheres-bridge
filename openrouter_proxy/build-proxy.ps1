# build-proxy.ps1 — compile le serveur IA dans bin\ : bin\openrouter_proxy.exe
# (windows/amd64) et bin\openrouter_proxy (linux/amd64). Les deux sont
# versionnés : ce sont eux que lancent bin\openrouter_proxy.ps1 et
# bin\openrouter_proxy.sh, sans Go ni Docker.
#
# Mêmes drapeaux que la cible « make bin » du Makefile : ce script est la voie
# sans make, sous Windows comme sous Linux.
#
# Usage : .\build-proxy.ps1
param()

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go est introuvable dans le PATH."
}

$root = $PSScriptRoot
$bin = Join-Path $root "bin"
New-Item -ItemType Directory -Force -Path $bin | Out-Null

# -trimpath : aucun chemin de la machine de compilation ne doit se retrouver
# dans un binaire commité. -s -w : sans table des symboles ni informations de
# débogage.
$ldflags = "-s -w"

# Les variables d'environnement sont restaurées ensuite, le script pouvant être
# lancé depuis une session PowerShell que l'on ne veut pas laisser modifiée.
$old = @{ GOOS = $env:GOOS; GOARCH = $env:GOARCH; CGO_ENABLED = $env:CGO_ENABLED }
Push-Location $root
try {
    # L'exécutable Windows d'abord, comme le Makefile : CGO_ENABLED y garde sa
    # valeur d'origine, que le binaire Linux doit au contraire forcer à 0 (pour
    # rester statique dans l'image Alpine de bin\docker-compose.yml).
    Write-Host "Compilation de bin\openrouter_proxy.exe (windows/amd64)..."
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -trimpath -ldflags $ldflags -o (Join-Path $bin "openrouter_proxy.exe") .
    if ($LASTEXITCODE -ne 0) { throw "go build a échoué pour windows/amd64 (code $LASTEXITCODE)" }

    Write-Host "Compilation de bin\openrouter_proxy (linux/amd64)..."
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags $ldflags -o (Join-Path $bin "openrouter_proxy") .
    if ($LASTEXITCODE -ne 0) { throw "go build a échoué pour linux/amd64 (code $LASTEXITCODE)" }
}
finally {
    Pop-Location
    $env:GOOS = $old.GOOS
    $env:GOARCH = $old.GOARCH
    $env:CGO_ENABLED = $old.CGO_ENABLED
}

foreach ($name in @("openrouter_proxy", "openrouter_proxy.exe")) {
    $file = Get-Item (Join-Path $bin $name)
    Write-Host ("  bin\{0} ({1:N1} Mo)" -f $file.Name, ($file.Length / 1MB))
}
