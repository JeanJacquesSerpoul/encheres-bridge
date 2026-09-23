#!/usr/bin/env bash
# run-macos.command — lanceur macOS : compile puis lance le serveur, et ouvre
# l'application dans le navigateur sur http://localhost:<port>/ .
#
# L'extension .command le rend lançable d'un double-clic dans le Finder, qui
# l'ouvre dans une fenêtre du Terminal. Depuis un terminal, il accepte les
# options de run.sh, auquel il délègue tout le travail :
#
#   ./run-macos.command            # port 9015, ouvre le navigateur
#   ./run-macos.command -p 9200    # autre port
#   ./run-macos.command -n         # ne pas ouvrir le navigateur
#   ./run-macos.command -f         # recompiler cli/bids.wasm au passage
#
# Prérequis : Go (https://go.dev/dl/ ou `brew install go`). Ctrl+C arrête le
# serveur.
set -euo pipefail

# Le Finder lance le script depuis le dossier personnel : on se replace à côté
# de lui, là où sont les sources.
root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$root"

# Lancé d'un double-clic, le Terminal hérite d'un PATH minimal qui ignore
# souvent le shell de connexion : Go installé par le paquet officiel ou par
# Homebrew n'y figure pas. On ajoute leurs emplacements habituels.
for dir in /usr/local/go/bin /opt/homebrew/bin /usr/local/bin "$HOME/go/bin"; do
    case ":$PATH:" in
        *":$dir:"*) ;;
        *) [ -d "$dir" ] && PATH="$PATH:$dir" ;;
    esac
done
export PATH

# Un échec doit rester lisible : fermée aussitôt, la fenêtre du Terminal
# emporterait le message avec elle. Ctrl+C (130) n'est pas un échec.
pause_on_error() {
    status=$?
    if [ "$status" -ne 0 ] && [ "$status" -ne 130 ] && [ -t 0 ]; then
        echo
        read -r -p "Échec (code $status). Appuyez sur Entrée pour fermer..." _ || true
    fi
}
trap pause_on_error EXIT

if ! command -v go >/dev/null 2>&1; then
    echo "Go est introuvable. Installez-le depuis https://go.dev/dl/ ou avec :" >&2
    echo "    brew install go" >&2
    exit 1
fi

# Une archive téléchargée depuis GitHub peut perdre le bit d'exécution.
chmod +x run.sh build-wasm.sh 2>/dev/null || true

# ${1+"$@"} et non "$@" : sous `set -u`, le bash 3.2 livré avec macOS tient un
# "$@" vide pour une variable non définie.
./run.sh ${1+"$@"}
