#!/usr/bin/env bash
# build-server.sh — compile les exécutables du serveur dans server/.
# Produit server/bids-linux et server/bids-windows.exe : le client de test
# (cli/) étant embarqué dans le binaire via //go:embed, ces fichiers sont
# autonomes, aucune dépendance au répertoire de travail à l'exécution.
# Le moteur en WebAssembly (cli/bids.wasm) est compilé d'abord, par
# build-wasm.sh : //go:embed le fige à la compilation du serveur.
#
# Usage : ./build-server.sh [-a amd64|arm64] [-o linux|windows|both] [-w] [-h]
#   -a  architecture cible                 (défaut : amd64)
#   -o  système à compiler                 (défaut : both)
#   -w  ne pas recompiler cli/bids.wasm
#   -h  affiche cette aide
set -euo pipefail

arch="amd64"
targets="both"
skip_wasm=0

usage() {
    sed -n '2,13p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while getopts "a:o:wh" opt; do
    case "$opt" in
        a) arch="$OPTARG" ;;
        o) targets="$OPTARG" ;;
        w) skip_wasm=1 ;;
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

case "$arch" in
    amd64|arm64) ;;
    *) echo "Architecture invalide : $arch (attendu : amd64 ou arm64)" >&2; exit 1 ;;
esac
case "$targets" in
    linux|windows|both) ;;
    *) echo "Cible invalide : $targets (attendu : linux, windows ou both)" >&2; exit 1 ;;
esac

command -v go >/dev/null 2>&1 || { echo "Go est introuvable dans le PATH." >&2; exit 1; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
out="$root/server"
mkdir -p "$out"

revision="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"
if ! git -C "$root" diff --quiet HEAD 2>/dev/null; then
    echo "Attention : dépôt modifié, la révision $revision ne reflète pas exactement les binaires."
fi

# //go:embed all:cli fige le contenu de cli/ à la compilation du serveur : le
# moteur en WebAssembly doit donc être produit avant, faute de quoi le mode
# « navigateur » du client répondrait 404 sur bids.wasm.
if [ "$skip_wasm" -eq 0 ]; then
    "$root/build-wasm.sh"
fi
if [ ! -f "$root/cli/bids.wasm" ]; then
    echo "cli/bids.wasm manquant : lancez ./build-wasm.sh (ou retirez -w)." >&2
    exit 1
fi

# CGO_ENABLED=0 : binaire statique, qui tourne sans dépendance système et sans
# chaîne de compilation C pour la cible croisée.
build() {
    local goos="$1" name="$2"
    echo "Compilation de server/$name ($goos/$arch, revision $revision)..."
    (
        cd "$root"
        GOOS="$goos" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath \
            -ldflags="-s -w -X main.buildRevision=$revision" -o "$out/$name" ./engine
    )
    local mo
    mo=$(awk -v b="$(wc -c < "$out/$name" | tr -d " ")" 'BEGIN { printf "%.1f", b / 1048576 }')
    echo "  server/$name (${mo} Mo)"
}

[ "$targets" = "windows" ] || build linux bids-linux
[ "$targets" = "linux" ] || build windows bids-windows.exe

cat <<'MSG'

Terminé. Pour utiliser l'application :
  1. lancer l'exécutable de votre système (server/bids-linux ou
     server\bids-windows.exe) — il écoute sur le port 9015 ;
  2. http://localhost:9015/ dans un navigateur.
Le client calcule les enchères dans le navigateur (cli/bids.wasm) dès la
première visite, comme sur une copie statique de cli/. Rien à configurer.
MSG
