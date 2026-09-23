package engine

import "testing"

// playFrom plays a deal dealt by North with the given hands and returns the
// calls; seats not given get the remaining cards (deterministically).
func playFrom(t *testing.T, given map[int]*Hand) []SeatCall {
	t.Helper()
	for seat, h := range given {
		n := 0
		for s := Clubs; s <= Spades; s++ {
			n += h.Len(s)
		}
		if n != 13 {
			t.Fatalf("siège %d : main de %d cartes", seat, n)
		}
	}
	return NewEngine(dealWith(0, given)).Run()
}

func assertCall(t *testing.T, sc SeatCall, seat int, want string, held ...string) {
	t.Helper()
	if sc.Seat != seat {
		t.Fatalf("enchère du siège %d attendue, siège %d trouvé", seat, sc.Seat)
	}
	if got := sc.Call.Format("fr"); got != want {
		t.Fatalf("siège %d : %s, attendu %s\n%v", seat, got, want, sc.Trace)
	}
	if len(sc.Trace) == 0 {
		t.Fatalf("siège %d : pas de trace", seat)
	}
	for _, w := range held {
		if !heldTest(sc.Trace, w) {
			t.Errorf("siège %d : le test %q devait tenir\n%v", seat, w, sc.Trace)
		}
	}
	for _, s := range sc.Trace {
		if s.fr == "" || s.en == "" {
			t.Errorf("étape sans libellé dans les deux langues : %+v", s)
		}
	}
}

// Nord 1♥, Est intervient 1♠ (belle cinquième, 10 H), Sud passe, Ouest
// soutient avec 3 atouts et 8 HLD.
func TestOvercallAndAdvanceTrace(t *testing.T) {
	calls := playFrom(t, map[int]*Hand{
		0: hand("A53", "KQJ85", "Q73", "92"),
		1: hand("KQJ96", "T4", "K84", "J63"),
		2: hand("72", "A73", "T96", "AQT84"),
		3: hand("T84", "962", "AJ52", "K75"),
	})
	if got := calls[0].Call.Format("fr"); got != "1C" {
		t.Fatalf("ouverture = %s, attendu 1C", got)
	}
	assertCall(t, calls[1], 1, "1P", "belle couleur de 5 cartes", "palier de 1 : 8 H et 9 HL")
	w := calls[3]
	if w.Call.Kind == KindPass || w.Call.Strain != SSpades {
		t.Fatalf("Ouest devait soutenir à Pique, il dit %s\n%v", w.Call.Format("fr"), w.Trace)
	}
	assertCall(t, w, 3, w.Call.Format("fr"), "3 atouts et plus", "7-10 HLD → soutien simple")
}

// Nord 1♣, Est contre (4-4-4-1, 15 H), Sud passe, Ouest répond 1SA :
// 10 H, régulier, arrêt à Trèfle.
func TestTakeoutDoubleAnswerTrace(t *testing.T) {
	calls := playFrom(t, map[int]*Hand{
		0: hand("A53", "K84", "Q7", "AJ962"),
		1: hand("KQ72", "AQ73", "KJ85", "3"),
		2: hand("JT9", "T92", "T964", "875"),
	})
	if got := calls[0].Call.Format("fr"); got != "1T" {
		t.Fatalf("ouverture = %s, attendu 1T", got)
	}
	assertCall(t, calls[1], 1, "Contre", "forme du contre")
	if calls[2].Call.Kind != KindPass {
		t.Fatalf("Sud devait passer, il dit %s", calls[2].Call.Format("fr"))
	}
	assertCall(t, calls[3], 3, "1SA", "8-10 H → réponse positive", "arrêt dans leur couleur")
}
