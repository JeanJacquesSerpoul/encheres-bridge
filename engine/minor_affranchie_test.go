package main

import "testing"

// TestMinorAffranchieOpening checks the shape detection for the 3NT opening
// (docs/bidings.md, "L'OUVERTURE DE 3SA"): a solid seven-card minor (A, K, Q
// all present) with essentially nothing else in the hand -- no outside ace
// or king, at most one outside queen. The modern treatment (per the source
// document) drops the old requirement of an outside ace.
func TestMinorAffranchieOpening(t *testing.T) {
	cases := []struct {
		name string
		h    *Hand
		want string // "" = no 3NT opening
	}{
		{"AKQ + 4 petites à Trèfle, rien dehors", hand("32", "432", "5", "AKQ9432"), "3SA"},
		{"AKQ + 4 petites à Carreau, une Dame dehors", hand("Q3", "432", "AKQ9432", "5"), "3SA"},
		{"AKQ septième mais un Roi dehors : trop fort", hand("K3", "432", "5", "AKQ9432"), ""},
		{"AKQ septième mais un As dehors : trop fort", hand("A3", "432", "5", "AKQ9432"), ""},
		{"AKQ septième mais deux Dames dehors", hand("Q3", "Q32", "5", "AKQ9432"), ""},
		{"Six cartes seulement : pas affranchie", hand("32", "9432", "5", "AKQ432"), ""},
		{"Belle couleur mais sans la Dame : pas affranchie", hand("32", "432", "5", "AKJ9432"), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEngine(dealWith(0, map[int]*Hand{0: tc.h}))
			c, mn := e.opening(e.ps[0])
			if tc.want == "" {
				if got := c.Format("fr"); got == "3SA" {
					t.Fatalf("opening = 3SA (%s), want no 3NT opening", mn.fr)
				}
				return
			}
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("opening = %s (%s), want %s", got, mn.fr, tc.want)
			}
		})
	}
}

// TestMinorAffranchieCorrection replays the mechanism from the source
// document: responder, unsure which minor opener holds, anchors on clubs at
// his target level ("I want to play here, whichever minor it is"); opener
// passes if his suit is clubs, or corrects to diamonds at the very same
// level if it is diamonds. This case has opener holding diamonds, so the
// auction runs 3SA - P - 5C - P - 5D - P - P (responder passes the
// correction, the real suit is now known).
func TestMinorAffranchieCorrection(t *testing.T) {
	const north, south = 0, 2
	opener := hand("32", "432", "AKQ9432", "5")
	resp := hand("AKQ84", "AKQ75", "6", "87")
	d := dealWith(north, map[int]*Hand{north: opener, south: resp})
	calls := NewEngine(d).Run()

	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		n++
		switch n {
		case 1:
			if got := sc.Call.Format("fr"); got != "3SA" {
				t.Fatalf("opener's opening = %s, want 3SA\nauction: %s", got, formatAuction(calls))
			}
		case 2:
			if got := sc.Call.Format("fr"); got != "5K" {
				t.Fatalf("opener's correction = %s (%s), want 5K (real suit is diamonds)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
		}
	}

	final, _, _ := finalContract(calls)
	if final.Format("fr") != "5K" {
		t.Fatalf("final contract = %s, want 5K\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}

// TestMinorAffranchieNoCorrection checks the matching case: responder
// anchors on clubs and opener's real suit is clubs, so opener simply passes
// (no rectification needed).
func TestMinorAffranchieNoCorrection(t *testing.T) {
	const north, south = 0, 2
	opener := hand("32", "432", "5", "AKQ9432")
	resp := hand("AKQ84", "AKQ75", "6", "87")
	d := dealWith(north, map[int]*Hand{north: opener, south: resp})
	calls := NewEngine(d).Run()

	final, _, _ := finalContract(calls)
	if final.Format("fr") != "5T" {
		t.Fatalf("final contract = %s, want 5T\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}

// TestMinorAffranchiePassDefault checks that a plain responder, with nothing
// extra to add, simply passes the 3NT opening -- the overwhelmingly common
// case (docs: "Naturellement, le répondant peut passer en estimant qu'il
// s'agit du meilleur contrat").
func TestMinorAffranchiePassDefault(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	opener := hand("32", "432", "5", "AKQ9432")
	resp := hand("QJT9", "QJT", "QJT", "765")
	// East/West split the remaining cards without concentrating enough
	// strength to double or overcall (both well under the 18H "toutes
	// distributions" double, the only defensive call the engine can produce
	// once the auction is past the one-and two-level natural ranges).
	eastH := hand("A86", "K97", "K9742", "T8")
	westH := hand("K754", "A865", "A863", "J")
	d := dealWith(north, map[int]*Hand{north: opener, east: eastH, south: resp, west: westH})
	calls := NewEngine(d).Run()

	final, _, _ := finalContract(calls)
	if final.Format("fr") != "3SA" {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}
