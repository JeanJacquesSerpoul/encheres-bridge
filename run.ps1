# run.ps1 — compile puis lance le serveur, et ouvre l'application dans le
# navigateur sur http://localhost:<port>/ .
#
# Le client de test (cli\) est embarqué dans le binaire par //go:embed : le
# moteur en WebAssembly doit donc exister AVANT la compilation du serveur, ce
# script le produit au besoin via build-wasm.ps1.
#
# Le binaire est écrit dans .\bids.exe (ignoré par git) et conservé : le
# lancement suivant repart de là. Ctrl+C arrête le serveur.
#
# Si un serveur répond déjà sur le port, la page est simplement ouverte.
#
# Usage : .\run.ps1 [-Port 9015] [-NoBrowser] [-ForceWasm]
#   -Port       port d'écoute                       (défaut : 9015)
#   -NoBrowser  ne pas ouvrir le navigateur
#   -ForceWasm  recompiler cli\bids.wasm même s'il existe déjà
param(
    [ValidateRange(1, 65535)]
    [int]$Port = 9015,
    [switch]$NoBrowser,
    [switch]$ForceWasm
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot
$url = "http://localhost:$Port/"
$bin = Join-Path $root "bids.exe"

# /ready ne répond qu'une fois le moteur capable de rejouer une donne de
# référence : c'est le signal à attendre avant d'ouvrir la page, plutôt qu'un
# délai au jugé. -UseBasicParsing : sans lui, PowerShell 5.1 passe par le
# moteur d'Internet Explorer, absent des installations récentes.
function Test-Ready {
    try {
        $r = Invoke-WebRequest -Uri "http://localhost:$Port/ready" -TimeoutSec 2 -UseBasicParsing
        return $r.StatusCode -eq 200
    } catch {
        return $false
    }
}

# Serveur déjà debout : recompiler pour échouer ensuite sur « address already
# in use » n'apprendrait rien à personne.
if (Test-Ready) {
    Write-Host "Un serveur répond déjà sur le port $Port."
    if (-not $NoBrowser) { Start-Process $url }
    return
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go est introuvable dans le PATH."
}

# //go:embed all:cli fige le contenu de cli\ à la compilation : sans ce
# fichier, le mode « navigateur » du client répondrait 404 sur bids.wasm.
if ($ForceWasm -or -not (Test-Path (Join-Path $root "cli\bids.wasm"))) {
    & (Join-Path $root "build-wasm.ps1")
}

try {
    $revision = (git -C $root rev-parse --short HEAD 2>$null)
    if (-not $revision) { $revision = "unknown" }
} catch {
    $revision = "unknown"
}

# Les variables d'environnement sont restaurées ensuite, le script pouvant être
# lancé depuis une session PowerShell que l'on ne veut pas laisser modifiée.
$old = @{ CGO_ENABLED = $env:CGO_ENABLED; PORT = $env:PORT }
Write-Host "Compilation de bids.exe (revision $revision)..."
Push-Location $root
try {
    $env:CGO_ENABLED = "0"
    go build -trimpath -ldflags="-s -w -X main.buildRevision=$revision" -o $bin .
    if ($LASTEXITCODE -ne 0) { throw "go build a échoué (code $LASTEXITCODE)" }
}
finally {
    Pop-Location
    $env:CGO_ENABLED = $old.CGO_ENABLED
}

Write-Host "Démarrage du serveur sur $url (Ctrl+C pour arrêter)"
$env:PORT = "$Port"
$proc = $null
try {
    # -PassThru pour garder la main sur le processus : Ctrl+C interrompt ce
    # script, et le bloc finally arrête alors le serveur au lieu de le laisser
    # orphelin, port occupé et console rendue.
    $proc = Start-Process -FilePath $bin -NoNewWindow -PassThru
    if (-not $NoBrowser) {
        $deadline = (Get-Date).AddSeconds(60)
        $ready = $false
        while ((Get-Date) -lt $deadline -and -not $proc.HasExited) {
            if (Test-Ready) { $ready = $true; break }
            Start-Sleep -Milliseconds 300
        }
        if ($ready) {
            Write-Host "Serveur prêt — ouverture de $url"
            Start-Process $url
        } else {
            Write-Warning "Le serveur n'a pas répondu en 60 s ; ouvrez $url à la main."
        }
    }
    Wait-Process -Id $proc.Id
}
finally {
    $env:PORT = $old.PORT
    if ($proc -and -not $proc.HasExited) {
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
}
