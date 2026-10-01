#!/usr/bin/env python3
"""Génère les enchères de référence du moteur Go (engine/testdata/golden_python.json).

Chaque donne est enchérie par pbn_auction.generate_auction (les quatre mains, adversaires
compris) avec un fichier de règles, cli/rules/default.yaml par défaut ; le test Go
TestGoldenPython rejoue les mêmes donnes et compare siège, enchère, règle et commentaire.
Un autre système écrit son propre fichier : mon-systeme.yaml → golden_python.mon-systeme.json.

Usage : python gen_golden.py [--rules ../../cli/rules/mon-systeme.yaml] [--random 300]
"""
import argparse
import glob
import gzip
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.normpath(os.path.join(HERE, "..", ".."))
sys.path.insert(0, HERE)
import sef_rules as sr          # noqa: E402
import pbn_auction as pa        # noqa: E402

RULES = os.path.join(ROOT, "cli", "rules", "default.yaml")
TESTDATA = os.path.join(ROOT, "engine", "testdata")
BENCH = os.path.join(ROOT, "engine", "testdata", "par_bench.jsonl.gz")


def deals_from_pbn():
    for path in sorted(glob.glob(os.path.join(ROOT, "engine", "testdata", "*.pbn"))):
        text = open(path, encoding="utf-8-sig").read()
        for i, lines in enumerate(pa.split_games(text)):
            tags, _ = pa.parse_game(lines)
            if "Deal" not in tags or "Dealer" not in tags:
                continue
            yield "%s#%d" % (os.path.basename(path), i + 1), tags["Dealer"].upper(), tags.get("Vulnerable", "None"), tags["Deal"]


def deals_from_bench(n):
    with gzip.open(BENCH, "rt", encoding="utf-8") as f:
        for i, line in enumerate(f):
            if i >= n:
                break
            b = json.loads(line)
            yield "bench#" + b["id"], b["dealer"], b["vul"], b["deal"]


def out_path(rules_path):
    """golden_python.json pour default.yaml, golden_python.<système>.json sinon (voir engine/systems_test.go)."""
    stem = os.path.splitext(os.path.basename(rules_path))[0]
    name = "golden_python.json" if stem == "default" else "golden_python.%s.json" % stem
    return os.path.join(TESTDATA, name)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--rules", default=RULES, help="fichier de règles (défaut : cli/rules/default.yaml)")
    ap.add_argument("--random", type=int, default=300, help="donnes prises au début du banc par")
    a = ap.parse_args()
    OUT = out_path(a.rules)
    rules = sr.load(a.rules)
    errs = sr.validate(rules)
    if errs:
        sys.exit("règles invalides : %s" % errs[:5])
    out = []
    for name, dealer, vul, deal in list(deals_from_pbn()) + list(deals_from_bench(a.random)):
        try:
            hands = pa.parse_deal(deal)
        except ValueError:
            continue
        if any(len(h.replace(".", "").replace("-", "")) != 13 for h in hands.values()):
            continue  # donne incomplète (testdata/bad.pbn) : le moteur Go la refuse
        case = {"name": name, "dealer": dealer, "vulnerable": vul, "deal": deal}
        for lang in ("FR", "EN"):
            bids = pa.generate_auction(rules, hands, dealer, (), lang, vulnerable=vul)
            case["bids"] = [{"seat": b["seat"], "call": b["call"], "rule": b["rule"]} for b in bids]
            case["meaning_" + lang.lower()] = [b["meaning"] for b in bids]
        out.append(case)
    with open(OUT, "w", encoding="utf-8") as f:
        f.write('{"source": "tools/python_tools/gen_golden.py", "count": %d, "cases": [\n' % len(out))
        f.write(",\n".join(json.dumps(c, ensure_ascii=False, separators=(",", ":")) for c in out))
        f.write("\n]}\n")
    print("écrit %s : %d donnes" % (OUT, len(out)))


if __name__ == "__main__":
    main()
