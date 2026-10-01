#!/usr/bin/env python3
"""Régénère tout ce qui dérive des règles d'un système, après leur modification.

Un système est un dossier, cli/systems/<id>/ (rules.yaml, ses PDF, ses donnes thématiques
pbn/), déclaré dans cli/systems/index.json ; ses données de référence sont dans
engine/testdata/systems/<id>/. Pour chaque système (par défaut : tous ceux de index.json) :
  1. valide rules.yaml (le moteur refuserait un fichier invalide en entier) ;
  2. exporte les règles expansées et les cas de test (rules.json, tests.json) ;
  3. enchérit les donnes de référence (golden.json) ;
  4. produit la description en PDF (rules.pdf, rules.en.pdf), au nom que lui donne index.json ;
  5. crée la référence du banc du par (par_baseline.json) si elle manque, sans jamais la
     réécrire sans --par-update : une dégradation doit rester visible dans le diff de la PR.
Les tests Go (engine/systems_test.go) cherchent ces mêmes fichiers.

Usage : python regen_system.py [sef new ...] [--par-update] [--no-pdf]
"""
import argparse
import json
import os
import shutil
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, "..", ".."))
SYSTEMS = os.path.join(ROOT, "cli", "systems")
DATA = os.path.join(ROOT, "engine", "testdata", "systems")
sys.path.insert(0, HERE)
import sef_rules as sr  # noqa: E402


def run(cmd, cwd=HERE, env=None):
    print("  $ " + " ".join(cmd), flush=True)
    subprocess.run(cmd, cwd=cwd, env=env, check=True)


def regen(entry, par_update, pdf):
    sid = entry["id"]
    rules = os.path.join(SYSTEMS, sid, "rules.yaml")
    data = os.path.join(DATA, sid)
    os.makedirs(data, exist_ok=True)
    print("== %s" % sid, flush=True)

    errs = sr.validate(sr.load(rules))
    if errs:
        for e in errs[:20]:
            print("  " + str(e))
        sys.exit("%s : %d erreur(s), rien n'est régénéré" % (sid, len(errs)))

    py = sys.executable
    run([py, "sef_rules.py", rules,
         "--json", os.path.join(data, "rules.json"),
         "--gen-tests", "1500", os.path.join(data, "tests.json"), "--seed", "2024"])
    run([py, "gen_golden.py", "--rules", rules, "--out", os.path.join(data, "golden.json")])

    if pdf:
        name = entry.get("name", {})
        for lang, out in (("FR", "rules.pdf"), ("EN", "rules.en.pdf")):
            title = name.get(lang.lower()) or name.get("fr") or sid
            run([py, "rules_pdf.py", rules, "-o", os.path.join(SYSTEMS, sid, out), "--lang", lang, "--title", title])

    if par_update or not os.path.exists(os.path.join(data, "par_baseline.json")):
        cmd = ["go", "test", "-run", "TestParBenchmark/^%s$" % sid, "./engine"]
        if shutil.which("go") is None:
            print("  Go introuvable : lancer depuis la racine  PAR_BENCH_UPDATE=1 %s" % " ".join(cmd))
        else:
            run(cmd, cwd=ROOT, env=dict(os.environ, PAR_BENCH_UPDATE="1"))


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("systems", nargs="*", help="identifiants des systèmes (défaut : tous ceux de cli/systems/index.json)")
    ap.add_argument("--par-update", action="store_true",
                    help="réécrire la référence du banc du par même si elle existe (dégradation voulue)")
    ap.add_argument("--no-pdf", action="store_true", help="ne pas régénérer les PDF")
    a = ap.parse_args()

    with open(os.path.join(SYSTEMS, "index.json"), encoding="utf-8") as f:
        systems = {s["id"]: s for s in json.load(f)["systems"]}
    for sid in a.systems or list(systems):
        if sid not in systems:
            sys.exit("%s n'est pas déclaré dans cli/systems/index.json : l'y ajouter d'abord" % sid)
        regen(systems[sid], a.par_update, not a.no_pdf)
    print("\nVérifier ensuite depuis la racine : go test ./...")


if __name__ == "__main__":
    main()
