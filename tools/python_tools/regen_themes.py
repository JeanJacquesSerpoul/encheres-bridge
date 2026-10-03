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

Les recettes sont écrites pour 25 donnes ; SCALE les multiplie (nombre de
donnes et plafond par règle), proportions gardées : 500 donnes par série. Le
tirage se répartit sur plusieurs processus (--jobs) : chaque tirage est coupé
en tranches, chacune avec sa graine dérivée, et le résultat reste
reproductible pour un nombre de tranches donné.

Usage : python regen_themes.py [sef] [--only drury roudi ...] [--jobs N]
"""
import argparse
import math
import multiprocessing
import os
import random

import gen_theme_pbn as gt

HERE = os.path.dirname(os.path.abspath(__file__))
SYSTEMS = os.path.normpath(os.path.join(HERE, "..", "..", "cli", "systems"))
MIX_SEED = 2024

# Chaque série : 25 × SCALE donnes.
SCALE = 20
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
        # Les huit barrages, trois donnes au plus pour chacun.
        {"file": "barrages.pbn", "fr": "Ouvertures de barrage au palier de 3 et de 4",
         "en": "Preemptive openings at the 3 and 4 level",
         "parts": [(r"^open\.[34][CDHS]$", 25, 2024, 4)]},
        # Toutes les réponses autres que passe, variées (trois donnes au plus par réponse).
        {"file": "reponses-mineure.pbn", "fr": "Réponses sur une ouverture mineure",
         "en": "Responses to a minor-suit opening",
         "parts": [(r"^1[CD]\.r\.(?!pass$)", 25, 2024, 3)]},
        {"file": "reponses-majeure.pbn", "fr": "Réponses sur une ouverture majeure",
         "en": "Responses to a major-suit opening",
         "parts": [(r"^1[HS]\.r\.(?!pass$)", 25, 2024, 3)]},
        {"file": "reponses-1sa-2sa.pbn", "fr": "Réponses à l'ouverture d'1SA et de 2SA",
         "en": "Responses to 1NT and 2NT openings",
         "parts": [(r"^1NT\.r\.(?!pass$)", 17, 2024, 2),
                   (r"^2NT\.r\.(?!pass$)", 8, 2025, 3)]},
        {"file": "spoutnik.pbn", "fr": "Spoutnik", "en": "Negative double (Spoutnik)",
         "parts": [(r"^spoutnik\.1[CD]\.1[DS]$", 25, 2024, 9)]},
        # Interventions par une couleur ou à 1SA ; le contre d'appel a sa série.
        {"file": "interventions.pbn", "fr": "Interventions", "en": "Overcalls",
         "parts": [(r"^def\.1[CDHS]\.(?!pass$|X)", 25, 2024, 2)]},
        {"file": "reveils.pbn", "fr": "Réveils", "en": "Balancing (reopening) bids",
         "parts": [(r"^rev\.1[CDHS]\.(?!pass$)", 25, 2024, 2)]},
        # La deuxième enchère de l'ouvreur après une réponse (passe exclu).
        {"file": "redemandes-ouvreur.pbn", "fr": "Redemandes de l'ouvreur", "en": "Opener's rebids",
         "parts": [(r"^1[CDHS]\.(1[DHS]|1NT|2[CDHS]|2NT)\.(?!pass$)[^.]+(\.(reverse|fort))?$", 25, 2024, 1)]},
        # Landy en intervention et en réveil sur 1SA.
        {"file": "landy.pbn", "fr": "Landy", "en": "Landy (over 1NT)",
         "parts": [(r"^def1NT\.2C$", 18, 2024, 0),
                   (r"^rev1NT\.2C$", 7, 2025, 0)]},
    ],
}


_rules = None


def _init(path):
    global _rules
    _rules = gt.load_rules(path)


def _draw(job):
    rule, count, seed, cap = job
    return gt.draw(_rules, rule, count, seed, cap)


def slices(rule, count, seed, cap, n):
    """Le tirage (rule, count, seed, cap) coupé en n tranches au plus : les
    donnes se répartissent, le plafond par règle aussi, et chaque tranche a sa
    graine dérivée de celle du tirage."""
    n = max(1, min(n, count))
    sizes = [count // n + (1 if i < count % n else 0) for i in range(n)]
    sub_cap = math.ceil(cap / n) if cap else 0
    return [(rule, c, seed * 1000 + i, sub_cap) for i, c in enumerate(sizes)]


def regen(system, theme, pool, jobs):
    deals, seen = [], set()
    for rule, count, seed, cap in theme["parts"]:
        count, cap = count * SCALE, cap * SCALE
        got, tried, per_rule = [], 0, {}
        for g, t, pr in pool.map(_draw, slices(rule, count, seed, cap, jobs)):
            got += g
            tried += t
            for k, v in pr.items():
                per_rule[k] = per_rule.get(k, 0) + v
        fresh = [d for d in got if tuple(d[2]) not in seen]
        seen.update(tuple(d[2]) for d in fresh)
        deals += fresh
        print(f"  {rule} : {len(fresh)}/{count} donnes sur {tried} tirées ; "
              + ", ".join(f"{k} {v}" for k, v in sorted(per_rule.items())), flush=True)
    if len(theme["parts"]) > 1:
        random.Random(MIX_SEED).shuffle(deals)
    out = os.path.join(SYSTEMS, system, "pbn", theme["file"])
    gt.write_pbn(out, theme["fr"], theme["en"], deals)
    print(f"  -> {out} : {len(deals)} donnes")


def main():
    ap = argparse.ArgumentParser(description="Régénère les donnes thématiques d'un système")
    ap.add_argument("system", nargs="?", default="sef", choices=sorted(THEMES))
    ap.add_argument("--only", nargs="*", help="fichiers à régénérer (tous par défaut)")
    ap.add_argument("--jobs", type=int, default=max(1, (os.cpu_count() or 2) - 1),
                    help="processus de tirage (défaut : les cœurs moins un)")
    args = ap.parse_args()
    path = os.path.join(SYSTEMS, args.system, "rules.yaml")
    with multiprocessing.Pool(args.jobs, initializer=_init, initargs=(path,)) as pool:
        for theme in THEMES[args.system]:
            if args.only and theme["file"] not in args.only and theme["file"][:-4] not in args.only:
                continue
            print(f"== {theme['file']}", flush=True)
            regen(args.system, theme, pool, args.jobs)


if __name__ == "__main__":
    main()
