package engine

import "fmt"

// The decision trace: the path the engine followed to reach a call, test by
// test, evaluated on the hand — what the quiz shows under a wrong answer
// (« Voir l'arbre de décision »).
//
// Decisions stay ordinary Go conditions. An instrumented one reads
//
//	if e.tr.check(hl >= 24, "24 HL et plus → 2♦", "24+ HL → 2♦", pts(hl, "HL")) {
//
// check records the test and hands its boolean straight back, so the trace
// can never change a decision. A nil tracer records nothing: the same code
// runs traced from Run and untraced from everywhere else.
//
// Situations not yet instrumented leave the trace empty, and the page then
// offers no link for them.

type traceStep struct {
	fr, en string // the test, in both languages
	value  string // what the hand measured, e.g. "14 HL" (language-neutral)
	ok     bool   // whether the test held
	note   bool   // informative line rather than a test
	depth  int    // nesting, for the sub-tests of a branch
}

type tracer struct {
	steps   []traceStep
	depth   int
	dropped bool // the decision was handed to untraced code: see drop
}

// check records a test and returns its outcome unchanged.
func (t *tracer) check(ok bool, fr, en, value string) bool {
	if t != nil && !t.dropped {
		t.steps = append(t.steps, traceStep{fr: fr, en: en, value: value, ok: ok, depth: t.depth})
	}
	return ok
}

// note records an informative line: a safety net that replaced the call, a
// call planned on an earlier turn.
func (t *tracer) note(fr, en string) {
	if t != nil && !t.dropped {
		t.steps = append(t.steps, traceStep{fr: fr, en: en, note: true, depth: t.depth})
	}
}

// drop discards the trace: the decision was finally taken by code that is not
// instrumented yet, and a path stopping before the real decision would mislead
// more than it explains. With no test left, the page shows no link.
func (t *tracer) drop() {
	if t != nil {
		t.steps, t.dropped = nil, true
	}
}

// untraced passes a decision through unchanged and drops the trace — for the
// calls handed to code not yet instrumented: return e.untraced(e.conclude(p)).
func (e *Engine) untraced(c Call, mn meaning) (Call, meaning) {
	e.tr.drop()
	return c, mn
}

// in and out nest the tests of a branch under the test that opened it.
func (t *tracer) in() {
	if t != nil {
		t.depth++
	}
}

func (t *tracer) out() {
	if t != nil && t.depth > 0 {
		t.depth--
	}
}

// hasTests reports whether at least one real test was recorded: a trace made
// only of notes explains nothing, and the page shows no link for it.
func (t *tracer) hasTests() bool {
	if t == nil {
		return false
	}
	for _, s := range t.steps {
		if !s.note {
			return true
		}
	}
	return false
}

// ---------- formatting the measured values ----------

var suitSymbol = [4]string{"♣", "♦", "♥", "♠"}

// pts formats a point count: "14 HL", "13 H".
func pts(n int, unit string) string { return fmt.Sprintf("%d %s", n, unit) }

// cards formats a suit length: "5 ♠".
func cards(h *Hand, s Suit) string { return fmt.Sprintf("%d %s", h.Len(s), suitSymbol[s]) }

// shape formats the hand pattern in ♠-♥-♦-♣ order: "4-3-4-2".
func shape(h *Hand) string {
	return fmt.Sprintf("%d-%d-%d-%d", h.Len(Spades), h.Len(Hearts), h.Len(Diamonds), h.Len(Clubs))
}
