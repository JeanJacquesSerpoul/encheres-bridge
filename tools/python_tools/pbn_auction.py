#!/usr/bin/env python3
"""Lit une donne d'un fichier PBN et sort sa séquence d'enchères en JSON.

Si la donne contient une section [Auction], elle est lue. Sinon (ou avec
--generate), les enchères sont générées à partir des quatre mains [Deal]
avec les règles SEF 2024 (cli/rules/default.yaml).

Usage : python pbn_auction.py fichier.pbn [index] [-o sortie.json]
                              [--generate] [--rules ../../cli/rules/default.yaml] [--option checkback2018]
                              [--lang FR|EN]   (FR=Français, défaut ; EN=Anglais)
                              [--opponents-pass]   (la paire qui n'a pas ouvert passe toujours)
  index : numéro de la donne (défaut 1)
"""
import argparse
import json
import os
import re
import sys

SEATS = ["N", "E", "S", "W"]
CALL_RE = re.compile(r"^(?:[1-7](?:C|D|H|S|NT|N)|Pass|P|X|XX|AP)$", re.IGNORECASE)
TAG_RE = re.compile(r'^\[(\w+)\s+"(.*)"\]\s*$')


def split_games(text):
    """Découpe le texte PBN en donnes (blocs séparés par des lignes vides)."""
    games, current = [], []
    for line in text.splitlines():
        line = line.lstrip("﻿").rstrip()
        if line.startswith("%"):  # commentaire / directive d'escape
            continue
        if not line.strip():
            if current:
                games.append(current)
                current = []
            continue
        current.append(line)
    if current:
        games.append(current)
    return games


def parse_game(lines):
    """Retourne (tags, lignes d'enchères brutes) pour une donne."""
    tags, auction_lines, in_auction = {}, [], False
    tags["_notes"] = {}
    for line in lines:
        m = TAG_RE.match(line.strip())
        if m:
            in_auction = m.group(1) == "Auction"
            if m.group(1) == "Note":  # [Note "1:texte de l'alerte"]
                num, _, text = m.group(2).partition(":")
                tags["_notes"][num.strip()] = text.strip()
            tags[m.group(1)] = m.group(2)
        elif line.startswith("["):
            in_auction = False
        elif in_auction:
            auction_lines.append(line)
    return tags, auction_lines


def normalize(call):
    c = call.upper()
    if c in ("P", "PASS"):
        return "Pass"
    if c in ("X", "XX", "AP"):
        return c
    return c.replace("NT", "N").replace("N", "NT")


def parse_calls(auction_lines, notes):
    """Retourne une liste de (annonce, alerte) ; alerte = None, True ou texte."""
    text = "\n".join(auction_lines)
    text = re.sub(r"\{[^}]*\}", " ", text)   # commentaires {...}
    text = re.sub(r";[^\n]*", " ", text)      # commentaires ; ...
    calls = []  # [annonce, alerte]
    for tok in text.split():
        if tok == "-":       # fin d'enchères non jouées
            break
        note = re.fullmatch(r"=(\d+)=", tok)
        if note and calls:   # renvoi vers un [Note "n:texte"]
            calls[-1][1] = notes.get(note.group(1), True)
            continue
        if tok.startswith(("$", "=", "?")) or tok == "+":
            continue         # autres annotations
        alert = tok.endswith("!")
        tok = tok.rstrip("!?")
        if not tok:          # "!" isolé : alerte sur l'annonce précédente
            if alert and calls and calls[-1][1] is None:
                calls[-1][1] = True
            continue
        if CALL_RE.match(tok):
            calls.append([normalize(tok), True if alert else None])
    # AP = All Pass : complète avec des Pass
    if calls and calls[-1][0] == "AP":
        calls.pop()
        if any(c != "Pass" for c, _ in calls):
            trailing = 0
            for c, _ in reversed(calls):
                if c != "Pass":
                    break
                trailing += 1
            n = max(0, 3 - trailing)
        else:
            n = 4 - len(calls)
        calls.extend([["Pass", None] for _ in range(n)])
    return [(c, a) for c, a in calls]


def build_result(tags, auction_lines, index):
    dealer = tags.get("Auction", tags.get("Dealer", "N")).upper()
    if dealer not in SEATS:
        dealer = "N"
    parsed = parse_calls(auction_lines, tags.get("_notes", {}))
    calls = [c for c, _ in parsed]
    start = SEATS.index(dealer)
    bids = [
        {"seat": SEATS[(start + i) % 4], "call": c,
         "alerted": a is not None, "alert": a if isinstance(a, str) else None}
        for i, (c, a) in enumerate(parsed)
    ]
    return {
        "index": index,
        "board": tags.get("Board"),
        "dealer": dealer,
        "vulnerable": tags.get("Vulnerable"),
        "sequence": calls,
        "bids": bids,
    }


LANGS = ("FR", "EN")
MSG = {
    "FR": {
        "desc": "Lit une donne d'un fichier PBN et sort sa séquence d'enchères en JSON.",
        "file": "fichier PBN",
        "index": "numéro de la donne (défaut 1)",
        "output": "fichier JSON de sortie (défaut : stdout)",
        "generate": "génère les enchères depuis [Deal], même si [Auction] existe",
        "rules": "fichier de règles (défaut : cli/rules/default.yaml)",
        "option": "option du système, ex. checkback2018 (répétable)",
        "lang": "langue des textes : FR=Français (défaut), EN=Anglais",
        "opps_pass": "les adversaires (la paire qui n'a pas ouvert) passent toujours",
        "opp_pass": "passe (option : les adversaires passent toujours)",
        "no_deal": "pas de section [Deal] : impossible de générer les enchères",
        "bad_deal": "[Deal] invalide : {v!r}",
        "no_rule": "passe par défaut (aucune règle pour cette séquence)",
        "illegal": "passe par défaut (la règle {id} donnerait {call}, illégal)",
        "not_found": "Donne {i} introuvable ({n} donne(s) dans le fichier).",
        "bad_rules": "Règles invalides : {e}",
        "game": "Donne {i} : {e}",
    },
    "EN": {
        "desc": "Reads a deal from a PBN file and outputs its auction sequence as JSON.",
        "file": "PBN file",
        "index": "deal number (default 1)",
        "output": "output JSON file (default: stdout)",
        "generate": "generate the auction from [Deal], even if [Auction] exists",
        "rules": "rules file (default: cli/rules/default.yaml)",
        "option": "system option, e.g. checkback2018 (repeatable)",
        "lang": "text language: FR=French (default), EN=English",
        "opps_pass": "opponents (the pair that did not open) always pass",
        "opp_pass": "pass (option: opponents always pass)",
        "no_deal": "no [Deal] section: cannot generate the auction",
        "bad_deal": "invalid [Deal]: {v!r}",
        "no_rule": "default pass (no rule for this sequence)",
        "illegal": "default pass (rule {id} would give {call}, which is illegal)",
        "not_found": "Deal {i} not found ({n} deal(s) in the file).",
        "bad_rules": "Invalid rules: {e}",
        "game": "Deal {i}: {e}",
    },
}


SUIT_RANK = {"C": 0, "D": 1, "H": 2, "S": 3, "NT": 4}


def parse_deal(value, lang="FR"):
    """'N:AK.. Q.. .. ..' -> {'N': main, 'E': main, ...} ; main = 'S.H.D.C'."""
    first, _, rest = value.partition(":")
    first = first.strip().upper()
    hands = rest.split()
    if first not in SEATS or len(hands) != 4:
        raise ValueError(MSG[lang]["bad_deal"].format(v=value))
    start = SEATS.index(first)
    return {SEATS[(start + i) % 4]: h for i, h in enumerate(hands)}


def _legal(call, history, seat):
    """history : liste de (index de siège 0-3, annonce PBN). `seat` : siège qui parle (0-3).
    Vérifie la légalité de `call`."""
    if call == "P":
        return True
    last = next(((i, c) for i, c in reversed(history) if c != "P"), None)
    if call in ("X", "XX"):
        if last is None:
            return False
        who, c = last
        if who % 2 == seat % 2:      # dernière annonce de sa propre paire
            return False
        return c[0].isdigit() if call == "X" else c == "X"
    bids = [c for _, c in history if c[0].isdigit()]
    if not bids:
        return True
    lv, su = bids[-1][0], bids[-1][1:]
    return (int(call[0]), SUIT_RANK[call[1:]]) > (int(lv), SUIT_RANK[su])


def parse_vulnerable(value):
    """Tag [Vulnerable] -> (N/S vulnérable, E/O vulnérable)."""
    v = (value or "").strip().upper()
    return (v in ("NS", "ALL", "BOTH"), v in ("EW", "ALL", "BOTH"))


def generate_auction(rules, hands, dealer, options=(), lang="FR", opps_pass=False, vulnerable=None):
    """Enchères des quatre mains selon les règles. Retourne une liste de dicts.
    vulnerable : valeur du tag [Vulnerable] (None, NS, EW, All...), lue par les conditions vul / opp_vul.
    Le rang du joueur à partir du donneur (1 à 4) est lu par la condition seat."""
    import sef_rules as sr
    vuln = parse_vulnerable(vulnerable)
    start = SEATS.index(dealer)
    parsed = {seat: sr.parse_hand(h) for seat, h in hands.items()}
    history, out = [], []            # history : (siège 0-3, annonce)
    trump = {0: None, 1: None}       # atout convenu par paire (NS=0, EW=1)
    for i in range(60):
        seat = (start + i) % 4
        pair = seat % 2
        seq = []
        for who, c in history:       # séquence vue par la paire qui parle
            if who % 2 == pair:
                seq.append(c)
            elif c != "P":
                seq.append(f"({c})")
        opener = next((w % 2 for w, c in history if c != "P"), None)  # paire qui a ouvert
        forced = opps_pass and opener is not None and pair != opener
        r = None if forced else sr.choose(rules, " ".join(seq), parsed[SEATS[seat]], options, trump[pair],
                                          vuln[pair], vuln[1 - pair], (seat - start) % 4 + 1)
        note = None
        if forced:
            note = MSG[lang]["opp_pass"]
        elif r is None:
            note = MSG[lang]["no_rule"]
        elif not _legal(r["call"], history, seat):
            note, r = MSG[lang]["illegal"].format(id=r["id"], call=r["call"]), None
        call = r["call"] if r else "P"
        if r and r.get("trump"):
            trump[pair] = r["trump"]
        history.append((seat, call))
        alerted = bool(r and r.get("alert"))
        text = (r["meaning_en"] if lang == "EN" else r["meaning"]) if r else note
        out.append({
            "seat": SEATS[seat], "call": normalize(call),
            "alerted": alerted, "alert": text if alerted else None,
            "rule": r["id"] if r else ("opponents.pass" if forced else None), "meaning": text,
        })
        calls = [c for _, c in history]
        if (len(calls) >= 4 and all(c == "P" for c in calls)) or            (any(c != "P" for c in calls) and calls[-3:] == ["P"] * 3):
            break
    return out


def build_generated(tags, index, rules, options, lang="FR", opps_pass=False):
    if "Deal" not in tags:
        raise ValueError(MSG[lang]["no_deal"])
    dealer = tags.get("Dealer", "N").upper()
    if dealer not in SEATS:
        dealer = "N"
    hands = parse_deal(tags["Deal"], lang)
    bids = generate_auction(rules, hands, dealer, tuple(options), lang, opps_pass, tags.get("Vulnerable"))
    return {
        "index": index,
        "board": tags.get("Board"),
        "dealer": dealer,
        "vulnerable": tags.get("Vulnerable"),
        "lang": lang,
        "opponents_pass": opps_pass,
        "source": "generated",
        "hands": hands,
        "default_calls": sum(1 for b in bids if b["rule"] is None),
        "sequence": [b["call"] for b in bids],
        "bids": bids,
    }


def main():
    pre = argparse.ArgumentParser(add_help=False)
    pre.add_argument("--lang", default="FR", type=str.upper, choices=LANGS)
    lang = pre.parse_known_args()[0].lang
    m = MSG[lang]
    ap = argparse.ArgumentParser(description=m["desc"])
    ap.add_argument("file", help=m["file"])
    ap.add_argument("index", nargs="?", type=int, default=1, help=m["index"])
    ap.add_argument("-o", "--output", help=m["output"])
    ap.add_argument("--generate", action="store_true", help=m["generate"])
    ap.add_argument("--rules", default=os.path.normpath(os.path.join(
        os.path.dirname(os.path.abspath(__file__)), "..", "..", "cli", "rules", "default.yaml")), help=m["rules"])
    ap.add_argument("--option", action="append", default=[], help=m["option"])
    ap.add_argument("--opponents-pass", action="store_true", help=m["opps_pass"])
    ap.add_argument("--lang", default="FR", type=str.upper, choices=LANGS, help=m["lang"])
    args = ap.parse_args()

    with open(args.file, encoding="utf-8", errors="replace") as f:
        games = split_games(f.read())
    if not 1 <= args.index <= len(games):
        sys.exit(m["not_found"].format(i=args.index, n=len(games)))

    tags, auction_lines = parse_game(games[args.index - 1])
    if "Auction" in tags and not args.generate:
        result = build_result(tags, auction_lines, args.index)
    else:
        sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
        import sef_rules as sr
        rules = sr.load(args.rules)
        errs = sr.validate(rules)
        if errs:
            sys.exit(m["bad_rules"].format(e="; ".join(errs[:3])))
        try:
            result = build_generated(tags, args.index, rules, args.option, lang, args.opponents_pass)
        except ValueError as e:
            sys.exit(m["game"].format(i=args.index, e=e))
    out = json.dumps(result, ensure_ascii=False, indent=2)
    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(out + "\n")
    else:
        sys.stdout.buffer.write((out + "\n").encode("utf-8"))


if __name__ == "__main__":
    main()
