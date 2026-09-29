package engine

// The decision tree of a call (the "trace" of the response, drawn by the page
// under each call): the sequence the pair saw, the rule that answered it, and
// its condition clause by clause, each evaluated on the hand.

import (
	"fmt"
	"strings"
)

// traceStep is one line of the tree, in both languages until the response
// picks one.
type traceStep struct {
	fr, en   string
	valueFR  string
	valueEN  string
	ok, note bool
	depth    int
}

// forcingText spells out the rules' forcing codes.
var forcingText = map[string][2]string{
	"NF":  {"non forcing", "non-forcing"},
	"F1":  {"forcing un tour", "forcing one round"},
	"FM":  {"forcing de manche", "game forcing"},
	"SO":  {"conclusion", "sign-off"},
	"INV": {"invitation", "invitational"},
	"REL": {"relais", "relay"},
	"ASK": {"question", "asking bid"},
	"TO":  {"contre d'appel", "takeout"},
	"PEN": {"punitif", "penalty"},
}

// maxTraceDepth bounds how far nested and/or clauses unfold.
const maxTraceDepth = 3

func traceCall(sc *SeatCall, f *features) []traceStep {
	seqFR, seqEN := "début d'enchère", "start of the auction"
	if len(sc.Seq) > 0 {
		seqFR, seqEN = formatSeq(sc.Seq, "fr"), formatSeq(sc.Seq, "en")
	}
	steps := []traceStep{{
		fr: "Séquence de la paire", en: "Pair's sequence",
		valueFR: seqFR, valueEN: seqEN, note: true,
	}}
	switch sc.Why {
	case noRule:
		return append(steps, traceStep{
			fr: "Aucune règle ne couvre cette séquence : passe par défaut",
			en: "No rule covers this sequence: default pass", note: true,
		})
	case illegalBy:
		r := sc.Denied
		return append(steps, traceStep{
			fr:      fmt.Sprintf("La règle %s donnerait %s, illégal : passe par défaut", r.ID, r.Call.Format("fr")),
			en:      fmt.Sprintf("Rule %s would give %s, which is illegal: default pass", r.ID, r.Call.Format("en")),
			valueFR: r.Meaning, valueEN: r.MeaningEN, note: true,
		})
	}
	r := sc.Rule
	fc := forcingText[r.Forcing]
	steps = append(steps, traceStep{
		fr: "Règle " + r.ID, en: "Rule " + r.ID,
		valueFR: fc[0], valueEN: fc[1], ok: true,
	})
	if c := r.cond; c.kind == nConst && c.val.truthy() {
		return append(steps, traceStep{fr: "Aucune condition sur la main", en: "No condition on the hand", ok: true, depth: 1})
	}
	return traceClauses(steps, r.cond, f, 1)
}

// traceClauses adds one step per clause of the condition's top-level and; a
// clause that is itself an and or an or gets its own step, its parts one
// level below.
func traceClauses(steps []traceStep, n *node, f *features, depth int) []traceStep {
	if n.kind == nAnd && depth == 1 {
		for _, k := range n.kids {
			steps = traceClauses(steps, k, f, depth)
		}
		return steps
	}
	steps = append(steps, clauseStep(n, f, depth))
	if (n.kind == nOr || n.kind == nAnd) && depth < maxTraceDepth {
		for _, k := range n.kids {
			steps = traceClauses(steps, k, f, depth+1)
		}
	}
	return steps
}

func clauseStep(n *node, f *features, depth int) traceStep {
	st := traceStep{fr: n.src, en: n.src, depth: depth}
	v, err := n.eval(f)
	st.ok = err == nil && v.truthy()
	var fr, en []string
	for _, l := range n.leaves(nil, map[string]bool{}) {
		lv, err := l.eval(f)
		if err != nil {
			continue
		}
		fr = append(fr, l.src+" = "+traceValue(lv, "fr"))
		en = append(en, l.src+" = "+traceValue(lv, "en"))
	}
	switch n.kind {
	case nName, nCall: // a lone fact: its value is the verdict
		fr, en = nil, nil
	case nAnd, nOr: // the parts below carry the values
		if depth < maxTraceDepth {
			fr, en = nil, nil
		}
	}
	st.valueFR, st.valueEN = strings.Join(fr, ", "), strings.Join(en, ", ")
	return st
}

func traceValue(v value, lang string) string {
	switch v.kind {
	case kBool:
		if lang == "fr" {
			return map[bool]string{true: "oui", false: "non"}[v.i != 0]
		}
		return map[bool]string{true: "yes", false: "no"}[v.i != 0]
	case kStr:
		return v.s
	}
	return v.String()
}
