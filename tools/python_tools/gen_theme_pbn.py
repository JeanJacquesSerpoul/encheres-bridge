"""Génère un fichier PBN de donnes thématiques pour cli/systems/<id>/pbn/.

Tire des donnes au hasard et garde celles dont les enchères, selon les règles,
emploient une règle dont l'id correspond à l'expression donnée. Le fichier
commence par son libellé en français et en anglais, que la fenêtre « Donnes
thématiques » de l'application affiche :

    % Titre-FR: 4e couleur forcing
    % Titre-EN: Fourth suit forcing

Les donnes sont enchéries avec les règles du système (--rules, le SEF par défaut) : un
thème se range dans le dossier pbn/ de ce système, et se déclare dans son index.json.

Exemple :
    python gen_theme_pbn.py "^fcf\\.[a-z]+\\.ask$" ../../cli/systems/sef/pbn/4e-couleur-forcing.pbn \\
        --fr "4e couleur forcing" --en "Fourth suit forcing"
"""
import argparse
import os
import random
import re
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
import sef_rules as sr  # noqa: E402
from pbn_auction import generate_auction  # noqa: E402

SEATS = "NESW"
# Donneur et vulnérabilité de la donne n (1 à 16), comme en tournoi.
VUL = ["None", "NS", "EW", "All", "NS", "EW", "All", "None",
       "EW", "All", "None", "NS", "All", "None", "NS", "EW"]
RANKS = "AKQJT98765432"


def random_deal(rng):
    """Quatre mains 'S.H.D.C', de Nord à Ouest."""
    cards = [(s, r) for s in range(4) for r in RANKS]
    rng.shuffle(cards)
    hands = []
    for k in range(4):
        hand = sorted(cards[k * 13:(k + 1) * 13], key=lambda c: RANKS.index(c[1]))
        hands.append(".".join("".join(r for s, r in hand if s == suit) for suit in range(4)))
    return hands


def main():
    ap = argparse.ArgumentParser(description="Donnes thématiques au format PBN")
    ap.add_argument("rule", help="expression régulière sur l'id de la règle recherchée")
    ap.add_argument("output", help="fichier PBN à écrire")
    ap.add_argument("--fr", required=True, help="libellé en français")
    ap.add_argument("--en", required=True, help="libellé en anglais")
    ap.add_argument("-n", "--count", type=int, default=50, help="nombre de donnes (50)")
    ap.add_argument("--seed", type=int, default=2024, help="graine du tirage (2024)")
    ap.add_argument("--max-per-rule", type=int, default=0,
                    help="au plus N donnes par règle retenue, pour équilibrer les variantes du thème (0 : sans limite)")
    ap.add_argument("--rules", default=os.path.normpath(os.path.join(HERE, "../../cli/systems/sef/rules.yaml")))
    args = ap.parse_args()

    rules = sr.load(args.rules)
    errs = sr.validate(rules)
    if errs:
        sys.exit("; ".join(errs[:3]))
    want = re.compile(args.rule)
    rng = random.Random(args.seed)

    games, tried, per_rule = [], 0, {}
    while len(games) < args.count:
        tried += 1
        n = len(games) + 1
        dealer, vul = SEATS[(n - 1) % 4], VUL[(n - 1) % 16]
        hands = random_deal(rng)
        bids = generate_auction(rules, dict(zip(SEATS, hands)), dealer, (), "FR", False, vul)
        hit = next((b["rule"] for b in bids if b["rule"] and want.search(b["rule"])), None)
        if hit is None or (args.max_per_rule and per_rule.get(hit, 0) >= args.max_per_rule):
            continue
        per_rule[hit] = per_rule.get(hit, 0) + 1
        games.append("\n".join([
            f'[Event "{args.fr}"]', f'[Board "{n}"]', f'[Dealer "{dealer}"]',
            f'[Vulnerable "{vul}"]', f'[Deal "N:{" ".join(hands)}"]']))

    head = f"% Titre-FR: {args.fr}\n% Titre-EN: {args.en}\n"
    with open(args.output, "w", encoding="utf-8", newline="\n") as f:
        f.write(head + "\n" + "\n\n".join(games) + "\n")
    print(f"{len(games)} donnes retenues sur {tried} tirées -> {args.output}")
    print("par règle : " + ", ".join(f"{k} {v}" for k, v in sorted(per_rule.items())))


if __name__ == "__main__":
    main()
