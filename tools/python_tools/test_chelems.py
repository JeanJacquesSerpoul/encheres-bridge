"""Évalue les chelems du système SEF_2024.md au double mort (endplay/dds)."""
import random, sys
from collections import Counter
from endplay.types import Deal, Denom, Player
from endplay.dds import calc_dd_table
import sef_rules as sr

rules = sr.load('sef_rules.yaml'); assert not sr.validate(rules)
N = int(sys.argv[1]) if len(sys.argv) > 1 else 20000
rng = random.Random(2026)
DEN = {'C': Denom.clubs, 'D': Denom.diamonds, 'H': Denom.hearts, 'S': Denom.spades, 'NT': Denom.nt}

def auction(hands):
    calls, who, opened, last, trump, bidder = [], 0, False, None, None, {}
    for _ in range(24):
        r = sr.choose(rules, ' '.join(calls), hands[who], (), trump)
        if r is None:
            return None, None
        if r.get('trump'):
            trump = r['trump']
        c = r['call']; calls.append(c)
        if c not in ('P', 'X', 'XX'):
            bidder.setdefault(c[1:], who)
            opened = True
        elif opened or len(calls) >= 2:
            break
        who = 1 - who
    bids = [c for c in calls if c not in ('P', 'X', 'XX')]
    if not bids:
        return 'P', None
    fin = bids[-1]
    return fin, bidder[fin[1:]]

res = Counter(); missed = Counter(); ex = []
for i in range(N):
    h = sr.deal(rng)   # 0=N 1=S 2=E 3=W
    fin, decl = auction(h)
    if fin is None:
        continue
    hcp = sum(sr.features(h[k])['hcp'] for k in (0, 1))
    is_slam = fin != 'P' and fin[0] in '67'
    if not is_slam and hcp < 29:
        continue
    pbn = 'N:' + ' '.join('.'.join(h[k][s] for s in 'SHDC') for k in (0, 2, 1, 3))
    tab = calc_dd_table(Deal(pbn))
    best_ns = lambda d: max(tab[d, Player.north], tab[d, Player.south])
    top = max(best_ns(DEN[s]) for s in DEN)
    if is_slam:
        need = 6 + int(fin[0]) if False else int(fin[0]) + 6
        got = tab[DEN[fin[1:]], Player.north if decl == 0 else Player.south]
        ok = got >= need
        res['chelem ' + fin[0] + (' réussi' if ok else ' chuté')] += 1
        if not ok and len(ex) < 8:
            ex.append((pbn, fin, got))
    elif top >= 12:
        missed['chelem possible non demandé (≥12 levées DD)'] += 1
        if hcp >= 33:
            missed['  dont avec 33+ H à deux'] += 1
            missed['    fin: ' + fin] += 1

print('Donnes :', N)
for k, v in sorted(res.items()):
    print(' ', k, v)
for k, v in missed.items():
    print(' ', k, v)
print('Exemples de chelems chutés (N E S W, contrat, levées DD) :')
for e in ex:
    print('  ', *e)
