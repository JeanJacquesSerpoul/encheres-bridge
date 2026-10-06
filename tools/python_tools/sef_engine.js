// Réimplémentation indépendante, écrite d'après SEF_2024_spec.md uniquement.
const fs = require('fs');
const rules = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const tests = JSON.parse(fs.readFileSync(process.argv[3], 'utf8'));
const SUITS = 'SHDC', RANKS = 'AKQJT98765432';
function parseHand(s) { const p = s.split('.'); const h = {};
  SUITS.split('').forEach((k, i) => h[k] = (p[i] === '-' ? '' : p[i]).split('').sort((a, b) => RANKS.indexOf(a) - RANKS.indexOf(b)).join('')); return h; }
function feats(h) {
  const L = {}, hv = { A: 4, K: 3, Q: 2, J: 1 }, hin = {}; let hcp = 0, hl = 0, sp = 0, dv = 0, losers = 0, pt = 0, qtr = 0, longest = 'S', qlong = 0;
  for (const s of SUITS) { const c = h[s], n = c.length; L[s] = n; hin[s] = [...c].reduce((a, x) => a + (hv[x] || 0), 0); hcp += hin[s];
    const hon = [...c].filter(x => 'AKQJ'.includes(x)).length;
    if (hon >= 2) hl += Math.max(0, n - 4); if ((n === 1 || n === 2) && hon === n) dv++; sp += ({ 0: 3, 1: 2, 2: 1 })[n] || 0;
    const k = Math.min(n, 3), top = c.slice(0, k); losers += [...'AKQ'.slice(0, k)].filter(w => !top.includes(w)).length;
    const A = c.includes('A'), K = c.includes('K'), Q = c.includes('Q'); let t = A ? 1 : 0;
    if (K && n >= 2) t += A ? 1 : 0.5; if (Q && n >= 3) t += (A && K) ? 1 : ((A || K) ? 0.5 : 0);
    pt += Math.min(n, t) + (n >= 4 ? n - 3.5 : 0);
    const q = A ? (K ? 2 : (Q ? 1.5 : 1)) : (K ? (Q ? 1 : (n >= 2 ? 0.5 : 0)) : 0); qtr += q;
    if (s === 'S' || n > L[longest]) { longest = s; qlong = q; } }
  hl += hcp - dv; const ls = Object.values(L).sort((a, b) => b - a), shape = ls.join('');
  const bico = ls[1] >= 5 && ls[0] + ls[1] >= 11 ? 4 : 0;   // bicolore 6-5 ou plus : +4 HLD
  const aces = SUITS.split('').filter(s => h[s].includes('A')).length;
  return { S: L.S, H: L.H, D: L.D, C: L.C, hcp, hl, dh: hcp + sp - dv, hld: hl + sp + bico, shape,
    balanced: ['4333', '4432', '5332'].includes(shape), semibalanced: ['5422', '6322'].includes(shape),
    aces, kings: SUITS.split('').filter(s => h[s].includes('K')).length, losers, ptricks: pt, qtricks: qtr, sidetricks: qtr - qlong,
    ace: s => h[s].includes('A'), king: s => h[s].includes('K'), queen: s => h[s].includes('Q'),
    top: s => [...'AKQ'].filter(x => h[s].includes(x)).length, solid: s => [...'AKQ'].every(x => h[s].includes(x)),
    stop: s => { const c = h[s], n = c.length; return c.includes('A') || (c.includes('K') && n >= 2) || (c.includes('Q') && n >= 3) || (c.includes('J') && n >= 4); },
    short: s => L[s] <= 1, hcp_in: s => hin[s], keycards: s => aces + (h[s].includes('K') ? 1 : 0),
    ctrl1: s => h[s].includes('A') || L[s] === 0, ctrl2: s => h[s].includes('A') || h[s].includes('K') || L[s] <= 1,
    max: (...a) => Math.max(...a), min: (...a) => Math.min(...a) };
}
// --- parseur d'expressions (sous-ensemble Python du § 4.1)
function tokenize(s) { const re = /\s*(\d+\.\d+|\d+|'[^']*'|[A-Za-z_]\w*|<=|>=|==|!=|[<>()+\-,])/y; const out = []; let m;
  while (re.lastIndex < s.length) { m = re.exec(s); if (!m) { if (/^\s*$/.test(s.slice(re.lastIndex))) break; throw Error('lex ' + s); } out.push(m[1]); } return out; }
function compile(src) { const tk = tokenize(src); let i = 0; const peek = () => tk[i], next = () => tk[i++];
  const expect = t => { if (next() !== t) throw Error('attendu ' + t + ' dans ' + src); };
  function or() { let l = and(); while (peek() === 'or') { next(); const a = l, b = and(); l = e => a(e) || b(e); } return l; }
  function and() { let l = not(); while (peek() === 'and') { next(); const a = l, b = not(); l = e => a(e) && b(e); } return l; }
  function not() { if (peek() === 'not') { next(); const a = not(); return e => !a(e); } return cmp(); }
  const OPS = ['<', '<=', '>', '>=', '==', '!=', 'in'];
  function cmp() { const first = add(); const parts = [];
    while (OPS.includes(peek()) || (peek() === 'not' && tk[i + 1] === 'in')) { let op = next(); if (op === 'not') { next(); op = 'notin'; } parts.push([op, add()]); }
    if (!parts.length) return first;
    return e => { let l = first(e); for (const [op, f] of parts) { const r = f(e); let ok;
        switch (op) { case '<': ok = l < r; break; case '<=': ok = l <= r; break; case '>': ok = l > r; break; case '>=': ok = l >= r; break;
          case '==': ok = l === r; break; case '!=': ok = l !== r; break; case 'in': ok = r.includes(l); break; case 'notin': ok = !r.includes(l); }
        if (!ok) return false; l = r; } return true; }; }
  function add() { let l = unary(); while (peek() === '+' || peek() === '-') { const op = next(), a = l, b = unary();
      l = op === '+' ? e => +a(e) + +b(e) : e => +a(e) - +b(e); } return l; }
  function unary() { if (peek() === '-') { next(); const a = unary(); return e => -a(e); } return atom(); }
  function atom() { const t = next();
    if (t === '(') { const items = [or()]; let tuple = false; while (peek() === ',') { next(); tuple = true; if (peek() === ')') break; items.push(or()); }
      expect(')'); return tuple ? e => items.map(f => f(e)) : items[0]; }
    if (/^\d/.test(t)) { const v = parseFloat(t); return () => v; }
    if (t[0] === "'") { const v = t.slice(1, -1); return () => v; }
    if (t === 'true') return () => true; if (t === 'false') return () => false;
    if (peek() === '(') { next(); const args = []; if (peek() !== ')') { args.push(or()); while (peek() === ',') { next(); args.push(or()); } } expect(')');
      return e => e[t](...args.map(f => f(e))); }
    return e => { if (!(t in e)) throw Error('nom inconnu ' + t); return e[t]; }; }
  const f = or(); if (i !== tk.length) throw Error('reste ' + tk.slice(i)); return f; }
// --- séquences (§ 3)
function tokMatch(pt, st) { if (pt === '*') return true; if (pt.startsWith('(') !== st.startsWith('(')) return false;
  return pt.replace(/[()]/g, '').split('|').includes(st.replace(/[()]/g, '')); }
function match(pat, seq, trump) { let p = pat === '' ? [] : pat.split(' '); const full = seq === '' ? [] : seq.split(' ');
  let s = full.slice(); if (p[0] !== 'P') while (s[0] === 'P') s.shift();
  if (p.length && (p[0] === '**' || p[0].startsWith('BW:'))) { if (p[0].startsWith('BW:') && trump !== p[0].slice(3)) return false;
    p = p.slice(1); if (s.length < p.length) return false; s = s.slice(s.length - p.length); }
  return p.length === s.length && p.every((t, k) => tokMatch(t, s[k])); }
for (const r of rules) r.fn = compile(r.cond);
function choose(hand, seq, trump, options, vul, oppVul, seat, ctx) { const e = feats(hand); e.vul = !!vul; e.opp_vul = !!oppVul; e.seat = seat || 1;
  // contexte de l'enchère (§ 5 de la spécification) : fit = longueur en main dans p_suit + p_len
  ctx = ctx || {}; e.lvl = ctx.lvl || 0; e.p_suit = ctx.p_suit || ''; e.p_len = ctx.p_len || 0; e.p_forcing = !!ctx.p_forcing; e.fit = e.p_suit ? e[e.p_suit] + e.p_len : 0;
  for (const r of rules) { if (r.option && !options.includes(r.option)) continue;
    const pats = Array.isArray(r.seq) ? r.seq : [r.seq]; if (!pats.some(p => match(p, seq, trump))) continue;
    if (r.fn(e)) return r; } return null; }
let bad = 0, badF = 0;
for (const c of tests.cases) { const h = parseHand(c.hand), e = feats(h);
  for (const k of ['S', 'H', 'D', 'C', 'hcp', 'hl', 'dh', 'hld', 'shape', 'balanced', 'semibalanced', 'aces', 'kings', 'losers', 'ptricks', 'qtricks', 'sidetricks'])
    if (e[k] !== c.features[k]) { badF++; if (badF < 5) console.log('feature', c.id, k, e[k], c.features[k]); }
  for (const f of ['ace', 'king', 'queen', 'top', 'solid', 'stop', 'short', 'hcp_in', 'keycards', 'ctrl1', 'ctrl2'])
    for (const s of SUITS) if (e[f](s) !== c.features[f][s]) { badF++; if (badF < 5) console.log('feature', c.id, f, s, e[f](s), c.features[f][s]); }
  const r = choose(h, c.seq, c.trump, c.options, c.vul, c.opp_vul, c.seat, c); const got = r ? r.id : null;
  if (got !== c.expected.rule) { bad++; if (bad < 8) console.log('écart', c.id, c.hand, JSON.stringify(c.seq), got, c.expected.rule); } }
console.log(`${tests.cases.length} cas : ${bad} écarts de règle, ${badF} écarts de caractéristiques`);
