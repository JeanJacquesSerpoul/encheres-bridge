#!/usr/bin/env bash
# build-proxy.sh — compile le serveur IA dans bin/ : bin/openrouter_proxy
# (linux/amd64) et bin/openrouter_proxy.exe (windows/amd64), que lancent
# ensuite bin/openrouter_proxy.sh et bin/openrouter_proxy.ps1. Ils ne sont pas
# versionnés : à lancer une fois après un clone, puis après chaque
# modification des sources.
#
# Mêmes drapeaux que la cible « make bin » du Makefile : ce script est la voie
# sans make, sous Windows comme sous Linux.
#
# Usage : ./build-proxy.sh [-h]
set -euo pipefail

usage() {
    sed -n '2,11p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while getopts "h" opt; do
    case "$opt" in
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

command -v go >/dev/null 2>&1 || { echo "Go est introuvable dans le PATH." >&2; exit 1; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
mkdir -p "$root/bin"

# -trimpath : aucun chemin de la machine de compilation ne doit se retrouver
# dans un binaire distribué. -s -w : sans table des symboles ni informations de
# débogage.
ldflags="-s -w"

echo "Compilation de bin/openrouter_proxy (linux/amd64)..."
(
    cd "$root"
    # CGO désactivé : le binaire Linux doit rester statique, c'est lui que
    # monte l'image Alpine de bin/docker-compose.yml.
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$ldflags" -o "$root/bin/openrouter_proxy" .
)

echo "Compilation de bin/openrouter_proxy.exe (windows/amd64)..."
(
    cd "$root"
    GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$ldflags" -o "$root/bin/openrouter_proxy.exe" .
)

# Un clone ou une archive peut perdre le bit d'exécution, qu'il vaut mieux
# rétablir ici que de le laisser manquant dans l'index.
chmod +x "$root/bin/openrouter_proxy"

# `wc -c` et non `stat -c%s` : cette option est propre à GNU, le stat de macOS
# la refuse et, sous `set -e`, arrêterait le script après la compilation.
for f in bin/openrouter_proxy bin/openrouter_proxy.exe; do
    size=$(wc -c < "$root/$f" | tr -d " ")
    awk -v f="$f" -v s="$size" 'BEGIN { printf "  %s (%.1f Mo)\n", f, s / 1048576 }'
done
