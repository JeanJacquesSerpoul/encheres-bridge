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

Les séries du SEF portent désormais leurs enchères : elles n'ont plus de
recette ici (voir THEMES).

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
# Les séries du SEF portent désormais leurs enchères et leurs commentaires
# ([Auction]), écrits à la main ou enchéris de bout en bout : les régénérer
# les remplacerait par des donnes sans enchères. Leurs recettes sont retirées ;
# une nouvelle série tirée sans enchères peut encore s'ajouter ici.
THEMES = {
    "sef": [],
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
