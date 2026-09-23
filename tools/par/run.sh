#!/usr/bin/env bash
# Audit complet du moteur contre le par : tire les donnes, joue les enchères,
# calcule le par en double-mort, écrit le rapport.
#
#   tools/par/run.sh [nombre de donnes] [graine]
#
# Sortie dans tools/par/out/ : auctions.jsonl (enchères brutes), par.json
# (analyse complète) et rapport.html (rapport lisible).
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"
out="$here/out"

export PAR_AUDIT_DEALS="${1:-1000}"
export PAR_AUDIT_SEED="${2:-20260906}"
export PAR_AUDIT_OUT="$out/auctions.jsonl"
export PAR_AUDIT_ENGINE="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo '')"

mkdir -p "$out"

echo "1/2  $PAR_AUDIT_DEALS donnes, graine $PAR_AUDIT_SEED : enchères du moteur"
(cd "$root" && go test -run TestParAuditDump -count=1 ./engine >/dev/null)

echo "2/2  levées double-mort et par"
node "$here/audit.js" "$PAR_AUDIT_OUT" "$out"
