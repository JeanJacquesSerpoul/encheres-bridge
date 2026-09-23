#!/usr/bin/env bash
# run.sh — compile puis lance le serveur, et ouvre l'application dans le
# navigateur sur http://localhost:<port>/ .
#
# Le client de test (cli/) est embarqué dans le binaire par //go:embed : le
# moteur en WebAssembly doit donc exister AVANT la compilation du serveur, ce
# script le produit au besoin via build-wasm.sh.
#
# Le binaire est écrit dans ./bids (ignoré par git) et conservé : le lancement
# suivant repart de là. Ctrl+C arrête le serveur.
#
# Si un serveur répond déjà sur le port, la page est simplement ouverte.
#
# Usage : ./run.sh [-p port] [-n] [-f] [-h]
#   -p  port d'écoute                       (défaut : 9015)
#   -n  ne pas ouvrir le navigateur
#   -f  recompiler cli/bids.wasm même s'il existe déjà
#   -h  affiche cette aide
set -euo pipefail

port="9015"
open_browser=1
force_wasm=0

usage() {
    sed -n '2,18p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while getopts "p:nfh" opt; do
    case "$opt" in
        p) port="$OPTARG" ;;
        n) open_browser=0 ;;
        f) force_wasm=1 ;;
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

case "$port" in
    ''|*[!0-9]*) echo "Port invalide : $port" >&2; exit 1 ;;
esac
if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
    echo "Port hors plage : $port (attendu : 1 à 65535)" >&2
    exit 1
fi

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
url="http://localhost:$port/"

# Git Bash et Cygwin exécutent bien un fichier sans extension, mais Go, lui,
# produit un binaire Windows : autant le nommer comme tel. Les deux noms sont
# déjà dans .gitignore.
bin="$root/bids"
case "${OSTYPE:-}" in
    msys*|cygwin*|win*) bin="$root/bids.exe" ;;
esac

# /ready ne répond qu'une fois le moteur capable de rejouer une donne : c'est
# le signal qu'attendre avant d'ouvrir la page, plutôt qu'un délai au jugé.
if command -v curl >/dev/null 2>&1; then
    probe() { curl -sf -m 2 "http://localhost:$port/ready" >/dev/null 2>&1; }
elif command -v wget >/dev/null 2>&1; then
    probe() { wget -q -T 2 -O /dev/null "http://localhost:$port/ready" >/dev/null 2>&1; }
else
    # Sans client HTTP on ne peut rien sonder : on laisse au serveur le temps
    # de se lever, une fois, et on ouvre.
    probe() { sleep 3; }
fi

open_url() {
    case "${OSTYPE:-}" in
        darwin*)
            open "$1"
            ;;
        msys*|cygwin*|win*)
            # « // » protège le /c du remplacement de chemins de MSYS, qui le
            # prendrait pour une racine Unix et le réécrirait en C:\.
            cmd.exe //c start "" "$1"
            ;;
        *)
            if command -v xdg-open >/dev/null 2>&1; then
                xdg-open "$1"
            else
                echo "Aucun lanceur de navigateur trouvé ; ouvrez $1 à la main."
                return 0
            fi
            ;;
    esac
}

# Serveur déjà debout : le recompiler et échouer sur « address already in use »
# n'apprendrait rien à personne.
if probe; then
    echo "Un serveur répond déjà sur le port $port."
    [ "$open_browser" -eq 0 ] || open_url "$url"
    exit 0
fi

command -v go >/dev/null 2>&1 || { echo "Go est introuvable dans le PATH." >&2; exit 1; }

# //go:embed all:cli fige le contenu de cli/ à la compilation : sans ce fichier,
# le mode « navigateur » du client répondrait 404 sur bids.wasm.
if [ "$force_wasm" -eq 1 ] || [ ! -f "$root/cli/bids.wasm" ]; then
    "$root/build-wasm.sh"
fi

revision="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"
echo "Compilation de $(basename "$bin") (revision $revision)..."
(
    cd "$root"
    CGO_ENABLED=0 go build -trimpath \
        -ldflags="-s -w -X main.buildRevision=$revision" -o "$bin" ./engine
)

# Le serveur garde le premier plan : Ctrl+C lui parvient directement et il
# referme ses connexions proprement. L'ouverture de la page, elle, attend en
# arrière-plan que /ready réponde.
if [ "$open_browser" -eq 1 ]; then
    (
        deadline=$((SECONDS + 60))
        while [ "$SECONDS" -lt "$deadline" ]; do
            if probe; then
                echo "Serveur prêt — ouverture de $url"
                open_url "$url"
                exit 0
            fi
            sleep 0.3
        done
        echo "Le serveur n'a pas répondu en 60 s ; ouvrez $url à la main." >&2
    ) &
    opener=$!
    # Le guetteur ne doit pas survivre à un démarrage avorté : sans cela il
    # tournerait encore une minute après l'échec du serveur.
    trap 'kill "$opener" 2>/dev/null || true' EXIT
fi

echo "Démarrage du serveur sur $url (Ctrl+C pour arrêter)"
PORT="$port" "$bin"
