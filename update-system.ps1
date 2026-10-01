# update-system.ps1 — met à jour un système d'enchères après modification de
# son fichier de règles (cli\rules\<système>.yaml).
#
# Régénère tout ce qui en dérive (données de test, enchères de référence,
# PDF, référence du banc du par s'il n'en a pas), puis lance les tests.
# Avec -Publish, publie ensuite la modification : branche (si l'on est sur
# main), commit, push et pull request.
#
# Usage : .\update-system.ps1 <système> [-AcceptPar] [-Publish] [-Message "..."]
#   <système>    nom du système : new, ou new.yaml (voir cli\rules\index.json)
#   -AcceptPar   accepter une hausse de l'écart au par (réécrit la référence du banc)
#   -Publish     publier : branche, commit, push et pull request (gh)
#   -Message     message du commit et titre de la PR (défaut : « Système <nom> : mise à jour des règles »)
param(
    [Parameter(Mandatory = $true, Position = 0)][string]$System,
    [switch]$AcceptPar,
    [switch]$Publish,
    [string]$Message = ""
)

$ErrorActionPreference = "Stop"
$root = $PSScriptRoot

# Les commandes externes ne lèvent pas d'erreur PowerShell : on lit leur code.
function Invoke-Checked([string]$what, [scriptblock]$cmd) {
    & $cmd
    if ($LASTEXITCODE -ne 0) { throw "$what a échoué (code $LASTEXITCODE)." }
}

$stem = [IO.Path]::GetFileNameWithoutExtension((Split-Path $System -Leaf))
$file = "$stem.yaml"
if (-not (Test-Path (Join-Path $root "cli\rules\$file"))) {
    throw "cli\rules\$file est introuvable."
}
if (-not $Message) { $Message = "Système $stem : mise à jour des règles" }

# Le premier Python qui sait lire le YAML : python et python3 ne sont pas
# toujours le même interpréteur.
$python = $null
foreach ($p in "python", "python3") {
    $cmd = Get-Command $p -ErrorAction SilentlyContinue
    if ($cmd) {
        & $cmd.Source -c "import yaml" *> $null
        if ($LASTEXITCODE -eq 0) { $python = $cmd; break }
    }
}
if (-not $python) { throw "Aucun Python avec PyYAML dans le PATH (pip install pyyaml)." }
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw "Go est introuvable dans le PATH." }

Write-Host "== Régénération de $file"
$regenArgs = @("regen_system.py", "../../cli/rules/$file")
if ($AcceptPar) { $regenArgs += "--par-update" }
Push-Location (Join-Path $root "tools\python_tools")
try {
    Invoke-Checked "La régénération" { & $python.Source @regenArgs }
} finally {
    Pop-Location
}

Write-Host "== Tests"
Push-Location $root
try {
    & go test ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host "Les tests échouent. Si seul l'écart au par a augmenté et que c'est voulu :"
        Write-Host "  .\update-system.ps1 $stem -AcceptPar"
        exit 1
    }

    if (-not $Publish) {
        Write-Host ""
        Write-Host "$file est à jour et les tests passent. Pour publier : .\update-system.ps1 $stem -Publish"
        exit 0
    }

    Write-Host "== Publication"
    $branch = (git rev-parse --abbrev-ref HEAD)
    if ($branch -eq "main") {
        $branch = "rules/$stem-" + (Get-Date -Format "yyyyMMdd-HHmm")
        Invoke-Checked "git switch" { git switch -c $branch }
    }
    # Seuls les fichiers des systèmes d'enchères : règles, PDF, index et données
    # de test. Les motifs sont appliqués par git, fichiers supprimés compris.
    Invoke-Checked "git add" { git add -A -- cli/rules "engine/testdata/*.json" "tools/python_tools/*.json" }
    git diff --cached --quiet
    if ($LASTEXITCODE -eq 0) {
        Write-Host "Rien à publier : aucun changement."
        exit 0
    }
    Invoke-Checked "git commit" { git commit -m $Message }
    Invoke-Checked "git push" { git push -u origin $branch }
    if (Get-Command gh -ErrorAction SilentlyContinue) {
        gh pr view $branch *> $null
        if ($LASTEXITCODE -ne 0) {
            Invoke-Checked "gh pr create" {
                gh pr create --base main --title $Message `
                    --body "Mise à jour du système d'enchères ``$file``, données de test et PDF régénérés par update-system.ps1."
            }
        }
    } else {
        Write-Host "gh est introuvable : ouvrir la pull request de $branch sur GitHub."
    }
} finally {
    Pop-Location
}
