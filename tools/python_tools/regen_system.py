#!/usr/bin/env python3
"""Régénère tout ce qui dérive d'un fichier de règles, après sa modification.

Pour chaque système (par défaut : tous ceux de cli/rules/index.json) :
  1. valide le fichier (le moteur refuserait un fichier invalide en entier) ;
  2. exporte les règles expansées et les cas de test (sef_rules.py --json --gen-tests) ;
  3. enchérit les donnes de référence (gen_golden.py) ;
  4. produit les PDF que index.json lui déclare, sous le nom qu'il lui donne ;
  5. crée la référence du banc du par si elle manque (jamais réécrite sans --par-update :
     une dégradation doit rester visible dans le diff de la PR).

default.yaml garde les noms historiques (sef_tests.json, golden_python.json...) ; un autre
système ajoute son nom avant l'extension : sef_tests.mon-systeme.json. Les tests Go
(engine/systems_test.go) cherchent ces mêmes fichiers.

Usage : python regen_system.py [../../cli/rules/mon-systeme.yaml ...] [--par-update] [--no-pdf]
"""
import argparse
import json
import os
import shutil
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, "..", ".."))
RULES_DIR = os.path.join(ROOT, "cli", "rules")
TESTDATA = os.path.join(ROOT, "engine", "testdata")
sys.path.insert(0, HERE)
import sef_rules as sr  # noqa: E402


def suffixed(path, stem):
    """Le fichier de référence d'un système : inchangé pour default, nom.<système>.ext sinon."""
    if stem == "default":
        return path
    base, ext = os.path.splitext(path)
    return "%s.%s%s" % (base, stem, ext)


def run(cmd, cwd=HERE, env=None):
    print("  $ " + " ".join(cmd), flush=True)
    subprocess.run(cmd, cwd=cwd, env=env, check=True)


def regen(entry, par_update, pdf):
    rules = os.path.join(RULES_DIR, entry["file"])
    stem = os.path.splitext(entry["file"])[0]
    print("== %s" % entry["file"], flush=True)

    errs = sr.validate(sr.load(rules))
    if errs:
        for e in errs[:20]:
            print("  " + str(e))
        sys.exit("%s : %d erreur(s), rien n'est régénéré" % (entry["file"], len(errs)))

    py = sys.executable
    run([py, "sef_rules.py", rules,
         "--json", suffixed("sef_rules.json", stem),
         "--gen-tests", "1500", suffixed("sef_tests.json", stem), "--seed", "2024"])
    run([py, "gen_golden.py", "--rules", rules])

    pdfs = entry.get("pdf") or {}
    if isinstance(pdfs, str):
        pdfs = {"fr": pdfs}
    for lang, name in (pdfs.items() if pdf else ()):
        title = entry.get("name", {}).get(lang) or entry.get("name", {}).get("fr") or stem
        run([py, "rules_pdf.py", rules, "-o", os.path.join(RULES_DIR, name),
             "--lang", lang.upper(), "--title", title])

    baseline = suffixed(os.path.join(TESTDATA, "par_bench_baseline.json"), stem)
    if par_update or not os.path.exists(baseline):
        cmd = ["go", "test", "-run", "TestParBenchmark/^%s$" % stem, "./engine"]
        if shutil.which("go") is None:
            print("  Go introuvable : lancer depuis la racine  PAR_BENCH_UPDATE=1 %s" % " ".join(cmd))
        else:
            run(cmd, cwd=ROOT, env=dict(os.environ, PAR_BENCH_UPDATE="1"))


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("rules", nargs="*", help="fichiers de règles (défaut : tous ceux de cli/rules/index.json)")
    ap.add_argument("--par-update", action="store_true",
                    help="réécrire la référence du banc du par même si elle existe (dégradation voulue)")
    ap.add_argument("--no-pdf", action="store_true", help="ne pas régénérer les PDF")
    a = ap.parse_args()

    with open(os.path.join(RULES_DIR, "index.json"), encoding="utf-8") as f:
        systems = {s["file"]: s for s in json.load(f)["systems"]}
    wanted = [os.path.basename(r) for r in a.rules] or list(systems)
    for name in wanted:
        if name not in systems:
            sys.exit("%s n'est pas déclaré dans cli/rules/index.json : l'y ajouter d'abord" % name)
        regen(systems[name], a.par_update, not a.no_pdf)
    print("\nVérifier ensuite depuis la racine : go test ./...")


if __name__ == "__main__":
    main()
