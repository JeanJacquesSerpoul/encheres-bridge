#!/usr/bin/env python3
"""Moteur de lecture des règles d'enchères : cli/systems/<id>/rules.yaml (source), ou blocs ```yaml sef-rules d'un .md.

Chaque système d'enchères vit dans cli/systems/<id>/, où le moteur Go/WASM de la page lit ses règles au
chargement ; le SEF 2024 est cli/systems/sef/rules.yaml. Ses données de référence sont dans
engine/testdata/systems/<id>/ (tools/python_tools/regen_system.py les régénère toutes).

Usage :
  python3 sef_rules.py ../../cli/systems/sef/rules.yaml --validate          # vérifie le fichier
  python3 sef_rules.py ../../cli/systems/sef/rules.yaml --json rules.json   # exporte les règles expansées
  python3 sef_rules.py ../../cli/systems/sef/rules.yaml --simulate 1000     # enchères sans intervention sur donnes aléatoires
  python3 sef_rules.py ../../cli/systems/sef/rules.yaml --hand "AK32.KQ4.A32.J32" --seq "1NT 2C"
  python3 sef_rules.py ../../cli/systems/sef/rules.yaml --gen-tests 1500 tests.json   # génère les cas de test
  python3 sef_rules.py ../../cli/systems/sef/rules.yaml --check-tests tests.json      # rejoue les cas de test
"""
import argparse, ast, itertools, json, os, random, re, sys
from collections import Counter

import yaml

HERE = os.path.dirname(os.path.abspath(__file__))
DEFAULT_RULES = os.path.normpath(os.path.join(HERE, '..', '..', 'cli', 'systems', 'sef', 'rules.yaml'))

SUITS = 'SHDC'
RANKS = 'AKQJT98765432'
CALL_RE = re.compile(r'^(P|X|XX|[1-7](C|D|H|S|NT))$')
FORCING = {'NF', 'F1', 'FM', 'SO', 'INV', 'REL', 'ASK', 'TO', 'PEN'}
STATUS = {'sef', 'choix', 'sef2018', 'infere', 'a_verifier'}
FIELDS = {'id', 'seq', 'call', 'cond', 'forcing', 'meaning', 'meaning_en', 'status', 'alert', 'option', 'for', 'trump',
          'natural'}
NAMES = {'S', 'H', 'D', 'C', 'hcp', 'hl', 'dh', 'hld', 'shape', 'balanced', 'semibalanced',
         'aces', 'kings', 'losers', 'ptricks', 'qtricks', 'sidetricks', 'ace', 'king', 'queen', 'top', 'solid', 'stop',
         'short', 'hcp_in', 'keycards', 'ctrl1', 'ctrl2', 'max', 'min', 'true', 'false', 'True', 'False',
         'vul', 'opp_vul', 'seat', 'lvl', 'p_suit', 'p_len', 'fit', 'p_forcing'}
NODES = (ast.Expression, ast.BoolOp, ast.And, ast.Or, ast.UnaryOp, ast.Not, ast.USub, ast.Compare,
         ast.Eq, ast.NotEq, ast.Lt, ast.LtE, ast.Gt, ast.GtE, ast.In, ast.NotIn, ast.BinOp, ast.Add,
         ast.Sub, ast.Name, ast.Load, ast.Constant, ast.Call, ast.Tuple)


# ---------------------------------------------------------------- lecture
def load(path):
    text = open(path, encoding='utf-8').read()
    if path.endswith(('.yaml', '.yml')):          # fichier YAML seul
        blocks = [text]
    else:                                          # blocs ```yaml sef-rules d'un .md
        blocks = re.findall(r'^```yaml sef-rules\n(.*?)^```', text, re.S | re.M)
    raw = []
    for b in blocks:
        raw.extend(yaml.safe_load(b) or [])
    rules = []
    for r in raw:
        subs = r.get('for') or [{}]
        for s in subs:
            e = {}
            for k, v in r.items():
                if k == 'for':
                    continue
                if isinstance(v, str):
                    for sk, sv in s.items():
                        v = v.replace('{%s}' % sk, str(sv))
                elif isinstance(v, list) and k == 'seq':
                    for sk, sv in s.items():
                        v = [x.replace('{%s}' % sk, str(sv)) for x in v]
                e[k] = v
            e['_src'] = r['id']
            rules.append(e)
    return rules


def compile_cond(expr):
    expr = expr.strip()
    expr = re.sub(r'\btrue\b', 'True', re.sub(r'\bfalse\b', 'False', expr))
    tree = ast.parse(expr, mode='eval')
    for n in ast.walk(tree):
        if not isinstance(n, NODES):
            raise ValueError('construction interdite : %s' % type(n).__name__)
        if isinstance(n, ast.Name) and n.id not in NAMES:
            raise ValueError('nom inconnu : %s' % n.id)
    return compile(tree, '<cond>', 'eval')


def validate(rules):
    errs = []
    ids = Counter(r.get('id') for r in rules)
    for r in rules:
        rid = r.get('id', '?')
        miss = {'id', 'seq', 'call', 'cond', 'forcing', 'meaning', 'meaning_en', 'status'} - set(r)
        if miss:
            errs.append('%s : champs manquants %s' % (rid, sorted(miss)))
            continue
        extra = set(r) - FIELDS - {'_src'}
        if extra:
            errs.append('%s : champs inconnus %s' % (rid, sorted(extra)))
        if ids[rid] > 1:
            errs.append('%s : id en double' % rid)
        if not CALL_RE.match(str(r['call'])):
            errs.append('%s : enchère invalide %r' % (rid, r['call']))
        if r['forcing'] not in FORCING:
            errs.append('%s : forcing invalide %r' % (rid, r['forcing']))
        if r.get('trump') and r['trump'] not in ('S', 'H', 'D', 'C'):
            errs.append('%s : trump invalide %r' % (rid, r['trump']))
        if r['status'] not in STATUS:
            errs.append('%s : status invalide %r' % (rid, r['status']))
        if 'natural' in r and not isinstance(r['natural'], bool):
            errs.append('%s : natural doit valoir true ou false' % rid)
        if any(isinstance(v, str) and k != '_src' and re.search(r'\{[A-Za-z]+\}', v) for k, v in r.items()):
            errs.append('%s : substitution non résolue' % rid)
        pats = r['seq'] if isinstance(r['seq'], list) else [r['seq']]
        for tok in ' '.join(map(str, pats)).split():
            if tok.startswith('BW:') or tok in ('*', '**'):
                continue
            for alt in tok.strip('()').split('|'):
                if not CALL_RE.match(alt):
                    errs.append('%s : séquence invalide %r' % (rid, r['seq']))
        try:
            r['_code'] = compile_cond(str(r['cond']))
        except Exception as ex:
            errs.append('%s : condition %r -> %s' % (rid, r['cond'], ex))
    return errs


_RANK = {'C': 0, 'D': 1, 'H': 2, 'S': 3, 'NT': 4}


def _illegal(call, seq):
    """Raison pour laquelle `call` est illégale après la séquence concrète `seq`, ou None."""
    if call == 'P':
        return None
    if call in ('X', 'XX'):
        last = next((t for t in reversed(seq) if t != 'P'), None)
        if last is None or not last.startswith('('):
            return 'contre sans enchère adverse à contrer'
        inner = last.strip('()')
        if call == 'X' and not inner[0].isdigit():
            return 'contre d\'un contre'
        if call == 'XX' and inner != 'X':
            return 'surcontre sans contre adverse'
        return None
    bids = [t.strip('()') for t in seq if t.strip('()')[0].isdigit()]
    if bids and (int(call[0]), _RANK[call[1:]]) <= (int(bids[-1][0]), _RANK[bids[-1][1:]]):
        return '%s n\'est pas au-dessus de %s' % (call, bids[-1])
    return None


def legality_warnings(rules):
    """Règles dont l'enchère est illégale pour une séquence que leur motif accepte.

    La séquence vue par la paire contient toutes les enchères de la table : la dernière enchère du
    motif est la dernière de la table. Un joker `*` placé après la dernière enchère connue empêche de
    conclure ; le moteur passerait alors (« passe par défaut ») au lieu de suivre la règle."""
    out = []
    for r in rules:
        pats = r['seq'] if isinstance(r['seq'], list) else [r['seq']]
        for pat in pats:
            toks = pat.split()
            if toks and (toks[0] == '**' or toks[0].startswith('BW:')):
                toks = toks[1:]
            if '*' in toks:
                last = max(i for i, t in enumerate(toks) if t == '*')
                if not any(a[0].isdigit() for t in toks[last + 1:] for a in t.strip('()').split('|')):
                    continue
                toks = ['P' if t == '*' else t for t in toks]
            choices = [['(%s)' % a if t.startswith('(') else a for a in t.strip('()').split('|')] for t in toks]
            reasons = sorted({_illegal(str(r['call']), list(s)) for s in itertools.product(*choices)} - {None})
            if reasons:
                out.append('%s : séquence %r -> %s' % (r.get('id', '?'), pat, '; '.join(reasons)))
    return out


# ---------------------------------------------------------------- main
def parse_hand(s):
    parts = s.split('.')
    assert len(parts) == 4, 'main au format S.H.D.C'
    hand = {k: ('' if p == '-' else p.upper().replace('10', 'T')) for k, p in zip(SUITS, parts)}
    return {k: ''.join(sorted(v, key=RANKS.index)) for k, v in hand.items()}


def ptricks(h, L):
    tot = 0.0
    for s in SUITS:
        c, n = h[s], L[s]
        a, k, q = 'A' in c, 'K' in c, 'Q' in c
        t = (1 if a else 0)
        if k and n >= 2:
            t += 1 if a else 0.5
        if q and n >= 3:
            t += 1 if (a and k) else (0.5 if (a or k) else 0)
        tot += min(n, t) + (0.5 + n - 4 if n >= 4 else 0)
    return tot


def length_points(c, n):
    """1 point par carte à partir de la 5e, dans une couleur commandée par au moins D V
    (deux honneurs parmi A R D V)."""
    return max(0, n - 4) if sum(1 for x in c if x in 'AKQJ') >= 2 else 0


def devalued(c, n):
    """Honneur sec (As compris) ou deux honneurs secs : 1 point de moins."""
    return 1 <= n <= 2 and all(x in 'AKQJ' for x in c)


def quick_tricks(c, n):
    """Levées de défense d'une couleur : AR 2 ; AD 1,5 ; A 1 ; RD 1 ; R (au moins second) 0,5."""
    if 'A' in c:
        return 2 if 'K' in c else (1.5 if 'Q' in c else 1)
    if 'K' in c:
        return 1 if 'Q' in c else (0.5 if n >= 2 else 0)
    return 0


def features(h):
    L = {s: len(h[s]) for s in SUITS}
    hv = {'A': 4, 'K': 3, 'Q': 2, 'J': 1}
    hcp_in = {s: sum(hv.get(c, 0) for c in h[s]) for s in SUITS}
    hcp = sum(hcp_in.values())
    shortp = sum({0: 3, 1: 2, 2: 1}.get(L[s], 0) for s in SUITS)
    deval = sum(1 for s in SUITS if devalued(h[s], L[s]))
    hl = hcp + sum(length_points(h[s], L[s]) for s in SUITS) - deval
    ls = sorted(L.values(), reverse=True)
    shape = ''.join(str(x) for x in ls)
    bico = 4 if ls[1] >= 5 and ls[0] + ls[1] >= 11 else 0     # bicolore 6-5 ou plus : +4 HLD

    def losers_suit(s):
        cards, n = h[s], L[s]
        top = cards[:min(n, 3)]
        want = 'AKQ'[:min(n, 3)]
        return sum(1 for w in want if w not in top)
    losers = sum(losers_suit(s) for s in SUITS)

    def stop(s):
        c, n = h[s], L[s]
        return 'A' in c or ('K' in c and n >= 2) or ('Q' in c and n >= 3) or ('J' in c and n >= 4)

    qt = {s: quick_tricks(h[s], L[s]) for s in SUITS}
    longest = max(SUITS, key=lambda s: L[s])      # égalité : la couleur la plus haute (♠ > ♥ > ♦ > ♣)
    env = dict(L)
    env.update(
        qtricks=sum(qt.values()), sidetricks=sum(qt.values()) - qt[longest],
        hcp=hcp, hl=hl, dh=hcp + shortp - deval, hld=hl + shortp + bico, shape=shape,
        balanced=shape in ('4333', '4432', '5332'), semibalanced=shape in ('5422', '6322'),
        aces=sum(h[s].count('A') for s in SUITS), kings=sum(h[s].count('K') for s in SUITS),
        losers=losers, ptricks=ptricks(h, L),
        ace=lambda s: 'A' in h[s], king=lambda s: 'K' in h[s], queen=lambda s: 'Q' in h[s],
        top=lambda s: sum(1 for c in 'AKQ' if c in h[s]), solid=lambda s: all(c in h[s] for c in 'AKQ'),
        stop=stop, short=lambda s: L[s] <= 1, hcp_in=lambda s: hcp_in[s],
        keycards=lambda s: sum(h[x].count('A') for x in SUITS) + ('K' in h[s]),
        ctrl1=lambda s: 'A' in h[s] or L[s] == 0,
        ctrl2=lambda s: 'A' in h[s] or 'K' in h[s] or L[s] <= 1,
        max=max, min=min)
    return env


# ---------------------------------------------------------------- choix
def _match(pattern, seq):
    p, s = pattern.split(), seq.split()
    if p and p[0] == '**':            # ** = n'importe quel début (0 ou plusieurs enchères)
        p = p[1:]
        if len(s) < len(p):
            return False
        s = s[len(s) - len(p):] if p else []
    if len(p) != len(s):
        return False
    for pt, st in zip(p, s):
        if pt == '*':
            continue
        if pt.startswith('(') != st.startswith('('):
            return False
        if st.strip('()') not in pt.strip('()').split('|'):
            return False
    return True


def rule_matches(r, seq, trump=None):
    pats = r['seq'] if isinstance(r['seq'], list) else [r['seq']]
    stripped = seq
    while stripped.split()[:1] == ['P']:
        stripped = stripped.split(' ', 1)[1] if ' ' in stripped else ''
    for pat in pats:
        head = pat.split()[:1]
        if head and head[0].startswith('BW:'):   # BW:x = n'importe quel début, atout convenu = x
            if trump != head[0][3:]:
                continue
            pat = '** ' + pat.split(' ', 1)[1]
        target = seq if head == ['P'] else stripped
        if _match(pat, target):
            return True
    return False


# Enchères qui obligent le partenaire à reparler (p_forcing).
CTX_FORCING = {'F1', 'FM', 'REL', 'ASK', 'TO'}


def is_natural(rule):
    """Une enchère à la couleur est naturelle si sa règle n'est ni alertée ni marquée natural: false."""
    return bool(rule) and not rule.get('alert') and rule.get('natural', True)


def context(history, seat):
    """Contexte de l'enchère vu par le joueur `seat` (0 = Nord … 3 = Ouest).
    history : liste de (siège, appel, règle ou None), dans l'ordre. Rend lvl, p_suit, p_len, p_forcing
    (fit se déduit de la main : longueur dans p_suit plus p_len). Voir SEF_2024_spec.md."""
    lvl, opening = 0, None
    for i, (_, call, _r) in enumerate(history):
        if call[0].isdigit():
            lvl = max(lvl, int(call[0]))
            if opening is None:
                opening = i
    partner = (seat + 2) % 4
    p_forcing = False
    for who, _call, r in reversed(history):
        if who == partner:
            p_forcing = bool(r) and r['forcing'] in CTX_FORCING
            break
    p_suit, p_len = '', 0
    for i in range(len(history) - 1, -1, -1):
        who, call, r = history[i]
        if who == partner and call[0].isdigit() and call[1:] in ('C', 'D', 'H', 'S') and is_natural(r):
            p_suit, level = call[1:], int(call[0])
            if i == opening:
                p_len = (5 if p_suit in 'HS' else 4) if level == 1 else 6
            elif (who % 2) != (history[opening][0] % 2):
                p_len = 5
            else:
                p_len = 4
            again = sum(1 for w, c, rr in history
                        if w == partner and c[0].isdigit() and c[1:] == p_suit and is_natural(rr))
            if again >= 2:
                p_len = min(max(p_len + 1, 5), 6)
            break
    return {'lvl': lvl, 'p_suit': p_suit, 'p_len': p_len, 'p_forcing': p_forcing}


def seq_level(seq):
    """Palier le plus élevé d'une séquence, quand l'historique complet manque."""
    lv = [int(a[0]) for t in seq.split() for a in t.strip('()').split('|') if a[:1].isdigit()]
    return max(lv or [0])


def choose(rules, seq, hand, options=(), trump=None, vul=False, opp_vul=False, seat=1, ctx=None):
    """vul / opp_vul : vulnérabilité du camp qui parle / du camp adverse.
    seat : rang du joueur qui parle dans le tour d'enchères (1 = donneur … 4).
    ctx : contexte de l'enchère (voir context) ; par défaut, lvl lu dans seq, sans couleur du partenaire."""
    env = features(hand)
    env.update(vul=bool(vul), opp_vul=bool(opp_vul), seat=int(seat))
    ctx = ctx or {'lvl': seq_level(seq), 'p_suit': '', 'p_len': 0, 'p_forcing': False}
    env.update(lvl=int(ctx['lvl']), p_suit=ctx['p_suit'], p_len=int(ctx['p_len']), p_forcing=bool(ctx['p_forcing']))
    env['fit'] = env[ctx['p_suit']] + int(ctx['p_len']) if ctx['p_suit'] else 0
    for r in rules:
        if r.get('option') and r['option'] not in options:
            continue
        if rule_matches(r, seq, trump) and eval(r['_code'], {'__builtins__': {}}, env):
            return r
    return None


# ---------------------------------------------------------------- simulation
def deal(rng):
    deck = [s + r for s in SUITS for r in RANKS]
    rng.shuffle(deck)
    hands = []
    for i in range(4):
        cards = deck[13 * i:13 * i + 13]
        hands.append({s: ''.join(sorted((c[1] for c in cards if c[0] == s), key=RANKS.index)) for s in SUITS})
    return hands


def simulate(rules, n, seed, options, show):
    rng = random.Random(seed)
    gaps, finals, used, fpass = Counter(), Counter(), Counter(), Counter()
    examples = []
    for _ in range(n):
        hands = deal(rng)
        # camp NS : Nord puis Sud ; les adversaires passent toujours
        calls, who, opened, last, trump = [], 0, False, None, None
        for _step in range(24):
            seq = ' '.join(calls)
            r = choose(rules, seq, hands[who], options, trump)
            if r is None:
                gaps[seq] += 1
                calls.append('?')
                break
            used[r['id']] += 1
            if r['call'] == 'P' and last is not None and last['forcing'] in ('F1', 'FM', 'REL', 'ASK'):
                fpass[seq] += 1
            last = r
            if r.get('trump'):
                trump = r['trump']
            calls.append(r['call'])
            if r['call'] == 'P' and (opened or len(calls) >= 2):
                break
            if r['call'] != 'P':
                opened = True
            who = 1 - who
        auction = ' '.join(calls)
        finals[auction] += 1
        if len(examples) < show:
            examples.append((hands[0], hands[1], auction))
    return gaps, finals, used, examples, fpass


FEATS = ('S', 'H', 'D', 'C', 'hcp', 'hl', 'dh', 'hld', 'shape', 'balanced', 'semibalanced',
         'aces', 'kings', 'losers', 'ptricks', 'qtricks', 'sidetricks')
FUNCS = ('ace', 'king', 'queen', 'top', 'solid', 'stop', 'short', 'hcp_in', 'keycards', 'ctrl1', 'ctrl2')


def feature_dump(h):
    env = features(h)
    out = {k: env[k] for k in FEATS}
    for f in FUNCS:
        out[f] = {s: env[f](s) if not isinstance(env[f](s), bool) else env[f](s) for s in SUITS}
        out[f] = {s: (int(v) if isinstance(v, bool) and f in ('keycards',) else v) for s, v in out[f].items()}
    return out


def gen_tests(rules, md, n, seed, path):
    import hashlib
    rng = random.Random(seed)
    cases, seen = [], set()
    for i in range(n + n // 2):
        opts = ('checkback2018',) if i % 4 == 3 else ()
        # vulnérabilité (camp N/S, camp adverse) : les quatre cas en rotation, sans toucher au tirage
        vul, opp_vul = ((False, False), (True, False), (False, True), (True, True))[(i // 4) % 4]
        # rang de Nord : donneur (1) ou 2e (Est a passé) ; Sud parle en 3e ou 4e position
        north = 1 + (i // 16) % 2
        hands = deal(rng)
        if i >= n:   # donnes fortes : couvrent Blackwood, contrôles et chelems
            while features(hands[0])['hcp'] + features(hands[1])['hcp'] < 31:
                hands = deal(rng)
        calls, who, opened, trump, history = [], 0, False, None, []
        for _ in range(24):
            seq = ' '.join(calls)
            seat = north + 2 * who
            ctx = context(history, 2 * who)          # Nord = siège 0, Sud = siège 2 ; E-O passent
            r = choose(rules, seq, hands[who], opts, trump, vul, opp_vul, seat, ctx)
            key = (fmt(hands[who]), seq, trump, opts, vul, opp_vul, seat, tuple(ctx.values()))
            if key not in seen:
                seen.add(key)
                cases.append({'id': len(cases) + 1, 'hand': fmt(hands[who]), 'seq': seq, 'trump': trump,
                              'options': list(opts), 'vul': vul, 'opp_vul': opp_vul, 'seat': seat, **ctx,
                              'expected': {'call': r['call'] if r else None, 'rule': r['id'] if r else None},
                              'features': feature_dump(hands[who])})
            if r is None:
                break
            if r.get('trump'):
                trump = r['trump']
            history.append((2 * who, r['call'], r))
            calls.append(r['call'])
            if r['call'] == 'P' and (opened or len(calls) >= 2):
                break
            if r['call'] != 'P':
                opened = True
            who = 1 - who
    data = {'format': 'sef-tests/1', 'source': os.path.basename(md),
            # empreinte indépendante des fins de ligne (checkout Windows CRLF ou CI LF)
            'source_sha256': hashlib.sha256(open(md, 'rb').read().replace(b'\r\n', b'\n')).hexdigest(),
            'seed': seed, 'deals': n + n // 2, 'strong_deals': n // 2, 'count': len(cases), 'cases': cases}
    with open(path, 'w', encoding='utf-8') as f:   # un cas par ligne
        head = json.dumps({k: v for k, v in data.items() if k != 'cases'}, ensure_ascii=False)[:-1]
        f.write(head + ', "cases": [\n')
        f.write(',\n'.join(json.dumps(c, ensure_ascii=False, separators=(',', ':')) for c in cases))
        f.write('\n]}\n')
    print('écrit %s : %d cas' % (path, len(cases)))


def check_tests(rules, path):
    data = json.load(open(path, encoding='utf-8'))
    bad = 0
    for c in data['cases']:
        h = parse_hand(c['hand'])
        ctx = {k: c[k] for k in ('lvl', 'p_suit', 'p_len', 'p_forcing')} if 'lvl' in c else None
        r = choose(rules, c['seq'], h, tuple(c['options']), c['trump'], c.get('vul', False), c.get('opp_vul', False),
                   c.get('seat', 1), ctx)
        got = {'call': r['call'] if r else None, 'rule': r['id'] if r else None}
        if got != c['expected'] or feature_dump(h) != c['features']:
            bad += 1
            if bad <= 10:
                print('  échec cas %d : %s | %s -> %s (attendu %s)' % (c['id'], c['hand'], c['seq'], got, c['expected']))
    print('%d cas, %d échecs' % (len(data['cases']), bad))
    return bad


def fmt(h):
    return '.'.join(h[s] or '-' for s in SUITS)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('md')
    ap.add_argument('--validate', action='store_true')
    ap.add_argument('--json')
    ap.add_argument('--simulate', type=int)
    ap.add_argument('--seed', type=int, default=1)
    ap.add_argument('--option', action='append', default=[])
    ap.add_argument('--show', type=int, default=0)
    ap.add_argument('--hand')
    ap.add_argument('--seq', default='')
    ap.add_argument('--trump')
    ap.add_argument('--vul', action='store_true', help='avec --hand : le camp qui parle est vulnérable')
    ap.add_argument('--opp-vul', action='store_true', help='avec --hand : les adversaires sont vulnérables')
    ap.add_argument('--seat', type=int, default=1, choices=[1, 2, 3, 4],
                    help="avec --hand : rang du joueur dans le tour d'enchères (1 = donneur)")
    ap.add_argument('--lang', default='FR', type=str.upper, choices=['FR', 'EN'], help='FR=Français (défaut), EN=Anglais')
    ap.add_argument('--gen-tests', nargs=2, metavar=('DONNES', 'FICHIER'))
    ap.add_argument('--check-tests', metavar='FICHIER')
    a = ap.parse_args()

    rules = load(a.md)
    errs = validate(rules)
    if a.validate or errs:
        print('%d règles (après expansion), %d erreurs' % (len(rules), len(errs)))
        for e in errs:
            print('  -', e)
        print('Statuts :', dict(Counter(r.get('status') for r in rules)))
        if errs:
            sys.exit(1)
        warns = legality_warnings(rules)
        print('%d avertissement(s) : enchère illégale' % len(warns))
        for w in warns:
            print('  !', w)
    if a.json:
        out = [{k: v for k, v in r.items() if not k.startswith('_')} for r in rules]
        json.dump(out, open(a.json, 'w', encoding='utf-8'), ensure_ascii=False, indent=1)
        print('exporté :', a.json)
    if a.gen_tests:
        gen_tests(rules, a.md, int(a.gen_tests[0]), a.seed, a.gen_tests[1])
    if a.check_tests:
        sys.exit(1 if check_tests(rules, a.check_tests) else 0)
    if a.hand is not None:
        r = choose(rules, a.seq, parse_hand(a.hand), a.option, a.trump, a.vul, a.opp_vul, a.seat)
        print('Aucune règle (séquence non codée)' if r is None else
              '%s  [%s, %s, %s]  %s' % (r['call'], r['forcing'], r['status'], r['id'], r['meaning_en'] if a.lang == 'EN' else r['meaning']))
    if a.simulate:
        gaps, finals, used, ex, fpass = simulate(rules, a.simulate, a.seed, a.option, a.show)
        tot = a.simulate
        print('\nDonnes : %d ; enchères terminées sans trou : %d (%.1f %%)'
              % (tot, tot - sum(gaps.values()), 100 * (tot - sum(gaps.values())) / tot))
        print('Séquences non codées les plus fréquentes :')
        for s, c in gaps.most_common(15):
            print('  %5d  "%s"' % (c, s))
        slams = Counter(a.split()[-2] for a in finals.elements() if len(a.split()) >= 2 and a.split()[-2][0] in '67')
        print('Chelems demandés :', dict(sorted(slams.items())))
        print('Passes sur une enchère forcing : %d' % sum(fpass.values()))
        for s, c in fpass.most_common(10):
            print('  %5d  "%s"' % (c, s))
        unused = sorted(r['id'] for r in rules if used[r['id']] == 0)
        print('Règles jamais utilisées : %d / %d' % (len(unused), len(rules)))
        for n_, h0, h1 in [(i,) + e[:2] for i, e in enumerate(ex)]:
            print('  N %s  S %s  :  %s' % (fmt(h0), fmt(h1), ex[n_][2]))


if __name__ == '__main__':
    main()
