package engine

// The condition language of the rules (tools/python_tools/SEF_2024_spec.md §5):
// the restricted Python expressions sef_rules.py compiles with ast, evaluated
// with Python's semantics wherever a rule could tell the difference --
// chained comparisons, booleans counting as 0/1, `and`/`or` returning an
// operand, `in` over tuples (and substrings).

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------- values

type valueKind int

const (
	kBool valueKind = iota
	kInt
	kFloat
	kStr
	kTuple
	kFunc
)

type value struct {
	kind valueKind
	i    int64   // kBool (0/1) and kInt
	f    float64 // kFloat
	s    string  // kStr, and the name of a kFunc
	tup  []value // kTuple
}

func boolVal(b bool) value {
	if b {
		return value{kind: kBool, i: 1}
	}
	return value{kind: kBool}
}
func intVal(n int) value       { return value{kind: kInt, i: int64(n)} }
func floatVal(x float64) value { return value{kind: kFloat, f: x} }
func strVal(s string) value    { return value{kind: kStr, s: s} }
func (v value) numeric() bool  { return v.kind == kBool || v.kind == kInt || v.kind == kFloat }
func (v value) asFloat() float64 {
	if v.kind == kFloat {
		return v.f
	}
	return float64(v.i)
}

func (v value) truthy() bool {
	switch v.kind {
	case kBool, kInt:
		return v.i != 0
	case kFloat:
		return v.f != 0
	case kStr:
		return v.s != ""
	case kTuple:
		return len(v.tup) > 0
	}
	return true
}

func (k valueKind) String() string {
	return [...]string{"bool", "int", "float", "str", "tuple", "function"}[k]
}

// String renders the value as Python's repr would, which is also what the
// decision tree shows.
func (v value) String() string {
	switch v.kind {
	case kBool:
		if v.i != 0 {
			return "True"
		}
		return "False"
	case kInt:
		return strconv.FormatInt(v.i, 10)
	case kFloat:
		return pyFloat(v.f)
	case kStr:
		return "'" + v.s + "'"
	case kTuple:
		parts := make([]string, len(v.tup))
		for i, e := range v.tup {
			parts[i] = e.String()
		}
		if len(parts) == 1 {
			return "(" + parts[0] + ",)"
		}
		return "(" + strings.Join(parts, ", ") + ")"
	}
	return v.s
}

// pyFloat formats a float the way Python's repr does for the values rules
// meet (4.5, 2.0, 0.25).
func pyFloat(x float64) string {
	s := strconv.FormatFloat(x, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eEn") {
		s += ".0"
	}
	return s
}

func pyEqual(a, b value) bool {
	switch {
	case a.numeric() && b.numeric():
		if a.kind == kFloat || b.kind == kFloat {
			return a.asFloat() == b.asFloat()
		}
		return a.i == b.i
	case a.kind != b.kind:
		return false
	case a.kind == kStr || a.kind == kFunc:
		return a.s == b.s
	case a.kind == kTuple:
		if len(a.tup) != len(b.tup) {
			return false
		}
		for i := range a.tup {
			if !pyEqual(a.tup[i], b.tup[i]) {
				return false
			}
		}
		return true
	}
	return false
}

// pyCompare orders two values (-1, 0, 1), or fails as Python's < would.
func pyCompare(a, b value) (int, error) {
	switch {
	case a.numeric() && b.numeric():
		if a.kind == kFloat || b.kind == kFloat {
			x, y := a.asFloat(), b.asFloat()
			switch {
			case x < y:
				return -1, nil
			case x > y:
				return 1, nil
			}
			return 0, nil
		}
		switch {
		case a.i < b.i:
			return -1, nil
		case a.i > b.i:
			return 1, nil
		}
		return 0, nil
	case a.kind == kStr && b.kind == kStr:
		return strings.Compare(a.s, b.s), nil
	case a.kind == kTuple && b.kind == kTuple:
		for i := 0; i < len(a.tup) && i < len(b.tup); i++ {
			if !pyEqual(a.tup[i], b.tup[i]) {
				return pyCompare(a.tup[i], b.tup[i])
			}
		}
		return pyCompare(intVal(len(a.tup)), intVal(len(b.tup)))
	}
	return 0, fmt.Errorf("comparaison impossible entre %s et %s", a.kind, b.kind)
}

func pyContains(container, item value) (bool, error) {
	switch container.kind {
	case kTuple:
		for _, e := range container.tup {
			if pyEqual(e, item) {
				return true, nil
			}
		}
		return false, nil
	case kStr:
		if item.kind != kStr {
			return false, fmt.Errorf("'in <str>' attend une chaîne, pas %s", item.kind)
		}
		return strings.Contains(container.s, item.s), nil
	}
	return false, fmt.Errorf("'in' impossible sur %s", container.kind)
}

func pyArith(op string, a, b value) (value, error) {
	switch {
	case a.numeric() && b.numeric():
		if a.kind == kFloat || b.kind == kFloat {
			if op == "+" {
				return floatVal(a.asFloat() + b.asFloat()), nil
			}
			return floatVal(a.asFloat() - b.asFloat()), nil
		}
		if op == "+" {
			return value{kind: kInt, i: a.i + b.i}, nil
		}
		return value{kind: kInt, i: a.i - b.i}, nil
	case op == "+" && a.kind == kStr && b.kind == kStr:
		return strVal(a.s + b.s), nil
	case op == "+" && a.kind == kTuple && b.kind == kTuple:
		return value{kind: kTuple, tup: append(append([]value{}, a.tup...), b.tup...)}, nil
	}
	return value{}, fmt.Errorf("opération %s impossible entre %s et %s", op, a.kind, b.kind)
}

// ---------------------------------------------------------------- syntax tree

type nodeKind int

const (
	nConst nodeKind = iota
	nName
	nCall
	nTuple
	nNeg
	nNot
	nAnd
	nOr
	nArith   // kids[0] ops[i] kids[i+1] ...
	nCompare // kids[0] ops[i] kids[i+1] ... (chained)
)

type node struct {
	kind nodeKind
	src  string // the source text of this sub-expression
	val  value  // nConst
	name string // nName, nCall
	ops  []string
	kids []*node
}

// evalEnv is what an expression is evaluated against: one hand's features.
type evalEnv interface {
	lookup(name string) (value, bool)
	call(name string, args []value) (value, error)
}

func (n *node) eval(env evalEnv) (value, error) {
	switch n.kind {
	case nConst:
		return n.val, nil
	case nName:
		v, ok := env.lookup(n.name)
		if !ok {
			return value{}, fmt.Errorf("nom inconnu : %s", n.name)
		}
		return v, nil
	case nCall:
		args := make([]value, len(n.kids))
		for i, k := range n.kids {
			v, err := k.eval(env)
			if err != nil {
				return value{}, err
			}
			args[i] = v
		}
		return env.call(n.name, args)
	case nTuple:
		t := value{kind: kTuple, tup: make([]value, len(n.kids))}
		for i, k := range n.kids {
			v, err := k.eval(env)
			if err != nil {
				return value{}, err
			}
			t.tup[i] = v
		}
		return t, nil
	case nNeg:
		v, err := n.kids[0].eval(env)
		if err != nil {
			return value{}, err
		}
		switch v.kind {
		case kBool, kInt:
			return value{kind: kInt, i: -v.i}, nil
		case kFloat:
			return floatVal(-v.f), nil
		}
		return value{}, fmt.Errorf("moins unaire impossible sur %s", v.kind)
	case nNot:
		v, err := n.kids[0].eval(env)
		if err != nil {
			return value{}, err
		}
		return boolVal(!v.truthy()), nil
	case nAnd, nOr:
		var v value
		for _, k := range n.kids {
			var err error
			if v, err = k.eval(env); err != nil {
				return value{}, err
			}
			if v.truthy() == (n.kind == nOr) {
				return v, nil
			}
		}
		return v, nil
	case nArith:
		acc, err := n.kids[0].eval(env)
		if err != nil {
			return value{}, err
		}
		for i, op := range n.ops {
			b, err := n.kids[i+1].eval(env)
			if err != nil {
				return value{}, err
			}
			if acc, err = pyArith(op, acc, b); err != nil {
				return value{}, err
			}
		}
		return acc, nil
	case nCompare:
		left, err := n.kids[0].eval(env)
		if err != nil {
			return value{}, err
		}
		for i, op := range n.ops {
			right, err := n.kids[i+1].eval(env)
			if err != nil {
				return value{}, err
			}
			ok, err := compareOp(op, left, right)
			if err != nil {
				return value{}, err
			}
			if !ok {
				return boolVal(false), nil
			}
			left = right
		}
		return boolVal(true), nil
	}
	return value{}, fmt.Errorf("nœud inconnu")
}

func compareOp(op string, a, b value) (bool, error) {
	switch op {
	case "==":
		return pyEqual(a, b), nil
	case "!=":
		return !pyEqual(a, b), nil
	case "in":
		return pyContains(b, a)
	case "not in":
		ok, err := pyContains(b, a)
		return !ok, err
	}
	c, err := pyCompare(a, b)
	if err != nil {
		return false, err
	}
	switch op {
	case "<":
		return c < 0, nil
	case "<=":
		return c <= 0, nil
	case ">":
		return c > 0, nil
	}
	return c >= 0, nil // ">="
}

// ---------------------------------------------------------------- parser

type tokKind int

const (
	tEOF tokKind = iota
	tName
	tNum
	tStr
	tOp
)

type token struct {
	kind tokKind
	text string
	pos  int
	val  value
}

func tokenize(src string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
			j := i
			for j < len(src) && (src[j] == '_' || src[j] >= 'a' && src[j] <= 'z' || src[j] >= 'A' && src[j] <= 'Z' || src[j] >= '0' && src[j] <= '9') {
				j++
			}
			toks = append(toks, token{kind: tName, text: src[i:j], pos: i})
			i = j
		case c >= '0' && c <= '9' || c == '.' && i+1 < len(src) && src[i+1] >= '0' && src[i+1] <= '9':
			j := i
			isFloat := false
			for j < len(src) && (src[j] >= '0' && src[j] <= '9' || src[j] == '.') {
				if src[j] == '.' {
					if isFloat {
						return nil, fmt.Errorf("nombre invalide à la position %d", i)
					}
					isFloat = true
				}
				j++
			}
			t := token{kind: tNum, text: src[i:j], pos: i}
			if isFloat {
				x, err := strconv.ParseFloat(t.text, 64)
				if err != nil {
					return nil, fmt.Errorf("nombre invalide %q", t.text)
				}
				t.val = floatVal(x)
			} else {
				n, err := strconv.ParseInt(t.text, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("nombre invalide %q", t.text)
				}
				t.val = value{kind: kInt, i: n}
			}
			toks = append(toks, t)
			i = j
		case c == '\'' || c == '"':
			j := i + 1
			var sb strings.Builder
			for j < len(src) && src[j] != c {
				if src[j] == '\\' && j+1 < len(src) {
					j++
				}
				sb.WriteByte(src[j])
				j++
			}
			if j >= len(src) {
				return nil, fmt.Errorf("chaîne non terminée à la position %d", i)
			}
			toks = append(toks, token{kind: tStr, text: src[i : j+1], pos: i, val: strVal(sb.String())})
			i = j + 1
		default:
			two := ""
			if i+1 < len(src) {
				two = src[i : i+2]
			}
			switch {
			case two == "<=" || two == ">=" || two == "==" || two == "!=":
				toks = append(toks, token{kind: tOp, text: two, pos: i})
				i += 2
			case strings.IndexByte("<>+-(),", c) >= 0:
				toks = append(toks, token{kind: tOp, text: string(c), pos: i})
				i++
			default:
				return nil, fmt.Errorf("caractère interdit %q à la position %d", string(c), i)
			}
		}
	}
	return append(toks, token{kind: tEOF, pos: len(src)}), nil
}

// condNames are the names a condition may use (sef_rules.py NAMES).
var condNames = func() map[string]bool {
	m := map[string]bool{"max": true, "min": true}
	for _, n := range featureNames {
		m[n] = true
	}
	for _, n := range funcNames {
		m[n] = true
	}
	for _, n := range contextNames {
		m[n] = true
	}
	return m
}()

func isFuncName(n string) bool { _, ok := suitFuncs[n]; return ok || n == "max" || n == "min" }

var keywords = map[string]bool{"and": true, "or": true, "not": true, "in": true, "is": true,
	"if": true, "else": true, "lambda": true, "for": true, "None": true}

type parser struct {
	src  string
	toks []token
	p    int
}

// compileCond parses a rule condition, rejecting what sef_rules.py rejects.
func compileCond(src string) (*node, error) {
	toks, err := tokenize(src)
	if err != nil {
		return nil, err
	}
	ps := &parser{src: src, toks: toks}
	n, err := ps.expr()
	if err != nil {
		return nil, err
	}
	if t := ps.peek(); t.kind != tEOF {
		return nil, fmt.Errorf("syntaxe invalide près de %q", t.text)
	}
	return n, nil
}

func (ps *parser) peek() token { return ps.toks[ps.p] }
func (ps *parser) next() token { t := ps.toks[ps.p]; ps.p++; return t }

func (ps *parser) isOp(text string) bool {
	t := ps.peek()
	return t.kind == tOp && t.text == text
}

func (ps *parser) isKw(text string) bool {
	t := ps.peek()
	return t.kind == tName && t.text == text
}

// span sets n.src to the text from token start up to the current token.
func (ps *parser) span(n *node, start int) *node {
	end := len(ps.src)
	if ps.p > 0 {
		last := ps.toks[ps.p-1]
		end = last.pos + len(last.text)
	}
	n.src = strings.TrimSpace(ps.src[ps.toks[start].pos:end])
	return n
}

func (ps *parser) expr() (*node, error) { return ps.boolOp("or", nOr, ps.andTest) }

func (ps *parser) andTest() (*node, error) { return ps.boolOp("and", nAnd, ps.notTest) }

func (ps *parser) boolOp(kw string, kind nodeKind, sub func() (*node, error)) (*node, error) {
	start := ps.p
	first, err := sub()
	if err != nil {
		return nil, err
	}
	if !ps.isKw(kw) {
		return first, nil
	}
	n := &node{kind: kind, kids: []*node{first}}
	for ps.isKw(kw) {
		ps.next()
		k, err := sub()
		if err != nil {
			return nil, err
		}
		n.kids = append(n.kids, k)
	}
	return ps.span(n, start), nil
}

func (ps *parser) notTest() (*node, error) {
	start := ps.p
	if ps.isKw("not") {
		ps.next()
		k, err := ps.notTest()
		if err != nil {
			return nil, err
		}
		return ps.span(&node{kind: nNot, kids: []*node{k}}, start), nil
	}
	return ps.comparison()
}

func (ps *parser) compOp() (string, bool) {
	t := ps.peek()
	switch {
	case t.kind == tOp && (t.text == "<" || t.text == ">" || t.text == "<=" || t.text == ">=" || t.text == "==" || t.text == "!="):
		ps.next()
		return t.text, true
	case ps.isKw("in"):
		ps.next()
		return "in", true
	case ps.isKw("not") && ps.toks[ps.p+1].kind == tName && ps.toks[ps.p+1].text == "in":
		ps.p += 2
		return "not in", true
	}
	return "", false
}

func (ps *parser) comparison() (*node, error) {
	start := ps.p
	first, err := ps.arith()
	if err != nil {
		return nil, err
	}
	n := &node{kind: nCompare, kids: []*node{first}}
	for {
		op, ok := ps.compOp()
		if !ok {
			break
		}
		k, err := ps.arith()
		if err != nil {
			return nil, err
		}
		n.ops = append(n.ops, op)
		n.kids = append(n.kids, k)
	}
	if len(n.ops) == 0 {
		return first, nil
	}
	return ps.span(n, start), nil
}

func (ps *parser) arith() (*node, error) {
	start := ps.p
	first, err := ps.factor()
	if err != nil {
		return nil, err
	}
	n := &node{kind: nArith, kids: []*node{first}}
	for ps.isOp("+") || ps.isOp("-") {
		op := ps.next().text
		k, err := ps.factor()
		if err != nil {
			return nil, err
		}
		n.ops = append(n.ops, op)
		n.kids = append(n.kids, k)
	}
	if len(n.ops) == 0 {
		return first, nil
	}
	return ps.span(n, start), nil
}

func (ps *parser) factor() (*node, error) {
	start := ps.p
	if ps.isOp("-") {
		ps.next()
		k, err := ps.factor()
		if err != nil {
			return nil, err
		}
		return ps.span(&node{kind: nNeg, kids: []*node{k}}, start), nil
	}
	if ps.isOp("+") {
		return nil, fmt.Errorf("construction interdite : UAdd")
	}
	return ps.atom()
}

func (ps *parser) atom() (*node, error) {
	start := ps.p
	t := ps.next()
	switch t.kind {
	case tNum, tStr:
		return ps.span(&node{kind: nConst, val: t.val}, start), nil
	case tName:
		switch t.text {
		case "True", "true":
			return ps.span(&node{kind: nConst, val: boolVal(true)}, start), nil
		case "False", "false":
			return ps.span(&node{kind: nConst, val: boolVal(false)}, start), nil
		}
		if keywords[t.text] {
			return nil, fmt.Errorf("syntaxe invalide près de %q", t.text)
		}
		if !condNames[t.text] {
			return nil, fmt.Errorf("nom inconnu : %s", t.text)
		}
		if !ps.isOp("(") {
			return ps.span(&node{kind: nName, name: t.text}, start), nil
		}
		ps.next()
		n := &node{kind: nCall, name: t.text}
		for !ps.isOp(")") {
			k, err := ps.expr()
			if err != nil {
				return nil, err
			}
			n.kids = append(n.kids, k)
			if !ps.isOp(",") {
				break
			}
			ps.next()
		}
		if !ps.isOp(")") {
			return nil, fmt.Errorf("parenthèse fermante attendue après %s(", t.text)
		}
		ps.next()
		if !isFuncName(t.text) {
			return nil, fmt.Errorf("%s n'est pas une fonction", t.text)
		}
		return ps.span(n, start), nil
	case tOp:
		if t.text != "(" {
			break
		}
		if ps.isOp(")") { // () : the empty tuple
			ps.next()
			return ps.span(&node{kind: nTuple}, start), nil
		}
		first, err := ps.expr()
		if err != nil {
			return nil, err
		}
		if ps.isOp(")") {
			ps.next()
			return first, nil // parentheses only group
		}
		n := &node{kind: nTuple, kids: []*node{first}}
		for ps.isOp(",") {
			ps.next()
			if ps.isOp(")") {
				break
			}
			k, err := ps.expr()
			if err != nil {
				return nil, err
			}
			n.kids = append(n.kids, k)
		}
		if !ps.isOp(")") {
			return nil, fmt.Errorf("parenthèse fermante attendue")
		}
		ps.next()
		return ps.span(n, start), nil
	case tEOF:
		return nil, fmt.Errorf("expression incomplète")
	}
	return nil, fmt.Errorf("syntaxe invalide près de %q", t.text)
}

// ---------------------------------------------------------------- hand environment

func (f *features) lookup(name string) (value, bool) {
	if v, ok := f.scalar(name); ok {
		return v, true
	}
	if isFuncName(name) {
		return value{kind: kFunc, s: name}, true
	}
	return value{}, false
}

func (f *features) call(name string, args []value) (value, error) {
	if name == "max" || name == "min" {
		return pyMinMax(name, args)
	}
	fn, ok := suitFuncs[name]
	if !ok {
		return value{}, fmt.Errorf("%s n'est pas une fonction", name)
	}
	if len(args) != 1 {
		return value{}, fmt.Errorf("%s() attend 1 argument, %d reçu(s)", name, len(args))
	}
	if args[0].kind != kStr {
		return value{}, fmt.Errorf("%s() attend une couleur 'S', 'H', 'D' ou 'C'", name)
	}
	s, ok := suitByLetter(args[0].s)
	if !ok {
		return value{}, fmt.Errorf("%s() : couleur inconnue %s", name, args[0])
	}
	return fn(f, s), nil
}

func pyMinMax(name string, args []value) (value, error) {
	items := args
	if len(args) == 1 {
		if args[0].kind != kTuple {
			return value{}, fmt.Errorf("%s() : un seul argument doit être un tuple", name)
		}
		items = args[0].tup
	}
	if len(items) == 0 {
		return value{}, fmt.Errorf("%s() : aucune valeur", name)
	}
	best := items[0]
	for _, v := range items[1:] {
		c, err := pyCompare(v, best)
		if err != nil {
			return value{}, err
		}
		if (name == "max" && c > 0) || (name == "min" && c < 0) {
			best = v
		}
	}
	return best, nil
}

// leaves lists the hand facts a sub-expression reads -- the names and the
// calls on a literal suit -- for the decision tree, in reading order and
// without repeats.
func (n *node) leaves(out []*node, seen map[string]bool) []*node {
	switch n.kind {
	case nName:
		if !isFuncName(n.name) && !seen[n.src] {
			seen[n.src] = true
			out = append(out, n)
		}
		return out
	case nCall:
		if n.name != "max" && n.name != "min" {
			if !seen[n.src] {
				seen[n.src] = true
				out = append(out, n)
			}
			return out
		}
	}
	for _, k := range n.kids {
		out = k.leaves(out, seen)
	}
	return out
}
