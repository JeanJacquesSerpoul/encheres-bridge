# run.ps1 — reprend les captures d'écran du tutoriel du client, sous Windows.
# Équivalent de run.sh.
#
# Sert cli\ en local le temps des captures, puis écrit
# cli\tutorial\fr\NN.jpg et cli\tutorial\en\NN.jpg. À relancer après un
# changement visible de l'interface. Demande Node, Python et Playwright
# (Chromium) : voir README.md.
#
# Les captures sont prises dans un dossier temporaire et ne remplacent
# cli\tutorial\ qu'une fois toutes réussies : un échec laisse les images en
# place.
#
# Usage : .\tools\tutorial\run.ps1 [-Port 9377]
#   -Port  port du serveur local  (défaut : 9377, ou $env:TUTORIAL_PORT)
param(
    [ValidateRange(1, 65535)]
    [int]$Port = $(if ($env:TUTORIAL_PORT) { [int]$env:TUTORIAL_PORT } else { 9377 })
)

$ErrorActionPreference = "Stop"

$here = $PSScriptRoot
$root = (Resolve-Path (Join-Path $here "..\..")).Path
$cli = Join-Path $root "cli"
$dest = Join-Path $cli "tutorial"
$url = "http://127.0.0.1:$Port/index.html"

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    throw "Node est introuvable dans le PATH."
}

# Playwright avant tout : sans lui, inutile de lancer le serveur. capture.js le
# cherche dans le projet puis parmi les paquets globaux ; même recherche ici.
# Sous PowerShell 5.1, la sortie d'erreur d'un exécutable redirigée devient
# une erreur PowerShell, qui arrêterait le script : on l'y tolère ici.
$ErrorActionPreference = "Continue"
node -e "try { require('playwright') } catch { require(require('path').join(require('child_process').execSync('npm root -g').toString().trim(), 'playwright')) }" 2>$null
$found = $LASTEXITCODE -eq 0
$ErrorActionPreference = "Stop"
if (-not $found) {
    throw "Playwright est introuvable. Installez-le : npm install -g playwright, puis npx playwright install chromium."
}

# Python : « py » (le lanceur Windows) ou « python ». « python3 » est souvent,
# sous Windows, un raccourci vers le Microsoft Store qui ne sert rien.
$python = $null
foreach ($name in "py", "python") {
    if (Get-Command $name -ErrorAction SilentlyContinue) { $python = $name; break }
}
if (-not $python) {
    throw "Python est introuvable dans le PATH (py ou python)."
}

# À côté de cli\tutorial, sur le même disque : Move-Item ne déplace pas un
# dossier d'un volume à l'autre.
$out = Join-Path $cli "tutorial.new"
if (Test-Path $out) { Remove-Item -Recurse -Force $out }
$server = Start-Process -FilePath $python -PassThru -WindowStyle Hidden `
    -ArgumentList "-m", "http.server", "$Port", "--bind", "127.0.0.1", "--directory", "`"$cli`""
try {
    # Le serveur met un instant à écouter. -UseBasicParsing : sans lui,
    # PowerShell 5.1 passe par le moteur d'Internet Explorer.
    $ready = $false
    for ($i = 0; $i -lt 50; $i++) {
        try {
            Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 2 | Out-Null
            $ready = $true
            break
        } catch {
            if ($server.HasExited) { throw "Le serveur Python s'est arrêté (port $Port déjà pris ?)." }
            Start-Sleep -Milliseconds 100
        }
    }
    if (-not $ready) { throw "Le serveur local ne répond pas sur $url." }

    node (Join-Path $here "capture.js") $url $out
    if ($LASTEXITCODE -ne 0) { throw "capture.js a échoué (code $LASTEXITCODE) : cli\tutorial\ est inchangé." }

    if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
    Move-Item $out $dest
    $size = (Get-ChildItem $dest -Recurse -File | Measure-Object Length -Sum).Sum
    Write-Host ("cli\tutorial : {0:N0} Ko" -f ($size / 1KB))
} finally {
    if (-not $server.HasExited) { Stop-Process -Id $server.Id -Force }
    if (Test-Path $out) { Remove-Item -Recurse -Force $out }
}
