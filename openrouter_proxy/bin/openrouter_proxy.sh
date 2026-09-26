#!/usr/bin/env bash
# openrouter_proxy.sh — lance le serveur IA compilé par build-proxy.sh
# (bin/openrouter_proxy, Linux amd64) sur http://localhost:<port>/. Ctrl+C l'arrête.
#
# Le serveur lit son .env dans le dossier courant : le script se place donc
# dans openrouter_proxy/, où se trouve .env (à copier depuis .env.example).
# Les variables déjà présentes dans l'environnement l'emportent sur .env.
#
# Usage : bin/openrouter_proxy.sh [-p port] [-h]
#   -p  port d'écoute     (défaut : PORT du .env, sinon 9013)
#   -h  affiche cette aide
set -euo pipefail

port=""

usage() {
    sed -n '2,11p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while getopts "p:h" opt; do
    case "$opt" in
        p) port="$OPTARG" ;;
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

if [ -n "$port" ]; then
    case "$port" in
        *[!0-9]*) echo "Port invalide : $port" >&2; exit 1 ;;
    esac
    if [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        echo "Port hors plage : $port (attendu : 1 à 65535)" >&2
        exit 1
    fi
    export PORT="$port"
fi

bindir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(dirname "$bindir")"

# Git Bash et Cygwin prennent le binaire Windows, le seul qu'ils sachent
# exécuter.
bin="$bindir/openrouter_proxy"
case "${OSTYPE:-}" in
    msys*|cygwin*|win*) bin="$bindir/openrouter_proxy.exe" ;;
esac

[ -f "$bin" ] || { echo "Binaire introuvable : $bin (lancez d'abord build-proxy.sh ou make bin)." >&2; exit 1; }
# Un clone ou une archive peut perdre le bit d'exécution.
[ -x "$bin" ] || chmod +x "$bin"

if [ ! -f "$root/.env" ] && [ -z "${OPENROUTER_API_KEY:-}" ]; then
    echo "Ni $root/.env ni OPENROUTER_API_KEY : copiez .env.example en .env et renseignez la clé." >&2
    exit 1
fi

cd "$root"
exec "$bin"
