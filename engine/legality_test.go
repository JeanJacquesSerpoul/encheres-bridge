package engine

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestNoIllegalRuleCalls: over the par bench, no rule may answer with a call
// the auction does not allow. The engine would pass instead (a "default
// pass"), which hides the gap: a rule whose pattern admits a sequence where
// its call is insufficient, or a double of partner's bid, is a bug in
// cli/rules/default.yaml. `python tools/python_tools/sef_rules.py
// cli/rules/default.yaml --validate` points at the same rules statically.
func TestNoIllegalRuleCalls(t *testing.T) {
	loadTestRules(t)
	found := map[string]int{}
	for _, b := range loadBench(t) {
		d := mustParsePBN(t, fmt.Sprintf("[Dealer %q]\n[Vulnerable %q]\n[Deal %q]\n", b.Dealer, b.Vul, b.Deal))
		for _, sc := range NewEngine(d).Run() {
			if sc.Why == illegalBy {
				found[fmt.Sprintf("%s (%s) après %q", sc.Denied.ID, sc.Denied.CallText, strings.Join(sc.Seq, " "))]++
			}
		}
	}
	if len(found) == 0 {
		return
	}
	lines := make([]string, 0, len(found))
	for k, n := range found {
		lines = append(lines, fmt.Sprintf("%5d  %s", n, k))
	}
	sort.Strings(lines)
	t.Fatalf("%d règles donnent une enchère illégale :\n%s", len(found), strings.Join(lines, "\n"))
}
