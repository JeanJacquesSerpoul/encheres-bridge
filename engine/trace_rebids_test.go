package engine

import (
	"strings"
	"testing"
)

// rebidTrace plays a deal where North (dealer) opens, East passes with a
// blank hand, South answers, and returns North's second call and its trace.
func rebidTrace(t *testing.T, north, south *Hand) (calls []SeatCall, rebid SeatCall) {
	t.Helper()
	east := hand("9862", "T9", "T98", "KT43")
	for _, h := range []*Hand{north, south} {
		n := 0
		for s := Clubs; s <= Spades; s++ {
			n += h.Len(s)
		}
		if n != 13 {
			t.Fatalf("main de %d cartes : %v", n, h.Suits)
		}
	}
	calls = NewEngine(dealWith(0, map[int]*Hand{0: north, 1: east, 2: south})).Run()
	if len(calls) < 5 {
		t.Fatalf("séquence trop courte : %d enchères", len(calls))
	}
	if calls[1].Call.Kind != KindPass || calls[3].Call.Kind != KindPass {
		t.Fatalf("Est et Ouest devaient passer : %s, %s",
			calls[1].Call.Format("fr"), calls[3].Call.Format("fr"))
	}
	return calls, calls[4]
}

func TestRebidTrace(t *testing.T) {
	cases := []struct {
		name          string
		north, south  *Hand
		open, answer  string
		rebid         string
		held          []string // tests (French substrings) that must hold
		noteSubstring string   // a note that must appear, if any
	}{
		{"1♥-2♥ : minimum → Passe",
			hand("A53", "KQJ85", "Q73", "92"), hand("KJ7", "764", "AJ5", "8765"),
			"1C", "2C", "Passe",
			[]string{"le partenaire soutient la couleur d'ouverture"}, "→ Passe"},
		{"1SA-2♣ : 4 ♥ sans 4 ♠ → 2♥",
			hand("AQ5", "KJ42", "KQ7", "J92"), hand("KJ74", "Q76", "AJ5", "765"),
			"1SA", "2T", "2C",
			[]string{"4 ♥ sans 4 ♠ → 2♥"}, "Stayman"},
		{"1♣-1♥ : régulière 13 H → 1SA",
			hand("K53", "K84", "Q73", "AJ62"), hand("QJ7", "QJ76", "AJ54", "Q7"),
			"1T", "1C", "1SA",
			[]string{"régulière, 12-14 H", "1SA est encore possible"}, "nouvelle couleur"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls, rb := rebidTrace(t, tc.north, tc.south)
			if got := calls[0].Call.Format("fr"); got != tc.open {
				t.Fatalf("ouverture = %s, attendu %s", got, tc.open)
			}
			if got := calls[2].Call.Format("fr"); got != tc.answer {
				t.Fatalf("réponse = %s, attendu %s", got, tc.answer)
			}
			if got := rb.Call.Format("fr"); got != tc.rebid {
				t.Fatalf("redemande = %s, attendu %s\n%v", got, tc.rebid, rb.Trace)
			}
			if len(rb.Trace) == 0 {
				t.Fatal("la redemande n'a pas de trace")
			}
			for _, want := range tc.held {
				if !heldTest(rb.Trace, want) {
					t.Errorf("le test %q devait tenir\n%v", want, rb.Trace)
				}
			}
			if tc.noteSubstring != "" {
				found := false
				for _, s := range rb.Trace {
					if s.note && strings.Contains(s.fr, tc.noteSubstring) {
						found = true
					}
				}
				if !found {
					t.Errorf("note %q absente\n%v", tc.noteSubstring, rb.Trace)
				}
			}
			for _, s := range rb.Trace {
				if s.fr == "" || s.en == "" {
					t.Errorf("étape sans libellé dans les deux langues : %+v", s)
				}
			}
		})
	}
}

// TestUntracedDrop: a decision handed to code not yet instrumented carries no
// trace at all, rather than a path that stops before the real decision.
func TestUntracedDrop(t *testing.T) {
	e := &Engine{tr: &tracer{}}
	e.tr.check(true, "a", "b", "")
	c, _ := e.untraced(passCall, meaning{})
	if c.Kind != KindPass {
		t.Fatal("untraced doit rendre l'enchère telle quelle")
	}
	e.tr.check(true, "après", "after", "")
	e.tr.note("note", "note")
	if e.tr.hasTests() {
		t.Fatalf("la trace devait être vide après drop : %+v", e.tr.steps)
	}
}
