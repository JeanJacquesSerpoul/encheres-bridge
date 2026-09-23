#!/usr/bin/env bash
# build-wasm.sh — compile le moteur d'enchères en WebAssembly dans cli/.
# Produit cli/bids.wasm (le moteur, tagué js && wasm dans main_js.go) et
# cli/wasm_exec.js (la glue de la distribution Go, copiée telle quelle). Les
# deux sont ignorés par git et embarqués dans le binaire du serveur par
# //go:embed all:cli : ce script doit donc tourner AVANT build-server.sh et
# build-docker.sh, qui s'en chargent eux-mêmes.
#
# Usage : ./build-wasm.sh [-h]
set -euo pipefail

usage() {
    sed -n '2,10p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while getopts "h" opt; do
    case "$opt" in
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

command -v go >/dev/null 2>&1 || { echo "Go est introuvable dans le PATH." >&2; exit 1; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

goroot="$(go env GOROOT)"
exec_js="$goroot/lib/wasm/wasm_exec.js"                        # Go >= 1.24
[ -f "$exec_js" ] || exec_js="$goroot/misc/wasm/wasm_exec.js"  # Go < 1.24
[ -f "$exec_js" ] || { echo "wasm_exec.js introuvable sous $goroot" >&2; exit 1; }

revision="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"

# Aucun test ne compile le fichier tagué « js && wasm » : ce vet est le seul
# garde-fou contre une faute de frappe dans main_js.go.
(cd "$root" && GOOS=js GOARCH=wasm go vet .)

cp "$exec_js" "$root/cli/wasm_exec.js"

echo "Compilation de cli/bids.wasm (js/wasm, revision $revision)..."
(
    cd "$root"
    GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -trimpath \
        -ldflags="-s -w -X main.buildRevision=$revision" -o "$root/cli/bids.wasm" .
)

# La taille gzip est le chiffre utile : c'est ce que le serveur transmet
# (voir gzipStatic dans main.go). Un bond au-delà de ~2 Mo signalerait qu'une
# dépendance serveur a fui dans la cible, donc un //go:build mal posé.
# `wc -c` et non `stat -c%s` : cette option est propre à GNU, le stat de
# macOS la refuse et, sous `set -e`, arrêtait le script après la compilation.
raw=$(wc -c < "$root/cli/bids.wasm" | tr -d " ")
gz=$(gzip -9 -c "$root/cli/bids.wasm" | wc -c)
awk -v r="$raw" -v g="$gz" 'BEGIN { printf "  cli/bids.wasm (%.1f Mo, %.1f Mo gzip)\n", r / 1048576, g / 1048576 }'
echo "  cli/wasm_exec.js ($(wc -c < "$root/cli/wasm_exec.js" | tr -d " ") octets, repris de $goroot)"
