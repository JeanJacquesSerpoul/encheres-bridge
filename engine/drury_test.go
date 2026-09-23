package engine

import (
	"strings"
	"testing"
)

// Drury (docs/regles_moteur.md [RM-2b]): having passed, the responder facing a
// third- or fourth-seat major opening asks with 2C whether that opening holds
// its values, before the fit's raw count buys a level the opening cannot pay
// for. 2NT says the same with four trumps and a singleton.

// druryAuction runs a deal where North deals and passes, South opens a major
// in third seat, and North is the Drury hand, then returns the calls.
func druryAuction(t *testing.T, north, south *Hand) []SeatCall {
	t.Helper()
	d := dealWithQuietOpponents(0, map[int]*Hand{0: north, 2: south})
	return NewEngine(d).Run()
}

// nthNonPass returns seat's nth call, passes included, as text plus comment.
func nthCallText(t *testing.T, calls []SeatCall, seat, nth int) (string, string) {
	t.Helper()
	n := 0
	for _, sc := range calls {
		if sc.Seat != seat {
			continue
		}
		if n == nth {
			return sc.Call.Format("fr"), sc.M.fr
		}
		n++
	}
	t.Fatalf("seat %s made fewer than %d calls\nauction: %s", seatNames[seat], nth+1, formatAuction(calls))
	return "", ""
}

func TestDruryOpenerRebids(t *testing.T) {
	const north, south = 0, 2
	// 11 HLD, three-card spade support, and no five-card side suit to show:
	// the Drury hand.
	resp := hand("K72", "A65", "Q9843", "76")
	cases := []struct {
		name  string
		south *Hand
		want  string
		hint  string
	}{
		{"ouverture minimale", hand("AQJ63", "KJ2", "J54", "98"), "2P", "restons-en là"},
		{"ambition de manche", hand("AQJ63", "K72", "A5", "9832"), "2K", "Drury"},
		{"la manche, sans chelem", hand("AQJ63", "K942", "A5", "32"), "4P", "manche"},
		{"ambition de chelem", hand("AKQ63", "K942", "A5", "32"), "3P", "chelem"},
		{"beau bicolore", hand("AQJ63", "KQJ2", "A5", "93"), "2C", "bicolore"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := druryAuction(t, resp, tc.south)
			if got, comment := nthCallText(t, calls, north, 1); got != "2T" || !strings.Contains(comment, "Drury") {
				t.Fatalf("North's response = %s (%s), want 2T Drury\nauction: %s", got, comment, formatAuction(calls))
			}
			got, comment := nthCallText(t, calls, south, 1)
			if got != tc.want {
				t.Fatalf("South's rebid = %s (%s), want %s\nauction: %s", got, comment, tc.want, formatAuction(calls))
			}
			if !strings.Contains(comment, tc.hint) {
				t.Fatalf("comment %q does not mention %q", comment, tc.hint)
			}
		})
	}
}

// TestDruryMinimumStopsBelowGame: the whole point of the convention. South's
// minimum opening repeats the major, and the auction dies there rather than
// climbing to a game the pair's 24 combined points cannot make.
func TestDruryMinimumStopsBelowGame(t *testing.T) {
	calls := druryAuction(t, hand("K72", "A65", "Q9843", "76"), hand("AQJ63", "KJ2", "J54", "98"))
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "2P" {
		t.Fatalf("final contract = %s, want 2P\nauction: %s", got, formatAuction(calls))
	}
}

// TestDruryGameTryAnswers: over the artificial 2D, the responder shows how
// long the fit really is -- the 2C ask only promised three cards.
func TestDruryGameTryAnswers(t *testing.T) {
	const north, south = 0, 2
	opener := hand("AQJ63", "K72", "A5", "9832") // 16 HLD: the game try
	cases := []struct {
		name  string
		north *Hand
		want  string
	}{
		{"3 cartes", hand("K72", "A65", "Q9843", "76"), "2P"},
		{"4 cartes", hand("K872", "A65", "QJ84", "76"), "3P"},
		{"5 cartes", hand("K8763", "A65", "QJ8", "74"), "4P"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := druryAuction(t, tc.north, opener)
			if got, comment := nthCallText(t, calls, south, 1); got != "2K" {
				t.Fatalf("South's rebid = %s (%s), want the 2K game try\nauction: %s", got, comment, formatAuction(calls))
			}
			got, comment := nthCallText(t, calls, north, 2)
			if got != tc.want {
				t.Fatalf("North's answer = %s (%s), want %s\nauction: %s", got, comment, tc.want, formatAuction(calls))
			}
			if !strings.Contains(comment, "soutien") {
				t.Fatalf("comment %q does not describe the support length", comment)
			}
		})
	}
}

// TestDruryShortVariant: four trumps and a singleton answer 2NT; opener asks
// with 3C and the singleton is named -- clubs by the return to the opening
// suit, since 3C was the question itself.
func TestDruryShortVariant(t *testing.T) {
	const north, south = 0, 2
	strong := hand("AQJ63", "K942", "A5", "32")
	t.Run("2SA puis singleton Trefle", func(t *testing.T) {
		calls := druryAuction(t, hand("K872", "A65", "Q9843", "6"), strong)
		if got, comment := nthCallText(t, calls, north, 1); got != "2SA" || !strings.Contains(comment, "Drury") {
			t.Fatalf("North's response = %s (%s), want 2SA Drury\nauction: %s", got, comment, formatAuction(calls))
		}
		if got, comment := nthCallText(t, calls, south, 1); got != "3T" || !strings.Contains(comment, "singleton") {
			t.Fatalf("South's rebid = %s (%s), want the 3T singleton ask\nauction: %s", got, comment, formatAuction(calls))
		}
		got, comment := nthCallText(t, calls, north, 2)
		if got != "3P" {
			t.Fatalf("North's answer = %s (%s), want 3P (the return to the opening suit = club singleton)\nauction: %s",
				got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "Trèfle") {
			t.Fatalf("comment %q does not name the club singleton", comment)
		}
	})
	t.Run("minimum : arret au palier de 3", func(t *testing.T) {
		pbn := `[Dealer "N"]
[Deal "N:K872.A65.Q9843.6 T9.Q87.AK5.JT432 AQJ63.KJ2.J76.98 54.T943.T2.AKQ75"]`
		d, err := ParsePBN([]byte(pbn))
		if err != nil {
			t.Fatalf("bad deal: %v", err)
		}
		calls := NewEngine(d).Run()
		if got, comment := nthCallText(t, calls, south, 1); got != "3P" || !strings.Contains(comment, "minimale") {
			t.Fatalf("South's rebid = %s (%s), want the 3P sign-off\nauction: %s", got, comment, formatAuction(calls))
		}
		if contract, _, _ := finalContract(calls); contract.Format("fr") != "3P" {
			t.Fatalf("final contract = %s, want 3P\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})
}

// TestDruryOnlyForAPassedHand: the same North hand facing a first-seat
// opening has the ordinary ladder available and must not borrow the ask --
// 2C would be a club suit there.
func TestDruryOnlyForAPassedHand(t *testing.T) {
	const north, south = 0, 2
	d := dealWithQuietOpponents(south, map[int]*Hand{
		north: hand("K72", "A65", "Q9843", "76"),
		south: hand("AQJ63", "K72", "A5", "9832"),
	})
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "Drury") {
			t.Fatalf("Drury used by a hand that never passed\nauction: %s", formatAuction(calls))
		}
	}
}
