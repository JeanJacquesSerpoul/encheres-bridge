#!/usr/bin/env python3
"""Régénère les séries de donnes thématiques d'un système, avec ses règles actuelles.

Les donnes d'une série ont été choisies parce que leurs enchères emploient une
convention : quand les règles changent, certaines ne l'emploient plus. Ce script
garde la recette de chaque série (expression sur l'id de la règle, nombre de
donnes, graine, plafond par règle) et la rejoue avec gen_theme_pbn.py.

Une série peut mêler plusieurs tirages (« 2 faible et sa défense », « Contre
d'appel et ses réponses ») : chacun a sa graine, les donnes en double sont
écartées, puis l'ensemble est mêlé (graine MIX_SEED) et renuméroté. Chaque
donne garde le donneur et la vulnérabilité de son tirage, dont dépend
l'enchère.

Usage : python regen_themes.py [sef] [--only drury roudi ...]
"""
import argparse
import os
import random

import gen_theme_pbn as gt

HERE = os.path.dirname(os.path.abspath(__file__))
SYSTEMS = os.path.normpath(os.path.join(HERE, "..", "..", "cli", "systems"))
MIX_SEED = 2024

# Recettes à 25 donnes par série : (expression, nombre, graine, plafond par règle).
THEMES = {
    "sef": [
        {"file": "4e-couleur-forcing.pbn", "fr": "4e couleur forcing", "en": "Fourth suit forcing",
         "parts": [(r"^fcf\.[a-z]+\.ask$", 25, 2024, 0)]},
        {"file": "drury.pbn", "fr": "Drury", "en": "Drury",
         "parts": [(r"^drury\.[HS]\.(2C|2NT)$", 25, 2024, 0)]},
        {"file": "2-trefle-fort.pbn", "fr": "2♣ fort indéterminé", "en": "Strong 2♣ (catch-all)",
         "parts": [(r"^open\.2C$", 25, 2024, 0)]},
        {"file": "roudi.pbn", "fr": "Roudi", "en": "Roudi (checkback after 1NT rebid)",
         "parts": [(r"^roudi\.[CDH]\.[HS]$", 25, 2024, 0)]},
        {"file": "2-carreau-fm.pbn", "fr": "2♦ forcing de manche", "en": "Game-forcing 2♦",
         "parts": [(r"^open\.2D$", 25, 2024, 0)]},
        # La défense, variée (deux donnes au plus par action, le réveil compris),
        # puis le 2SA forcing de manche de Lévy après le contre.
        {"file": "2-faible.pbn", "fr": "2 faible (2♥ / 2♠) et sa défense",
         "en": "Weak two (2♥ / 2♠) and its defence",
         "parts": [(r"^d2f\.2[HS]\.(?!pass$|rev\.pass$)", 19, 2024, 2),
                   (r"^d2f\.2[HS]\.X\.2NT$", 6, 2025, 0)]},
        # Les trois zones de réponse au contre d'appel : 0-7 H (réponse sans
        # saut), 8-10 H (sauts, cue-bid bimajeur, 1SA), 11 H et plus (cue-bid,
        # 2SA, manche en majeure, et un 3SA).
        {"file": "contre-appel.pbn", "fr": "Contre d'appel et ses réponses",
         "en": "Takeout double and its answers",
         "parts": [(r"^advX\.1[CDHS]\.[12][CDHS]\.\d+$", 8, 2024, 2),
                   (r"^advX\.1[CDHS]\.(saut\.[CDHS]|dsaut\.[HS]|cue\.2M|1NT)$", 10, 2025, 2),
                   (r"^advX\.1[CDHS]\.(cue|2NT|4[HS])$", 6, 2026, 2),
                   (r"^advX\.1[CDHS]\.3NT$", 1, 2027, 0)]},
    ],
}


def regen(system, theme, rules):
    deals, seen = [], set()
    for rule, count, seed, cap in theme["parts"]:
        got, tried, per_rule = gt.draw(rules, rule, count, seed, cap)
        fresh = [d for d in got if tuple(d[2]) not in seen]
        seen.update(tuple(d[2]) for d in fresh)
        deals += fresh
        print(f"  {rule} : {len(fresh)}/{count} donnes sur {tried} tirées ; "
              + ", ".join(f"{k} {v}" for k, v in sorted(per_rule.items())))
    if len(theme["parts"]) > 1:
        random.Random(MIX_SEED).shuffle(deals)
    out = os.path.join(SYSTEMS, system, "pbn", theme["file"])
    gt.write_pbn(out, theme["fr"], theme["en"], deals)
    print(f"  -> {out} : {len(deals)} donnes")


def main():
    ap = argparse.ArgumentParser(description="Régénère les donnes thématiques d'un système")
    ap.add_argument("system", nargs="?", default="sef", choices=sorted(THEMES))
    ap.add_argument("--only", nargs="*", help="fichiers à régénérer (tous par défaut)")
    args = ap.parse_args()
    rules = gt.load_rules(os.path.join(SYSTEMS, args.system, "rules.yaml"))
    for theme in THEMES[args.system]:
        if args.only and theme["file"] not in args.only and theme["file"][:-4] not in args.only:
            continue
        print(f"== {theme['file']}", flush=True)
        regen(args.system, theme, rules)


if __name__ == "__main__":
    main()
