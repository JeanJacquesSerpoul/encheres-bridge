#!/usr/bin/env bash
# update-system.sh — met à jour un système d'enchères après modification de
# son fichier de règles (cli/rules/<système>.yaml).
#
# Régénère tout ce qui en dérive (données de test, enchères de référence,
# PDF, référence du banc du par s'il n'en a pas), puis lance les tests.
# Avec -p, publie ensuite la modification : branche (si l'on est sur main),
# commit, push et pull request.
#
# Usage : ./update-system.sh [-a] [-p] [-m message] [-h] <système>
#   <système>  nom du système : new, ou new.yaml (voir cli/rules/index.json)
#   -a  accepter une hausse de l'écart au par (réécrit la référence du banc)
#   -p  publier : branche, commit, push et pull request (gh)
#   -m  message du commit et titre de la PR (défaut : « Système <nom> : mise à jour des règles »)
#   -h  affiche cette aide
set -euo pipefail

usage() {
    sed -n '2,15p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

accept_par=0
publish=0
message=""
while getopts "apm:h" opt; do
    case "$opt" in
        a) accept_par=1 ;;
        p) publish=1 ;;
        m) message="$OPTARG" ;;
        h) usage; exit 0 ;;
        *) usage >&2; exit 1 ;;
    esac
done
shift $((OPTIND - 1))
[ $# -eq 1 ] || { usage >&2; exit 1; }

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
name="$(basename "$1")"
stem="${name%.yaml}"
stem="${stem%.yml}"
file="$stem.yaml"
[ -f "$root/cli/rules/$file" ] || { echo "cli/rules/$file est introuvable." >&2; exit 1; }
[ -n "$message" ] || message="Système $stem : mise à jour des règles"

# Le premier Python qui sait lire le YAML : python3 et python ne sont pas
# toujours le même interpréteur (sous Windows notamment).
python=""
for p in python3 python; do
    if command -v "$p" >/dev/null 2>&1 && "$p" -c "import yaml" >/dev/null 2>&1; then
        python="$p"
        break
    fi
done
[ -n "$python" ] || { echo "Aucun Python avec PyYAML dans le PATH (pip install pyyaml)." >&2; exit 1; }
command -v go >/dev/null 2>&1 || { echo "Go est introuvable dans le PATH." >&2; exit 1; }

echo "== Régénération de $file"
regen_args=("../../cli/rules/$file")
[ "$accept_par" -eq 1 ] && regen_args+=(--par-update)
(cd "$root/tools/python_tools" && "$python" regen_system.py "${regen_args[@]}")

echo "== Tests"
if ! (cd "$root" && go test ./...); then
    echo >&2
    echo "Les tests échouent. Si seul l'écart au par a augmenté et que c'est voulu :" >&2
    echo "  ./update-system.sh -a $stem" >&2
    exit 1
fi

if [ "$publish" -eq 0 ]; then
    echo
    echo "$file est à jour et les tests passent. Pour publier : ./update-system.sh -p $stem"
    exit 0
fi

echo "== Publication"
cd "$root"
branch="$(git rev-parse --abbrev-ref HEAD)"
if [ "$branch" = "main" ]; then
    branch="rules/$stem-$(date +%Y%m%d-%H%M)"
    git switch -c "$branch"
fi
# Seuls les fichiers des systèmes d'enchères : règles, PDF, index et données de test.
# Motifs entre guillemets : git les applique lui-même, fichiers supprimés compris.
git add -A -- cli/rules 'engine/testdata/*.json' 'tools/python_tools/*.json'
if git diff --cached --quiet; then
    echo "Rien à publier : aucun changement."
    exit 0
fi
git commit -m "$message"
git push -u origin "$branch"
if command -v gh >/dev/null 2>&1; then
    gh pr view "$branch" >/dev/null 2>&1 || gh pr create --base main --title "$message" \
        --body "Mise à jour du système d'enchères \`$file\`, données de test et PDF régénérés par update-system.sh."
else
    echo "gh est introuvable : ouvrir la pull request de $branch sur GitHub."
fi
