"""Tests de pbn_auction.py.  Lancer : python -m unittest test_pbn_auction -v"""
import json
import os
import subprocess
import sys
import tempfile
import unittest

import pbn_auction as pa
import sef_rules as sr

HERE = os.path.dirname(os.path.abspath(__file__))
RULES = sr.load(sr.DEFAULT_RULES)
assert not sr.validate(RULES)   # compile aussi les conditions des règles

# Donne de donne.pbn : Sud ouvre 2♦ (19 H, 6-4-0-3)
DEAL = "W:.J5.Q8532.J76532 J84.T74.KJ97.Q84 T762.K862.AT64.K AKQ953.AQ93..AT9"

PBN_TWO_DEALS = """% PBN 2.1
[Board "1"]
[Dealer "N"]
[Vulnerable "None"]
[Auction "N"]
1C =1= 1S X!
2C =2= Pass Pass AP
[Note "1:Forcing, 16+ HCP"]
[Note "2:Stayman"]

[Board "2"]
[Dealer "E"]
[Auction "E"]
1S {commentaire} X 2S ; autre commentaire
Pass Pass Pass
"""


def parse(text, index=1):
    return pa.parse_game(pa.split_games(text)[index - 1])


def generate(deal=DEAL, dealer="W", **kw):
    hands = pa.parse_deal(deal)
    return pa.generate_auction(RULES, hands, dealer, **kw)


class TestLecture(unittest.TestCase):
    def test_split_games(self):
        self.assertEqual(len(pa.split_games(PBN_TWO_DEALS)), 2)

    def test_split_ignore_commentaires_pourcent_et_bom(self):
        games = pa.split_games("﻿% commentaire\n[Board \"1\"]\n\n\n[Board \"2\"]\n")
        self.assertEqual(len(games), 2)
        self.assertNotIn("% commentaire", games[0])

    def test_donne_par_defaut_est_la_premiere(self):
        tags, lines = parse(PBN_TWO_DEALS)
        res = pa.build_result(tags, lines, 1)
        self.assertEqual(res["board"], "1")
        self.assertEqual(res["dealer"], "N")

    def test_choix_de_la_donne_2(self):
        tags, lines = parse(PBN_TWO_DEALS, 2)
        res = pa.build_result(tags, lines, 2)
        self.assertEqual(res["board"], "2")
        self.assertEqual(res["dealer"], "E")
        self.assertEqual(res["sequence"], ["1S", "X", "2S", "Pass", "Pass", "Pass"])

    def test_sequence_et_sieges(self):
        tags, lines = parse(PBN_TWO_DEALS)
        res = pa.build_result(tags, lines, 1)
        self.assertEqual(res["sequence"][:4], ["1C", "1S", "X", "2C"])
        self.assertEqual([b["seat"] for b in res["bids"][:4]], ["N", "E", "S", "W"])

    def test_ap_complete_trois_passes_finaux(self):
        tags, lines = parse(PBN_TWO_DEALS)
        res = pa.build_result(tags, lines, 1)
        self.assertEqual(res["sequence"][-3:], ["Pass", "Pass", "Pass"])
        self.assertEqual(len(res["sequence"]), 7)  # 1C 1S X 2C P P P

    def test_ap_sans_enchere_donne_quatre_passes(self):
        calls = pa.parse_calls(["AP"], {})
        self.assertEqual([c for c, _ in calls], ["Pass"] * 4)

    def test_alertes_via_notes(self):
        tags, lines = parse(PBN_TWO_DEALS)
        bids = pa.build_result(tags, lines, 1)["bids"]
        self.assertTrue(bids[0]["alerted"])
        self.assertEqual(bids[0]["alert"], "Forcing, 16+ HCP")
        self.assertEqual(bids[3]["alert"], "Stayman")
        self.assertFalse(bids[1]["alerted"])
        self.assertIsNone(bids[1]["alert"])

    def test_alerte_marque_point_exclamation_sans_texte(self):
        tags, lines = parse(PBN_TWO_DEALS)
        x = pa.build_result(tags, lines, 1)["bids"][2]
        self.assertEqual(x["call"], "X")
        self.assertTrue(x["alerted"])
        self.assertIsNone(x["alert"])

    def test_note_inconnue_alerte_sans_texte(self):
        calls = pa.parse_calls(["1C =9="], {})
        self.assertEqual(calls, [("1C", True)])

    def test_point_exclamation_isole(self):
        calls = pa.parse_calls(["1C !"], {})
        self.assertEqual(calls, [("1C", True)])

    def test_commentaires_et_annotations_ignores(self):
        calls = pa.parse_calls(["1S {un commentaire} $1 X ; fin", "Pass ? 2S"], {})
        self.assertEqual([c for c, _ in calls], ["1S", "X", "Pass", "2S"])

    def test_normalisation_des_annonces(self):
        calls = pa.parse_calls(["1n p 2nt pass xx"], {})
        self.assertEqual([c for c, _ in calls], ["1NT", "Pass", "2NT", "Pass", "XX"])


class TestDeal(unittest.TestCase):
    def test_parse_deal_place_les_mains(self):
        hands = pa.parse_deal(DEAL)
        self.assertEqual(hands["W"], ".J5.Q8532.J76532")
        self.assertEqual(hands["N"], "J84.T74.KJ97.Q84")
        self.assertEqual(hands["E"], "T762.K862.AT64.K")
        self.assertEqual(hands["S"], "AKQ953.AQ93..AT9")

    def test_parse_deal_autre_premier_joueur(self):
        hands = pa.parse_deal("N:a.b.c.d e.f.g.h i.j.k.l m.n.o.p")
        self.assertEqual((hands["N"], hands["E"], hands["S"], hands["W"]),
                         ("a.b.c.d", "e.f.g.h", "i.j.k.l", "m.n.o.p"))

    def test_parse_deal_invalide(self):
        with self.assertRaises(ValueError):
            pa.parse_deal("W:une seule main")
        with self.assertRaises(ValueError):
            pa.parse_deal("X:a b c d")


class TestLegalite(unittest.TestCase):
    def test_passe_toujours_legal(self):
        self.assertTrue(pa._legal("P", [], 0))

    def test_enchere_superieure(self):
        h = [(0, "1H")]
        self.assertTrue(pa._legal("1S", h, 1))
        self.assertTrue(pa._legal("2C", h, 1))
        self.assertFalse(pa._legal("1D", h, 1))
        self.assertFalse(pa._legal("1H", h, 1))

    def test_contre_seulement_sur_enchere_adverse(self):
        self.assertFalse(pa._legal("X", [], 0))
        self.assertTrue(pa._legal("X", [(0, "1H")], 1))          # E contre N
        self.assertFalse(pa._legal("X", [(0, "1H"), (1, "P")], 2))  # S ne contre pas son partenaire
        self.assertFalse(pa._legal("X", [(0, "1H"), (1, "X")], 2))  # pas de contre sur un contre

    def test_contre_avec_donneur_est_ou_ouest(self):
        # W (3) ouvre, N (0) contre : la parité du siège doit compter, pas la longueur de l'historique
        self.assertTrue(pa._legal("X", [(3, "1H")], 0))
        self.assertFalse(pa._legal("X", [(3, "1H"), (0, "P")], 1))  # E ne contre pas son partenaire W
        self.assertTrue(pa._legal("X", [(1, "1H")], 2))              # E ouvre, S contre

    def test_surcontre(self):
        self.assertTrue(pa._legal("XX", [(0, "1H"), (1, "X")], 2))
        self.assertFalse(pa._legal("XX", [(0, "1H")], 1))
        self.assertTrue(pa._legal("XX", [(3, "1H"), (0, "X")], 1))
        self.assertFalse(pa._legal("XX", [(3, "1H"), (0, "X")], 2))  # W/E : son propre camp n'est pas visé


class TestGeneration(unittest.TestCase):
    def test_donne_pbn_sud_ouvre_2k(self):
        bids = generate()
        self.assertEqual([b["call"] for b in bids[:4]], ["Pass", "Pass", "Pass", "2D"])
        self.assertEqual(bids[3]["seat"], "S")
        self.assertEqual(bids[3]["rule"], "open.2D")
        self.assertTrue(bids[3]["alerted"])

    def test_les_quatre_joueurs_enchérissent_et_l_enchere_finit(self):
        bids = generate()
        calls = [b["call"] for b in bids]
        self.assertEqual(calls[-3:], ["Pass", "Pass", "Pass"])
        self.assertEqual([b["seat"] for b in bids[:4]], ["W", "N", "E", "S"])

    def test_quatre_passes_si_personne_n_ouvre(self):
        hands = {s: "5432.432.432.432" for s in "NESW"}
        bids = pa.generate_auction(RULES, hands, "N")
        self.assertEqual([b["call"] for b in bids], ["Pass"] * 4)

    def test_sans_option_les_adversaires_passent_par_defaut(self):
        bids = generate()
        self.assertGreater(sum(b["rule"] is None for b in bids), 0)

    def test_option_adversaires_passent(self):
        bids = generate(opps_pass=True)
        self.assertEqual(sum(b["rule"] is None for b in bids), 0)
        opp = [b for b in bids[4:] if b["seat"] in ("E", "W")]
        self.assertTrue(opp)
        for b in opp:
            self.assertEqual((b["call"], b["rule"]), ("Pass", "opponents.pass"))

    def test_option_adversaires_ne_change_pas_l_avant_ouverture(self):
        with_opt, without = generate(opps_pass=True), generate()
        self.assertEqual(with_opt[:4], without[:4])

    def test_langue(self):
        fr, en = generate(lang="FR"), generate(lang="EN")
        self.assertEqual(fr[3]["meaning"][:4], "Fort")
        self.assertTrue(en[3]["meaning"].startswith("Strong"))
        self.assertEqual(en[3]["alert"], en[3]["meaning"])
        self.assertIn("aucune règle", fr[4]["meaning"])
        self.assertIn("no rule", en[4]["meaning"])

    def test_annonces_toujours_legales(self):
        import random
        rng = random.Random(7)
        for _ in range(60):
            hands = {s: sr.fmt(h) for s, h in zip("NSEW", sr.deal(rng))}
            bids = pa.generate_auction(RULES, hands, rng.choice("NESW"))
            history = []
            for i, b in enumerate(bids):
                seat = pa.SEATS.index(b["seat"])
                call = "P" if b["call"] == "Pass" else b["call"]
                self.assertTrue(pa._legal(call, history, seat), (bids, i))
                history.append((seat, call))

    def test_build_generated_sans_deal(self):
        with self.assertRaises(ValueError):
            pa.build_generated({"Dealer": "N"}, 1, RULES, [])

    def test_build_generated_structure(self):
        tags = {"Dealer": "W", "Vulnerable": "None", "Deal": DEAL, "Board": "5"}
        res = pa.build_generated(tags, 1, RULES, [], "EN", True)
        for k in ("index", "board", "dealer", "vulnerable", "lang", "opponents_pass",
                  "source", "hands", "default_calls", "sequence", "bids"):
            self.assertIn(k, res)
        self.assertEqual((res["source"], res["lang"], res["board"]), ("generated", "EN", "5"))
        self.assertEqual(res["default_calls"], 0)
        self.assertEqual(res["sequence"], [b["call"] for b in res["bids"]])


class TestCLI(unittest.TestCase):
    def run_cli(self, *args):
        env = dict(os.environ, PYTHONUTF8="1")
        return subprocess.run([sys.executable, os.path.join(HERE, "pbn_auction.py"), *args],
                              capture_output=True, encoding="utf-8", env=env)

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.donne = os.path.join(self.tmp.name, "donne.pbn")
        with open(self.donne, "w", encoding="utf-8") as f:
            f.write('[Dealer "W"]\n[Vulnerable "None"]\n[Deal "%s"]\n' % DEAL)
        self.avec_auction = os.path.join(self.tmp.name, "auction.pbn")
        with open(self.avec_auction, "w", encoding="utf-8") as f:
            f.write(PBN_TWO_DEALS)

    def test_generation_par_defaut_sans_auction(self):
        p = self.run_cli(self.donne)
        self.assertEqual(p.returncode, 0, p.stderr)
        res = json.loads(p.stdout)
        self.assertEqual(res["source"], "generated")
        self.assertEqual(res["lang"], "FR")
        self.assertEqual(res["sequence"][3], "2D")

    def test_lecture_de_l_auction_existante(self):
        p = self.run_cli(self.avec_auction, "2")
        res = json.loads(p.stdout)
        self.assertEqual(res["board"], "2")
        self.assertNotIn("source", res)

    def test_generate_force_la_generation_mais_exige_deal(self):
        p = self.run_cli(self.avec_auction, "--generate")
        self.assertNotEqual(p.returncode, 0)
        self.assertIn("[Deal]", p.stderr)

    def test_option_langue_et_adversaires(self):
        p = self.run_cli(self.donne, "--lang", "en", "--opponents-pass")
        res = json.loads(p.stdout)
        self.assertEqual(res["lang"], "EN")
        self.assertTrue(res["opponents_pass"])
        self.assertEqual(res["default_calls"], 0)

    def test_sortie_dans_un_fichier(self):
        out = os.path.join(self.tmp.name, "sortie.json")
        p = self.run_cli(self.donne, "-o", out)
        self.assertEqual(p.returncode, 0, p.stderr)
        with open(out, encoding="utf-8") as f:
            self.assertEqual(json.load(f)["dealer"], "W")

    def test_donne_introuvable(self):
        p = self.run_cli(self.donne, "5")
        self.assertNotEqual(p.returncode, 0)
        self.assertIn("introuvable", p.stderr)
        p = self.run_cli(self.donne, "5", "--lang", "EN")
        self.assertIn("not found", p.stderr)

    def test_langue_invalide(self):
        self.assertNotEqual(self.run_cli(self.donne, "--lang", "XX").returncode, 0)


if __name__ == "__main__":
    unittest.main()
