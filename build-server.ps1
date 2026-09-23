# build-server.ps1 — compile les exécutables du serveur dans server\.
# Produit server\bids-linux et server\bids-windows.exe : le client de test
# (cli\) étant embarqué dans le binaire via //go:embed, ces fichiers sont
# autonomes, aucune dépendance au répertoire de travail à l'exécution.
# Le moteur en WebAssembly (cli\bids.wasm) est compilé d'abord, par
# build-wasm.ps1 : //go:embed le fige à la compilation du serveur.
#
# Usage : .\build-server.ps1 [-Arch amd64|arm64] [-Targets linux|windows|both] [-NoWasm]
param(
    [ValidateSet("amd64", "arm64")]
    [string]$Arch = "amd64",
    [ValidateSet("linux", "windows", "both")]
    [string]$Targets = "both",
    [switch]$NoWasm
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
$out = Join-Path $root "server"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go est introuvable dans le PATH."
}
New-Item -ItemType Directory -Force -Path $out | Out-Null

try {
    $revision = (git -C $root rev-parse --short HEAD 2>$null)
    if (-not $revision) { $revision = "unknown" }
} catch {
    $revision = "unknown"
}

# Un simple avertissement de git sur stderr (fins de ligne CRLF/LF, par exemple)
# ne doit pas interrompre la construction sous $ErrorActionPreference = "Stop".
$prev = $ErrorActionPreference
$ErrorActionPreference = "Continue"
try {
    git -C $root diff --quiet HEAD 2>&1 | Out-Null
    $dirty = $LASTEXITCODE -ne 0
} finally {
    $ErrorActionPreference = $prev
}
if ($dirty) {
    Write-Host "Attention : dépôt modifié, la révision $revision ne reflète pas exactement les binaires."
}

# //go:embed all:cli fige le contenu de cli\ à la compilation du serveur : le
# moteur en WebAssembly doit donc être produit avant, faute de quoi le mode
# « navigateur » du client répondrait 404 sur bids.wasm. Appelé hors du
# Push-Location et de la bascule de $env: ci-dessous, que build-wasm.ps1 gère
# de son côté.
if (-not $NoWasm) {
    & (Join-Path $root "build-wasm.ps1")
}
if (-not (Test-Path (Join-Path $root "cli\bids.wasm"))) {
    throw "cli\bids.wasm manquant : lancez .\build-wasm.ps1 (ou retirez -NoWasm)."
}

# CGO_ENABLED=0 : binaire statique, qui tourne sans dépendance système et sans
# chaîne de compilation C pour la cible croisée. Les variables d'environnement
# sont restaurées ensuite, le script pouvant être lancé depuis une session
# PowerShell que l'on ne veut pas laisser modifiée.
$old = @{ GOOS = $env:GOOS; GOARCH = $env:GOARCH; CGO_ENABLED = $env:CGO_ENABLED }
Push-Location $root
try {
    $env:GOARCH = $Arch
    $env:CGO_ENABLED = "0"

    $builds = @()
    if ($Targets -ne "windows") { $builds += @{ GOOS = "linux";   Name = "bids-linux" } }
    if ($Targets -ne "linux")   { $builds += @{ GOOS = "windows"; Name = "bids-windows.exe" } }

    foreach ($b in $builds) {
        $env:GOOS = $b.GOOS
        Write-Host "Compilation de server\$($b.Name) ($($b.GOOS)/$Arch, revision $revision)..."
        go build -trimpath -ldflags="-s -w -X main.buildRevision=$revision" -o (Join-Path $out $b.Name) ./engine
        if ($LASTEXITCODE -ne 0) { throw "go build a échoué pour $($b.GOOS) (code $LASTEXITCODE)" }
        $bin = Get-Item (Join-Path $out $b.Name)
        Write-Host ("  server\{0} ({1:N1} Mo)" -f $bin.Name, ($bin.Length / 1MB))
    }
}
finally {
    Pop-Location
    $env:GOOS = $old.GOOS
    $env:GOARCH = $old.GOARCH
    $env:CGO_ENABLED = $old.CGO_ENABLED
}

Write-Host ""
Write-Host "Terminé. Pour utiliser l'application :"
Write-Host "  1. lancer l'exécutable de votre système (server\bids-windows.exe ou"
Write-Host "     server/bids-linux) — il écoute sur le port 9015 ;"
Write-Host "  2. http://localhost:9015/ dans un navigateur."
Write-Host "Le client calcule les enchères dans le navigateur (cli\bids.wasm) dès la"
Write-Host "première visite, comme sur une copie statique de cli/. Rien à configurer."
