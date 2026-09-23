package engine

import (
	"strings"
	"testing"
)

// responseTrace plays a deal where North (dealer) opens, East passes with a
// blank hand and South answers, and returns South's first call and trace.
func responseTrace(t *testing.T, north, south *Hand) (SeatCall, []traceStep) {
	t.Helper()
	for _, h := range []*Hand{north, south} {
		n := 0
		for s := Clubs; s <= Spades; s++ {
			n += h.Len(s)
		}
		if n != 13 {
			t.Fatalf("main de %d cartes : %v", n, h.Suits)
		}
	}
	east := hand("9862", "T9", "T98", "KT43")
	calls := NewEngine(dealWith(0, map[int]*Hand{0: north, 1: east, 2: south})).Run()
	if calls[1].Call.Kind != KindPass {
		t.Fatalf("Est devait passer, il dit %s", calls[1].Call.Format("fr"))
	}
	for _, sc := range calls {
		if sc.Seat == 2 {
			return sc, sc.Trace
		}
	}
	t.Fatal("Sud n'a pas parlé")
	return SeatCall{}, nil
}

// heldTest reports whether a test containing want (French label) held.
func heldTest(steps []traceStep, want string) bool {
	for _, s := range steps {
		if !s.note && s.ok && strings.Contains(s.fr, want) {
			return true
		}
	}
	return false
}

func TestResponseTrace(t *testing.T) {
	open1H := hand("A53", "KQJ85", "Q73", "92")  // 12 H, 5♥ → 1♥
	open1C := hand("A53", "K84", "Q7", "AJ962")  // 14 H, 5♣ → 1♣
	open1NT := hand("AQ5", "KJ4", "KQ73", "J92") // 16 H régulier → 1SA
	cases := []struct {
		name   string
		north  *Hand
		open   string
		south  *Hand
		answer string
		held   []string // tests (French substrings) that must hold
	}{
		{"1♥ : 3 atouts, 9 HLD → soutien simple", open1H, "1C",
			hand("KJ7", "764", "AJ5", "8765"), "2C",
			[]string{"fit ♥", "6-10 HLD → soutien simple"}},
		{"1♥ : 4 ♠, 2 ♥ → 1♠", open1H, "1C",
			hand("KJ74", "76", "AJ5", "8765"), "1P",
			[]string{"sur 1♥ : 4 ♠"}},
		{"1♣ : 4 ♥ → 1♥", open1C, "1T",
			hand("KJ7", "QJ76", "AJ54", "Q7"), "1C",
			[]string{"une majeure quatrième"}},
		{"1SA : 5 ♠ → Texas", open1NT, "1SA",
			hand("KJT74", "876", "J5", "765"), "2C",
			[]string{"une majeure cinquième"}},
		{"1SA : 13 H régulier sans majeure → 3SA", open1NT, "1SA",
			hand("KJ7", "Q76", "AJ5", "Q765"), "3SA",
			[]string{"10-15 H → 3SA"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sc, steps := responseTrace(t, tc.north, tc.south)
			if got := sc.Call.Format("fr"); got != tc.answer {
				t.Fatalf("réponse = %s, attendu %s\n%v", got, tc.answer, steps)
			}
			if len(steps) == 0 {
				t.Fatal("la réponse n'a pas de trace")
			}
			for _, want := range tc.held {
				if !heldTest(steps, want) {
					t.Errorf("le test %q devait tenir\n%v", want, steps)
				}
			}
			for _, s := range steps {
				if s.fr == "" || s.en == "" {
					t.Errorf("étape sans libellé dans les deux langues : %+v", s)
				}
				if s.depth < 0 || s.depth > 3 {
					t.Errorf("profondeur incohérente : %+v", s)
				}
			}
		})
	}
	// L'ouverture elle-même, pour chaque cas, est bien celle attendue.
	for _, tc := range cases {
		calls := NewEngine(dealWith(0, map[int]*Hand{0: tc.north, 1: hand("9862", "T9", "T98", "KT43"), 2: tc.south})).Run()
		if got := calls[0].Call.Format("fr"); got != tc.open {
			t.Errorf("%s : ouverture = %s, attendu %s", tc.name, got, tc.open)
		}
	}
}
