# build-wasm.ps1 — compile le moteur d'enchères en WebAssembly dans cli\.
# Produit cli\bids.wasm (le moteur engine\, par son point d'entrée wasm\) et
# cli\wasm_exec.js (la glue de la distribution Go, copiée telle quelle). Les
# deux sont versionnés ; le workflow wasm.yml les recompile sur main à chaque
# modification des sources Go.
#
# Usage : .\build-wasm.ps1
param()

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go est introuvable dans le PATH."
}

$goroot = (go env GOROOT)
$execJs = Join-Path $goroot "lib\wasm\wasm_exec.js"          # Go >= 1.24
if (-not (Test-Path $execJs)) {
    $execJs = Join-Path $goroot "misc\wasm\wasm_exec.js"     # Go < 1.24
}
if (-not (Test-Path $execJs)) {
    throw "wasm_exec.js introuvable sous $goroot"
}

try {
    $revision = (git -C $root rev-parse --short HEAD 2>$null)
    if (-not $revision) { $revision = "unknown" }
} catch {
    $revision = "unknown"
}

# Les variables d'environnement sont restaurées ensuite, le script pouvant être
# lancé depuis une session PowerShell que l'on ne veut pas laisser modifiée.
$old = @{ GOOS = $env:GOOS; GOARCH = $env:GOARCH; CGO_ENABLED = $env:CGO_ENABLED }
Push-Location $root
try {
    $env:GOOS = "js"
    $env:GOARCH = "wasm"
    $env:CGO_ENABLED = "0"

    # Aucun test ne compile le fichier tagué « js && wasm » : ce vet est le seul
    # garde-fou contre une faute de frappe dans main_js.go.
    go vet ./wasm ./engine
    if ($LASTEXITCODE -ne 0) { throw "go vet a échoué pour js/wasm (code $LASTEXITCODE)" }

    Copy-Item $execJs (Join-Path $root "cli\wasm_exec.js") -Force

    Write-Host "Compilation de cli\bids.wasm (js/wasm, revision $revision)..."
    go build -trimpath -ldflags="-s -w -X bids/engine.buildRevision=$revision" -o (Join-Path $root "cli\bids.wasm") ./wasm
    if ($LASTEXITCODE -ne 0) { throw "go build a échoué pour js/wasm (code $LASTEXITCODE)" }
}
finally {
    Pop-Location
    $env:GOOS = $old.GOOS
    $env:GOARCH = $old.GOARCH
    $env:CGO_ENABLED = $old.CGO_ENABLED
}

# La taille gzip est le chiffre utile : c'est ce que l'hébergeur transmet. Un
# bond au-delà de ~2 Mo signalerait qu'une dépendance lourde (net/http...) a
# fui dans le moteur.
$wasm = Get-Item (Join-Path $root "cli\bids.wasm")
$buffer = New-Object System.IO.MemoryStream
# leaveOpen = $true : sans lui, Dispose fermerait aussi le MemoryStream et
# sa longueur ne serait plus lisible.
$gzip = New-Object System.IO.Compression.GZipStream($buffer, [System.IO.Compression.CompressionLevel]::Optimal, $true)
$gzip.Write([System.IO.File]::ReadAllBytes($wasm.FullName), 0, $wasm.Length)
$gzip.Dispose()
Write-Host ("  cli\bids.wasm ({0:N1} Mo, {1:N1} Mo gzip)" -f ($wasm.Length / 1MB), ($buffer.Length / 1MB))
$exec = Get-Item (Join-Path $root "cli\wasm_exec.js")
Write-Host ("  cli\wasm_exec.js ({0} octets, repris de {1})" -f $exec.Length, $goroot)
