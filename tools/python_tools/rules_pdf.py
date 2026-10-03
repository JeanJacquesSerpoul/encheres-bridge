#!/usr/bin/env python3
"""Description lisible d'un système d'enchères : un fichier de règles YAML rendu en PDF.

Usage : python rules_pdf.py ../../cli/systems/sef/rules.yaml [-o rules.pdf] [--lang FR|EN] [--title "SEF 2024"]

Les règles sont présentées dans l'ordre du fichier, section par section : la séquence (vue par la
paire qui parle), l'enchère, sa condition traduite en clair, sa signification et son caractère forcing.
Le PDF est écrit sans dépendance : polices standard du format (Helvetica, et Symbol pour ♠ ♥ ♦ ♣ et
≤ ≥ ≠), texte compressé. Il porte l'empreinte SHA-256 du fichier source (métadonnée Subject) : le test
Go TestRulesPDFUpToDate signale un PDF qui n'a pas été régénéré après une modification des règles.
"""
import argparse
import ast
import hashlib
import io
import os
import re
import sys
import unicodedata
import zlib

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import sef_rules as sr  # noqa: E402

SUIT = {"S": "♠", "H": "♥", "D": "♦", "C": "♣"}

TXT = {
    "FR": dict(
        opening="Ouverture", pas="Passe", nt="SA", any="?", prefix="…", trump="atout", notrump="SA",
        cond="Condition", none="aucune", meaning="Sens", forcing="Forcing", status="Statut", option="option",
        page="page", source="Fichier source", rules="règles (modèles développés)", sha="empreinte SHA-256",
        title="Système d'enchères", toc="Sections",
        intro=[
            "Ce document décrit, règle par règle, le système que le moteur d'enchères applique. Il est produit "
            "automatiquement à partir du fichier de règles : il en suit l'ordre et les sections.",
            "Pour chaque enchère, le moteur prend la PREMIÈRE règle applicable, dans l'ordre du document : sa "
            "séquence correspond à l'enchère en cours et sa condition est vraie sur la main. Si aucune ne "
            "s'applique, ou si l'enchère de la règle serait illégale, le joueur passe.",
            "La séquence est vue par la paire qui parle : ses propres enchères, passes comprises, et celles des "
            "adversaires entre parenthèses, leurs passes omises. « Ouverture » : personne n'a encore parlé dans la "
            "paire. « … » en tête : n'importe quel début. « ? » : n'importe quelle enchère. « [atout ♠] » : "
            "l'atout convenu par la paire est ♠. Les passes initiales de la paire sont ignorées, sauf quand la "
            "séquence commence elle-même par Passe.",
            "Points : H = points d'honneur (A 4, R 3, D 2, V 1) ; HL = H + un point par carte à partir de la 5e "
            "dans une couleur commandée par au moins D V, moins un point par honneur sec ou paire d'honneurs secs ; "
            "DH = H + points de courte (chicane 3, singleton 2, doubleton 1), même dévaluation ; HLD = HL + "
            "points de courte, + 4 pour un bicolore 6-5 ou plus. ♠ ♥ ♦ ♣ seuls désignent la longueur de la couleur. Position : rang du joueur dans le "
            "tour d'enchères (1 = donneur, 4 = 4e position).",
            "Forcing : NF non forcing, F1 forcing un tour, FM forcing de manche, INV invitation, SO conclusion, "
            "REL relais, ASK question, TO contre d'appel, PEN punitif. Statut : SEF (fiches SEF 2024), SEF 2018, "
            "choix (option retenue), inféré (complétion naturelle, absente des fiches), à vérifier.",
        ],
        names=dict(hcp="H", hl="HL", dh="DH", hld="HLD", shape="forme", balanced="main régulière",
                   semibalanced="main semi-régulière", aces="nombre d'As", kings="nombre de Rois", losers="perdantes",
                   ptricks="levées de jeu", qtricks="levées rapides", sidetricks="levées rapides annexes",
                   vul="vulnérable", opp_vul="adversaires vulnérables", seat="position"),
        neg=dict(balanced="main irrégulière", semibalanced="main non semi-régulière", vul="non vulnérable",
                 opp_vul="adversaires non vulnérables"),
        funcs=dict(ace="As {s}", king="Roi {s}", queen="Dame {s}", top="gros honneurs {s}", solid="{s} ARD",
                   stop="arrêt {s}", short="courte {s}", hcp_in="H à {s}", keycards="clés (atout {s})",
                   ctrl1="contrôle du 1er tour {s}", ctrl2="contrôle du 2e tour {s}"),
        negf=dict(ace="sans As {s}", king="sans Roi {s}", queen="sans Dame {s}", stop="sans arrêt {s}",
                  short="pas de courte {s}", solid="{s} pas ARD", ctrl1="sans contrôle du 1er tour {s}",
                  ctrl2="sans contrôle du 2e tour {s}"),
        and_=" et ", or_=" ou ", not_="non ", true="vrai", false="faux",
        forcing_names=dict(NF="non forcing", F1="forcing un tour", FM="forcing de manche", SO="conclusion",
                           INV="invitation", REL="relais", ASK="question", TO="contre d'appel", PEN="punitif"),
        status_names=dict(sef="SEF", choix="choix", sef2018="SEF 2018", infere="inféré", a_verifier="à vérifier"),
    ),
    "EN": dict(
        opening="Opening", pas="Pass", nt="NT", any="?", prefix="…", trump="trumps", notrump="NT",
        cond="Condition", none="none", meaning="Meaning", forcing="Forcing", status="Status", option="option",
        page="page", source="Source file", rules="rules (templates expanded)", sha="SHA-256 fingerprint",
        title="Bidding system", toc="Sections",
        intro=[
            "This document describes, rule by rule, the system the bidding engine applies. It is generated from "
            "the rules file and follows its order and sections.",
            "For every call the engine takes the FIRST applicable rule, in document order: its sequence matches "
            "the auction so far and its condition holds on the hand. If none applies, or if the rule's call would "
            "be illegal, the player passes.",
            "The sequence is seen by the bidding pair: its own calls, passes included, and the opponents' calls in "
            "parentheses, their passes left out. \"Opening\": nobody in the pair has spoken yet. A leading \"…\": "
            "any beginning. \"?\": any call. \"[trumps ♠]\": the pair's agreed suit is ♠. The pair's initial passes "
            "are ignored, unless the sequence itself starts with Pass.",
            "Points: HCP = high-card points (A 4, K 3, Q 2, J 1); HL = HCP + one point per card from the fifth "
            "in a suit headed by at least Q-J, less one point per bare honour or bare pair of honours; "
            "DH = HCP + shortness points (void 3, singleton 2, doubleton 1), same deduction; HLD = HL + shortness "
            "points, + 4 for a 6-5 two-suiter or longer. ♠ ♥ ♦ ♣ alone mean the length of the suit. Seat: the player's rank in the auction (1 = dealer, "
            "4 = fourth seat).",
            "Forcing: NF non-forcing, F1 forcing one round, FM game forcing, INV invitational, SO sign-off, "
            "REL relay, ASK asking bid, TO takeout double, PEN penalty. Status: SEF (SEF 2024 cards), SEF 2018, "
            "choix (chosen option), inferred (natural completion, not on the cards), to be checked.",
        ],
        names=dict(hcp="HCP", hl="HL", dh="DH", hld="HLD", shape="shape", balanced="balanced",
                   semibalanced="semi-balanced", aces="aces", kings="kings", losers="losers",
                   ptricks="playing tricks", qtricks="quick tricks", sidetricks="side quick tricks",
                   vul="vulnerable", opp_vul="opponents vulnerable", seat="seat"),
        neg=dict(balanced="unbalanced", semibalanced="not semi-balanced", vul="not vulnerable",
                 opp_vul="opponents not vulnerable"),
        funcs=dict(ace="{s} ace", king="{s} king", queen="{s} queen", top="{s} top honours", solid="{s} AKQ",
                   stop="{s} stopper", short="{s} shortness", hcp_in="{s} HCP", keycards="keycards ({s} trumps)",
                   ctrl1="{s} first-round control", ctrl2="{s} second-round control"),
        negf=dict(ace="no {s} ace", king="no {s} king", queen="no {s} queen", stop="no {s} stopper",
                  short="no {s} shortness", solid="{s} not AKQ", ctrl1="no {s} first-round control",
                  ctrl2="no {s} second-round control"),
        and_=" and ", or_=" or ", not_="not ", true="true", false="false",
        forcing_names=dict(NF="non-forcing", F1="forcing one round", FM="game forcing", SO="sign-off",
                           INV="invitational", REL="relay", ASK="asking bid", TO="takeout", PEN="penalty"),
        status_names=dict(sef="SEF", choix="choix", sef2018="SEF 2018", infere="inferred", a_verifier="to be checked"),
    ),
}


# ---------------------------------------------------------------- enchères et séquences
def call_text(call, t):
    if call == "P":
        return t["pas"]
    if call in ("X", "XX"):
        return call
    st = call[1:]
    return call[0] + (t["nt"] if st == "NT" else SUIT[st])


def seq_text(pattern, t):
    toks = pattern.split()
    if not toks:
        return t["opening"]
    out = []
    for i, tok in enumerate(toks):
        if tok == "**":
            out.append(t["prefix"])
        elif tok == "*":
            out.append(t["any"])
        elif tok.startswith("BW:"):
            out.append("[%s %s] %s" % (t["trump"], SUIT.get(tok[3:], tok[3:]), t["prefix"]))
        else:
            paren = tok.startswith("(")
            alts = "/".join(call_text(a, t) for a in tok.strip("()").split("|"))
            out.append("(%s)" % alts if paren else alts)
    return " ".join(out)


# ---------------------------------------------------------------- conditions en clair
OPS = {ast.Lt: "<", ast.LtE: "≤", ast.Gt: ">", ast.GtE: "≥", ast.Eq: "=", ast.NotEq: "≠", ast.In: "∈", ast.NotIn: "∉"}


def constant(node):
    """La valeur d'un sous-arbre sans nom, ou None s'il en dépend."""
    if any(isinstance(n, ast.Name) and n.id not in ("True", "False") for n in ast.walk(node)):
        return None
    try:
        return eval(compile(ast.Expression(node), "<c>", "eval"), {"__builtins__": {}}, {})
    except Exception:
        return None


def simplify(node):
    """Retire les parties constantes (issues des modèles développés) d'une condition."""
    if isinstance(node, ast.BoolOp):
        vals = [simplify(v) for v in node.values]
        keep = []
        for v in vals:
            c = constant(v)
            if c is None:
                keep.append(v)
            elif isinstance(node.op, ast.And) and not c:
                return ast.Constant(False)
            elif isinstance(node.op, ast.Or) and c:
                return ast.Constant(True)
        if not keep:
            return ast.Constant(isinstance(node.op, ast.And))
        return keep[0] if len(keep) == 1 else ast.BoolOp(op=node.op, values=keep)
    if isinstance(node, ast.UnaryOp) and isinstance(node.op, ast.Not):
        inner = simplify(node.operand)
        c = constant(inner)
        return ast.Constant(not c) if c is not None else ast.UnaryOp(op=node.op, operand=inner)
    c = constant(node)
    return ast.Constant(c) if c is not None and not isinstance(node, ast.Constant) else node


def expr_text(node, t, parent=None):
    if isinstance(node, ast.BoolOp):
        sep = t["and_"] if isinstance(node.op, ast.And) else t["or_"]
        s = sep.join(expr_text(v, t, node) for v in node.values)
        # un « ou » dans un « et » (ou sous une négation) se lit entre parenthèses
        if parent is not None and (isinstance(parent, ast.UnaryOp) or
                                   (isinstance(parent, ast.BoolOp) and type(parent.op) is not type(node.op))):
            s = "(%s)" % s
        return s
    if isinstance(node, ast.UnaryOp) and isinstance(node.op, ast.Not):
        o = node.operand
        if isinstance(o, ast.Name) and o.id in t["neg"]:
            return t["neg"][o.id]
        if isinstance(o, ast.Call) and isinstance(o.func, ast.Name) and o.func.id in t["negf"] and o.args:
            return t["negf"][o.func.id].format(s=expr_text(o.args[0], t))
        return t["not_"] + ("(%s)" % expr_text(o, t) if not isinstance(o, (ast.Name, ast.Call)) else expr_text(o, t))
    if isinstance(node, ast.UnaryOp) and isinstance(node.op, ast.USub):
        return "-" + expr_text(node.operand, t, node)
    if isinstance(node, ast.Compare):
        parts = [expr_text(node.left, t, node)]
        for op, right in zip(node.ops, node.comparators):
            parts += [OPS[type(op)], expr_text(right, t, node)]
        return " ".join(parts)
    if isinstance(node, ast.BinOp):
        return "%s %s %s" % (expr_text(node.left, t, node), "+" if isinstance(node.op, ast.Add) else "-",
                             expr_text(node.right, t, node))
    if isinstance(node, ast.Call):
        name = node.func.id
        args = [expr_text(a, t) for a in node.args]
        if name in t["funcs"] and args:
            return t["funcs"][name].format(s=args[0])
        return "%s(%s)" % (name, ", ".join(args))
    if isinstance(node, ast.Tuple):
        return "{%s}" % ", ".join(expr_text(e, t) for e in node.elts)
    if isinstance(node, ast.Name):
        if node.id in SUIT:
            return SUIT[node.id]
        return t["names"].get(node.id, node.id)
    if isinstance(node, ast.Constant):
        v = node.value
        if isinstance(v, bool):
            return t["true"] if v else t["false"]
        if isinstance(v, str):
            return SUIT.get(v, v)
        return repr(v)
    return ast.unparse(node)


def cond_text(cond, t):
    src = re.sub(r"\btrue\b", "True", re.sub(r"\bfalse\b", "False", str(cond).strip()))
    node = simplify(ast.parse(src, mode="eval").body)
    c = constant(node)
    if c is True:
        return t["none"]
    return expr_text(node, t)


# ---------------------------------------------------------------- structure du fichier
def outline(path, lang="FR"):
    """Les événements du fichier dans l'ordre : sections, sous-sections, commentaires, modèles de règles.

    Une ligne « # en: … » qui suit un titre (ou un commentaire) en donne la traduction anglaise : elle
    remplace le texte français dans le PDF anglais et n'apparaît jamais comme un commentaire."""
    lines = io.open(path, encoding="utf-8").read().replace("\r\n", "\n").split("\n")
    events, started, i = [], False, 0
    while i < len(lines):
        ln = lines[i]
        if ln.startswith("# ====") and i + 2 < len(lines) and lines[i + 2].startswith("# ===="):
            events.append(["section", lines[i + 1][2:].strip()])
            started = True
            i += 3
            continue
        if started and ln.startswith("# en: "):
            if events and events[-1][0] != "rule" and lang == "EN":
                events[-1][1] = ln[6:].strip()
        elif started and ln.startswith("# --- "):
            events.append(["sub", ln[6:].strip()])
        elif started and ln.startswith("# "):
            events.append(["note", ln[2:].strip()])
        elif started and ln.startswith("- id:"):
            events.append(["rule", ln[5:].strip().strip('"')])
        i += 1
    return [tuple(e) for e in events]


# ---------------------------------------------------------------- PDF
W, H = 595.28, 841.89          # A4
MARGIN = 42
# chasses Helvetica (1/1000 em), ASCII 32-126
HELV = [278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278] + [556] * 10 + \
    [278, 278, 584, 584, 584, 556, 1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722,
     778, 667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556, 333, 556, 556, 500,
     556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556, 556, 556, 333, 500, 278, 556, 500, 722, 500,
     500, 500, 334, 260, 334, 584]
EXTRA = {"«": 556, "»": 556, "’": 222, "‘": 222, "“": 333, "”": 333, "–": 556, "—": 1000, "…": 1000, "œ": 944,
         "Œ": 1000, "€": 556, "°": 400, "·": 278, " ": 278}
# police Symbol : code et chasse
SYMBOL = {"♣": (0xA7, 753), "♦": (0xA8, 753), "♥": (0xA9, 753), "♠": (0xAA, 753), "≤": (0xA3, 549),
          "≥": (0xB3, 549), "≠": (0xB9, 549), "∈": (0xCE, 713), "∉": (0xCF, 713), "→": (0xAE, 987)}
RED = {"♥", "♦"}


def char_width(ch):
    if ch in SYMBOL:
        return SYMBOL[ch][1]
    o = ord(ch)
    if 32 <= o < 127:
        return HELV[o - 32]
    if ch in EXTRA:
        return EXTRA[ch]
    base = unicodedata.normalize("NFD", ch)[0]
    return HELV[ord(base) - 32] if 32 <= ord(base) < 127 else 556


def text_width(s, size, bold=False):
    return sum(char_width(c) for c in s) * size / 1000 * (1.07 if bold else 1)


def pdf_string(s):
    b = s.encode("cp1252", "replace")
    return b"(" + b.replace(b"\\", b"\\\\").replace(b"(", b"\\(").replace(b")", b"\\)") + b")"


class Doc:
    def __init__(self, title, lang):
        self.title, self.t = title, TXT[lang]
        self.pages, self.ops, self.y = [], None, 0
        self.bookmarks = []            # (titre, page)

    def new_page(self):
        self.ops = []
        self.pages.append(self.ops)
        self.y = H - MARGIN - 10

    def ensure(self, h):
        if self.ops is None or self.y - h < MARGIN + 14:
            self.new_page()

    def run(self, x, y, s, size, font="F1", color=(0, 0, 0)):
        """Écrit s en une ligne, en basculant sur Symbol pour les signes qui n'existent pas en Helvetica."""
        segs, cur, cur_sym = [], "", None
        for ch in s:
            sym = ch in SYMBOL
            key = (sym, ch in RED)
            if cur and key != cur_sym:
                segs.append((cur_sym, cur))
                cur = ""
            cur_sym = key
            cur += ch
        if cur:
            segs.append((cur_sym, cur))
        for (sym, red), txt in segs:
            if sym:
                data = b"(" + bytes(SYMBOL[c][0] for c in txt) + b")"
                col = (0.78, 0, 0) if red else color
                self.ops.append(b"BT %.2f %.2f %.2f rg /F4 %.1f Tf %.2f %.2f Td %s Tj ET" % (col[0], col[1], col[2], size, x, y, data))
                x += sum(SYMBOL[c][1] for c in txt) * size / 1000
            else:
                self.ops.append(b"BT %.2f %.2f %.2f rg /%s %.1f Tf %.2f %.2f Td %s Tj ET" % (
                    color[0], color[1], color[2], font.encode(), size, x, y, pdf_string(txt)))
                x += text_width(txt, size, font == "F2")
        return x

    def paragraph(self, s, size=9, font="F1", indent=0, hang=0, color=(0, 0, 0), before=0, after=2):
        """Texte justifié à gauche, coupé aux espaces ; `hang` décale les lignes suivantes."""
        lead = size * 1.28
        words = s.split(" ")
        lines, cur = [], ""
        width = W - 2 * MARGIN - indent
        for w in words:
            trial = (cur + " " + w) if cur else w
            avail = width - (hang if lines else 0)
            if cur and text_width(trial, size, font == "F2") > avail:
                lines.append(cur)
                cur = w
            else:
                cur = trial
        if cur:
            lines.append(cur)
        self.y -= before
        for k, ln in enumerate(lines):
            self.ensure(lead)
            self.y -= lead
            self.run(MARGIN + indent + (hang if k else 0), self.y, ln, size, font, color)
        self.y -= after

    def labelled(self, label, value, size=8.3, indent=14, color=(0.1, 0.1, 0.1)):
        """« Libellé : valeur », la valeur coupée sous elle-même."""
        lead = size * 1.25
        lab = label + " : " if self.t is TXT["FR"] else label + ": "
        lw = text_width(lab, size, True)
        width = W - 2 * MARGIN - indent - lw
        words, lines, cur = value.split(" "), [], ""
        for w in words:
            trial = (cur + " " + w) if cur else w
            if cur and text_width(trial, size) > width:
                lines.append(cur)
                cur = w
            else:
                cur = trial
        lines.append(cur)
        for k, ln in enumerate(lines):
            self.ensure(lead)
            self.y -= lead
            if k == 0:
                self.run(MARGIN + indent, self.y, lab, size, "F2", (0.35, 0.35, 0.35))
            self.run(MARGIN + indent + lw, self.y, ln, size, "F1", color)

    def rule(self, r):
        t = self.t
        pats = r["seq"] if isinstance(r["seq"], list) else [r["seq"]]
        # la variante « 1NT (X) … » d'un motif « 1NT … » (contre ignoré) n'est pas répétée
        pats = [p for p in pats if not (p.startswith("1NT (X)") and ("1NT" + p[7:]) in pats)]
        seq = "  |  ".join(seq_text(p, t) for p in pats)
        head = "%s   →   %s" % (seq, call_text(str(r["call"]), t))
        tags = "%s · %s" % (t["forcing_names"].get(r["forcing"], r["forcing"]), t["status_names"].get(r["status"], r["status"]))
        if r.get("option"):
            tags += " · %s %s" % (t["option"], r["option"])
        self.ensure(40)
        self.y -= 4
        self.paragraph(head, size=9.2, font="F2", indent=0, hang=14, after=0)
        self.labelled(t["cond"], cond_text(r["cond"], t))
        meaning = r["meaning_en"] if t is TXT["EN"] else r["meaning"]
        self.labelled(t["meaning"], meaning)
        self.labelled(t["forcing"], tags, color=(0.4, 0.4, 0.4))

    def table_of_contents(self):
        """Une page de sommaire après l'introduction ; les numéros de page tiennent compte de son insertion."""
        body = self.pages
        self.pages = []
        self.new_page()
        self.paragraph(self.t["toc"], size=14, font="F2", color=(0.12, 0.3, 0.2), after=6)
        for title, page, _y in self.bookmarks:
            self.ensure(14)
            self.y -= 13
            num = str(page + 2)
            self.run(MARGIN, self.y, title, 9.5)
            self.run(W - MARGIN - text_width(num, 9.5), self.y, num, 9.5)
        toc = self.pages
        assert len(toc) == 1, "sommaire sur plus d'une page"
        self.pages = body[:1] + toc + body[1:]
        self.bookmarks = [(tt, p + 1 if p >= 1 else p, y) for tt, p, y in self.bookmarks]

    def heading(self, s, level):
        size = 14 if level == 1 else 10.5
        self.ensure(size * 3)
        if level == 1:
            self.bookmarks.append((s, len(self.pages) - 1, self.y))
        self.paragraph(s, size=size, font="F2", before=8 if level == 1 else 5, after=3,
                       color=(0.12, 0.3, 0.2) if level == 1 else (0.15, 0.15, 0.15))

    # ------------------------------------------------------------ écriture du fichier
    def write(self, path, subject):
        n = len(self.pages)
        objs = []                                   # corps des objets, numérotés à partir de 1

        def add(body):
            objs.append(body)
            return len(objs)

        fonts = {"F1": add(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"),
                 "F2": add(b"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>"),
                 "F4": add(b"<< /Type /Font /Subtype /Type1 /BaseFont /Symbol >>")}
        res = b"<< /Font << " + b" ".join(b"/%s %d 0 R" % (k.encode(), v) for k, v in fonts.items()) + b" >> >>"
        pages_id = len(objs) + 2 * n + 1            # réservé après les pages et leurs contenus
        page_ids = []
        for k, ops in enumerate(self.pages):
            foot = []
            self.ops = foot
            txt = "%s · %s %d / %d" % (self.title, self.t["page"], k + 1, n)
            self.run(W - MARGIN - text_width(txt, 7), MARGIN - 18, txt, 7, "F1", (0.5, 0.5, 0.5))
            stream = zlib.compress(b"\n".join(ops + foot), 9)
            cid = add(b"<< /Length %d /Filter /FlateDecode >>\nstream\n" % len(stream) + stream + b"\nendstream")
            page_ids.append(add(b"<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.2f %.2f] /Resources %s /Contents %d 0 R >>"
                                % (pages_id, W, H, res, cid)))
        assert add(b"<< /Type /Pages /Kids [" + b" ".join(b"%d 0 R" % p for p in page_ids) + b"] /Count %d >>" % n) == pages_id
        # signets : un par section
        outline_id = None
        if self.bookmarks:
            first = len(objs) + 2
            items = []
            for i, (title, page, y) in enumerate(self.bookmarks):
                me = first + i
                d = b"<< /Title %s /Parent %d 0 R /Dest [%d 0 R /XYZ 0 %.1f 0]" % (
                    self.utf16(title), first - 1, page_ids[page], y + 20)
                if i:
                    d += b" /Prev %d 0 R" % (me - 1)
                if i + 1 < len(self.bookmarks):
                    d += b" /Next %d 0 R" % (me + 1)
                items.append(d + b" >>")
            outline_id = add(b"<< /Type /Outlines /First %d 0 R /Last %d 0 R /Count %d >>"
                             % (first, first + len(items) - 1, len(items)))
            for it in items:
                add(it)
        cat = b"<< /Type /Catalog /Pages %d 0 R" % pages_id
        if outline_id:
            cat += b" /Outlines %d 0 R /PageMode /UseOutlines" % outline_id
        catalog = add(cat + b" >>")
        info = add(b"<< /Title %s /Subject (%s) /Producer (tools/python_tools/rules_pdf.py) >>" % (self.utf16(self.title), subject.encode()))
        out = io.BytesIO()
        out.write(b"%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
        offsets = []
        for i, body in enumerate(objs, 1):
            offsets.append(out.tell())
            out.write(b"%d 0 obj\n" % i + body + b"\nendobj\n")
        xref = out.tell()
        out.write(b"xref\n0 %d\n0000000000 65535 f \n" % (len(objs) + 1))
        for off in offsets:
            out.write(b"%010d 00000 n \n" % off)
        out.write(b"trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n" % (len(objs) + 1, catalog, info, xref))
        io.open(path, "wb").write(out.getvalue())

    @staticmethod
    def utf16(s):
        return b"<FEFF" + s.encode("utf-16-be").hex().upper().encode() + b">"


def fingerprint(path):
    """Empreinte du fichier de règles, fins de ligne ramenées à LF (comme tests.json)."""
    return hashlib.sha256(open(path, "rb").read().replace(b"\r\n", b"\n")).hexdigest()


def build(src, out, lang, title):
    t = TXT[lang]
    rules = sr.load(src)
    errs = sr.validate(rules)
    if errs:
        sys.exit("règles invalides : %s" % errs[:3])
    by_src = {}
    for r in rules:
        by_src.setdefault(r["_src"], []).append(r)
    doc = Doc(title, lang)
    doc.new_page()
    doc.paragraph(title, size=22, font="F2", color=(0.12, 0.3, 0.2), after=2)
    doc.paragraph(t["title"], size=12, color=(0.35, 0.35, 0.35), after=10)
    for p in t["intro"]:
        doc.paragraph(p, size=9.5, after=6)
    sha = fingerprint(src)
    doc.paragraph("%s : %s · %d %s · %s %s" % (t["source"], os.path.basename(src), len(rules), t["rules"], t["sha"], sha[:16] + "…"),
                  size=7.5, color=(0.45, 0.45, 0.45), before=6)
    for kind, value in outline(src, lang):
        if kind == "section":
            doc.new_page()
            doc.heading(value, 1)
        elif kind == "sub":
            doc.heading(value, 2)
        elif kind == "note":
            doc.paragraph(value, size=8.3, color=(0.3, 0.3, 0.3), after=1)
        elif kind == "rule":
            for r in by_src.get(value, []):
                doc.rule(r)
    doc.table_of_contents()
    doc.write(out, "sha256=%s" % sha)
    return len(doc.pages), len(rules)


def main():
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    ap.add_argument("rules", help="fichier de règles YAML (ex. ../../cli/systems/sef/rules.yaml)")
    ap.add_argument("-o", "--output", help="PDF à écrire (défaut : même nom que les règles, .pdf ; .en.pdf en anglais)")
    ap.add_argument("--lang", default="FR", type=str.upper, choices=["FR", "EN"])
    ap.add_argument("--title", default="SEF 2024", help="nom du système, en tête du document")
    a = ap.parse_args()
    out = a.output or os.path.splitext(a.rules)[0] + (".en.pdf" if a.lang == "EN" else ".pdf")
    pages, n = build(a.rules, out, a.lang, a.title)
    print("écrit %s : %d pages, %d règles" % (out, pages, n))


if __name__ == "__main__":
    main()
