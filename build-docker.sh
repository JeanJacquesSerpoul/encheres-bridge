#!/usr/bin/env bash
# build-docker.sh — construit l'image Docker du serveur (Dockerfile multi-étapes).
# La révision Git courte est injectée dans le binaire via --build-arg REVISION,
# de sorte que /version renvoie le bon commit (voir README).
#
# Usage : ./build-docker.sh [-t nom:tag] [-p plateforme] [-n] [-r] [-h]
#   -t  nom:tag de l'image                 (défaut : bridge-bids:latest)
#   -p  plateforme cible                   (ex. linux/amd64, linux/arm64)
#   -n  build sans cache (--no-cache)
#   -r  démarre le conteneur après le build et vérifie /ready
#   -h  affiche cette aide
set -euo pipefail

image="bridge-bids:latest"
platform=""
build_opts=()
run_after=0

usage() {
    sed -n '2,11p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while getopts "t:p:nrh" opt; do
    case "$opt" in
        t) image="$OPTARG" ;;
        p) platform="$OPTARG" ;;
        n) build_opts+=(--no-cache) ;;
        r) run_after=1 ;;
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done

# Sous WSL, /usr/bin/docker est un lien vers Docker Desktop : il pend tant que
# Docker Desktop n'est pas démarré, d'où le message commun aux deux contrôles.
command -v docker >/dev/null 2>&1 || {
    echo "Docker est introuvable (sous WSL : démarrez Docker Desktop et activez l'intégration WSL)." >&2
    exit 1
}
docker info >/dev/null 2>&1 || {
    echo "Le démon Docker ne répond pas (démarrez Docker Desktop / le service docker)." >&2
    exit 1
}

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[ -f "$root/Dockerfile" ] || { echo "Dockerfile introuvable dans $root" >&2; exit 1; }

revision="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"
# Le dépôt modifié ne correspond plus au commit : on le signale.
if ! git -C "$root" diff --quiet HEAD 2>/dev/null; then
    echo "Attention : dépôt modifié, la révision $revision ne reflète pas exactement l'image."
fi

port="${PORT:-9015}"
container="bridge-bids-check"

echo "Construction de $image (revision $revision${platform:+, $platform})..."
if [ -n "$platform" ]; then build_opts+=(--platform "$platform"); fi
docker build "${build_opts[@]}" \
    --build-arg "REVISION=$revision" \
    -t "$image" \
    "$root"

size_mo=$(awk -v b="$(docker image inspect -f '{{.Size}}' "$image")" \
    'BEGIN { printf "%.1f", b / 1048576 }')
echo "Terminé : $image (${size_mo} Mo)."

[ "$run_after" -eq 1 ] || {
    echo "Lancez le serveur avec : docker run -d -p ${port}:9015 --name bridge-bids $image"
    exit 0
}

# Vérification : démarrage du conteneur puis attente de /ready (le moteur rejoue
# une donne de référence, donc un 200 atteste d'un moteur fonctionnel).
docker rm -f "$container" >/dev/null 2>&1 || true
echo "Démarrage de $container sur le port $port..."
if ! docker run -d --name "$container" -p "${port}:9015" "$image" >/dev/null; then
    echo "docker run a échoué : le port $port est peut-être déjà occupé (relancez avec PORT=9415 $0 -r)." >&2
    exit 1
fi

for _ in $(seq 30); do
    if docker exec "$container" wget -qO- http://localhost:9015/ready >/dev/null 2>&1; then
        echo "OK : /ready répond. Version :"
        docker exec "$container" wget -qO- http://localhost:9015/version || true
        echo
        echo "Conteneur $container laissé actif sur http://localhost:${port}/"
        echo "Pour l'arrêter : docker rm -f $container"
        exit 0
    fi
    sleep 1
done

echo "Échec : /ready n'a pas répondu en 30 s. Journaux :" >&2
docker logs "$container" >&2 || true
docker rm -f "$container" >/dev/null 2>&1 || true
exit 1
