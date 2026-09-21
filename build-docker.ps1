# build-docker.ps1 — construit l'image Docker du serveur (Dockerfile multi-étapes).
# La révision Git courte est injectée dans le binaire via --build-arg REVISION,
# de sorte que /version renvoie le bon commit (voir README).
#
# Usage : .\build-docker.ps1 [-Image nom:tag] [-Platform linux/amd64] [-NoCache] [-Run]
param(
    [string]$Image = "bridge-bids:latest",
    [string]$Platform = "",
    [switch]$NoCache,
    [switch]$Run
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

# Avec $ErrorActionPreference = "Stop", le stderr d'une commande native devient une
# erreur terminante dès qu'il est redirigé. Ce helper isole les appels dont seul le
# code de retour nous intéresse (contrôles, sondes, nettoyage).
# Les arguments sont passés en un seul tableau : sinon PowerShell tenterait de lier
# « -p » aux paramètres communs de la fonction au lieu de le transmettre à docker.
function Invoke-DockerQuiet {
    param([string[]]$DockerArgs)
    $prev = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        & docker @DockerArgs 2>&1 | Out-Null
        return $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $prev
    }
}

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker est introuvable dans le PATH (démarrez Docker Desktop)."
}
if ((Invoke-DockerQuiet @("info")) -ne 0) {
    throw "Le démon Docker ne répond pas (démarrez Docker Desktop)."
}

$dockerfile = Join-Path $root "Dockerfile"
if (-not (Test-Path $dockerfile)) { throw "Dockerfile introuvable dans $root" }

try {
    $revision = (git -C $root rev-parse --short HEAD 2>$null)
    if (-not $revision) { $revision = "unknown" }
} catch {
    $revision = "unknown"
}

# Le dépôt modifié ne correspond plus au commit : on le signale.
# Même précaution que Invoke-DockerQuiet : un simple avertissement de git sur stderr
# (fins de ligne CRLF/LF, par exemple) ne doit pas interrompre la construction.
$prev = $ErrorActionPreference
$ErrorActionPreference = "Continue"
try {
    git -C $root diff --quiet HEAD 2>&1 | Out-Null
    $dirty = $LASTEXITCODE -ne 0
} finally {
    $ErrorActionPreference = $prev
}
if ($dirty) {
    Write-Host "Attention : dépôt modifié, la révision $revision ne reflète pas exactement l'image."
}

$port = if ($env:PORT) { $env:PORT } else { "9015" }
$container = "bridge-bids-check"

$buildArgs = @("build")
if ($NoCache) { $buildArgs += "--no-cache" }
if ($Platform) { $buildArgs += @("--platform", $Platform) }
$buildArgs += @("--build-arg", "REVISION=$revision", "-t", $Image, $root)

$suffix = if ($Platform) { ", $Platform" } else { "" }
Write-Host "Construction de $Image (revision $revision$suffix)..."
docker @buildArgs
if ($LASTEXITCODE -ne 0) { throw "docker build a échoué (code $LASTEXITCODE)" }

$size = [double](docker image inspect -f '{{.Size}}' $Image)
Write-Host ("Terminé : {0} ({1:N1} Mo)." -f $Image, ($size / 1MB))

if (-not $Run) {
    Write-Host "Lancez le serveur avec : docker run -d -p ${port}:9015 --name bridge-bids $Image"
    exit 0
}

# Vérification : démarrage du conteneur puis attente de /ready (le moteur rejoue
# une donne de référence, donc un 200 atteste d'un moteur fonctionnel).
$null = Invoke-DockerQuiet @("rm", "-f", $container)
Write-Host "Démarrage de $container sur le port $port..."
# Seule la sortie standard (l'identifiant du conteneur) est masquée : le message
# d'erreur éventuel de docker reste visible avant notre propre diagnostic.
$null = docker run -d --name $container -p "${port}:9015" $Image
if ($LASTEXITCODE -ne 0) {
    throw "docker run a échoué : le port $port est peut-être déjà occupé (relancez en fixant la variable PORT sur un port libre)."
}

foreach ($i in 1..30) {
    if ((Invoke-DockerQuiet @("exec", $container, "wget", "-qO-", "http://localhost:9015/ready")) -eq 0) {
        Write-Host "OK : /ready répond. Version :"
        docker exec $container wget -qO- http://localhost:9015/version
        Write-Host ""
        Write-Host "Conteneur $container laissé actif sur http://localhost:${port}/"
        Write-Host "Pour l'arrêter : docker rm -f $container"
        exit 0
    }
    Start-Sleep -Seconds 1
}

Write-Host "Échec : /ready n'a pas répondu en 30 s. Journaux :"
$ErrorActionPreference = "Continue"
docker logs $container 2>&1 | Write-Host
$null = Invoke-DockerQuiet @("rm", "-f", $container)
exit 1
