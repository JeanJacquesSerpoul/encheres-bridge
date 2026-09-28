#!/usr/bin/env bash
# Reprend les captures d'écran du tutoriel du client.
#
#   tools/tutorial/run.sh
#
# Sert cli/ en local le temps des captures, puis écrit
# cli/tutorial/fr/NN.jpg et cli/tutorial/en/NN.jpg. À relancer après un
# changement visible de l'interface. Demande Node et Playwright (Chromium).
#
# Les captures sont prises dans cli/tutorial.new/ et ne remplacent
# cli/tutorial/ qu'une fois toutes réussies : un échec laisse les images en
# place.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"
port="${TUTORIAL_PORT:-9377}"

out="$root/cli/tutorial.new"
rm -rf "$out"

python3 -m http.server "$port" --bind 127.0.0.1 --directory "$root/cli" >/dev/null 2>&1 &
server=$!
trap 'kill "$server" 2>/dev/null || true; rm -rf "$out"' EXIT

# Le serveur met un instant à écouter.
for _ in $(seq 1 50); do
  curl -fs -o /dev/null "http://127.0.0.1:$port/index.html" && break
  sleep 0.1
done

node "$here/capture.js" "http://127.0.0.1:$port/index.html" "$out"
rm -rf "$root/cli/tutorial"
mv "$out" "$root/cli/tutorial"
du -sh "$root/cli/tutorial"
