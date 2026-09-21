# Audit complet du moteur contre le par : tire les donnes, joue les enchères,
# calcule le par en double-mort, écrit le rapport.
#
#   tools\par\run.ps1 [-Deals 1000] [-Seed 20260906]
#
# Sortie dans tools\par\out\ : auctions.jsonl (enchères brutes), par.json
# (analyse complète) et rapport.html (rapport lisible).
param(
    [int]$Deals = 1000,
    [long]$Seed = 20260906
)

$ErrorActionPreference = "Stop"

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
$root = Resolve-Path (Join-Path $here "..\..")
$out = Join-Path $here "out"

if (-not (Test-Path $out)) { New-Item -ItemType Directory -Path $out | Out-Null }

$env:PAR_AUDIT_DEALS = $Deals
$env:PAR_AUDIT_SEED = $Seed
$env:PAR_AUDIT_OUT = Join-Path $out "auctions.jsonl"
$env:PAR_AUDIT_ENGINE = (git -C $root rev-parse --short HEAD 2>$null)
if (-not $?) { $env:PAR_AUDIT_ENGINE = "" }

Write-Host "1/2  $Deals donnes, graine $Seed : enchères du moteur"
Push-Location $root
try {
    go test -run TestParAuditDump -count=1 . | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "go test a échoué (code $LASTEXITCODE)" }
} finally {
    Pop-Location
}

Write-Host "2/2  levées double-mort et par"
node (Join-Path $here "audit.js") $env:PAR_AUDIT_OUT $out
if ($LASTEXITCODE -ne 0) { throw "audit.js a échoué (code $LASTEXITCODE)" }
