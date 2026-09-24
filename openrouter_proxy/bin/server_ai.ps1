# server_ai.ps1 — lance le serveur IA précompilé (bin\server_ai.exe, Windows
# amd64) sur http://localhost:<port>/. Ctrl+C l'arrête.
#
# Le serveur lit son .env dans le dossier courant : le script se place donc
# dans openrouter_proxy\, où se trouve .env (à copier depuis .env.example).
# Les variables déjà présentes dans l'environnement l'emportent sur .env.
#
# Usage : .\bin\server_ai.ps1 [-Port 9013]
#   -Port  port d'écoute     (défaut : PORT du .env, sinon 9013)
param(
    [ValidateRange(0, 65535)]
    [int]$Port = 0
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$bin = Join-Path $PSScriptRoot "server_ai.exe"

if (-not (Test-Path $bin)) {
    throw "Binaire introuvable : $bin (make bin le recompile)."
}
if (-not (Test-Path (Join-Path $root ".env")) -and -not $env:OPENROUTER_API_KEY) {
    throw "Ni $root\.env ni OPENROUTER_API_KEY : copiez .env.example en .env et renseignez la clé."
}

# PORT et le dossier courant sont restaurés ensuite, le script pouvant être
# lancé depuis une session PowerShell que l'on ne veut pas laisser modifiée.
$oldPort = $env:PORT
Push-Location $root
try {
    if ($Port -gt 0) { $env:PORT = "$Port" }
    & $bin
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}
finally {
    Pop-Location
    $env:PORT = $oldPort
}
