package engine

import (
	"math/rand"
	"testing"
)

// Nord 1♣ (régulier 14 H), Sud 1♠, Nord 1SA (12-14) : la seconde enchère de
// Sud est la décision générale de la suite de l'enchère, sur la force
// combinée.
func TestConcludeTrace(t *testing.T) {
	north := hand("K84", "Q73", "KJ5", "AJ62")
	cases := []struct {
		name  string
		south *Hand
		want  string
		held  []string
	}{
		{"14 H → 3SA", hand("AQ95", "K82", "Q73", "K84"), "3SA",
			[]string{"minimum combiné suffisant pour une manche", "→ la manche : 3SA"}},
		{"11 H → 2SA", hand("AQ95", "K82", "T73", "Q84"), "2SA",
			[]string{"proposition de manche possible", "→ proposition de manche : 2SA"}},
		{"7 H → Passe", hand("AQ95", "J82", "962", "T84"), "Passe", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := playFrom(t, map[int]*Hand{0: north, 2: tc.south})
			for i, want := range []string{"1T", "Passe", "1P", "Passe", "1SA", "Passe"} {
				if got := calls[i].Call.Format("fr"); got != want {
					t.Fatalf("enchère %d = %s, attendu %s", i+1, got, want)
				}
			}
			assertCall(t, calls[6], 2, tc.want, tc.held...)
			if tc.want == "Passe" && !heldTest(calls[6].Trace, "Sans-Atout jouable") {
				t.Errorf("le test des arrêts devait tenir\n%v", calls[6].Trace)
			}
		})
	}
}

// TestTraceRewind: a rule that applied but made no call leaves nothing in the
// trace, not even a drop, before the next rule is tried.
func TestTraceRewind(t *testing.T) {
	tr := &tracer{}
	tr.check(true, "avant", "before", "")
	m := tr.mark()
	tr.in()
	tr.check(false, "règle", "rule", "")
	tr.drop()
	tr.rewind(m)
	if !tr.hasTests() || len(tr.steps) != 1 || tr.depth != 0 {
		t.Fatalf("rewind incomplet : %+v", tr)
	}
	var nilTr *tracer
	nilTr.rewind(nilTr.mark())
}

// TestTraceEndsOnConclusion: over random deals, no trace stops on a test that
// failed -- the call would then come with no explanation. The last line is
// always the test that held, or a note naming the outcome. Every step also
// carries its label in both languages.
func TestTraceEndsOnConclusion(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 1500; i++ {
		for _, sc := range NewEngine(randomDeal(rng)).Run() {
			if len(sc.Trace) == 0 {
				continue
			}
			for _, s := range sc.Trace {
				if s.fr == "" || s.en == "" {
					t.Fatalf("étape sans libellé dans les deux langues : %+v", s)
				}
			}
			if end := sc.Trace[len(sc.Trace)-1]; !end.ok && !end.note {
				t.Errorf("%s : la trace finit sur un test qui ne tient pas\n%v", sc.Call.Format("fr"), sc.Trace)
			}
		}
	}
}
