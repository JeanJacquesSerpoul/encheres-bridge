package main

import (
	"testing"
)

// TestOpenerDeclinesCompetitiveRaiseGame checks that opener does not strain to
// game over a competitive single raise when the combined count falls short.
// East opens 1H, South overcalls 2D, West raises to 2H (competitive, 11+ HLD
// but only three trumps), North passes. East holds a bare 14 HCP: combined with
// West's ~11 HLD floor that is under the 27 a major-suit game needs, so East
// must pass 2H rather than jump to 4H (which fails, making only eight tricks).
// The bug had East "accept the game try, maximum" because the +2-over-my-own-
// opening-minimum test cleared on a wide, unclarified 12-23 opening.
func TestOpenerDeclinesCompetitiveRaiseGame(t *testing.T) {
	// Donneur East. E = Q86.AKQ53.QJ3.52 (14H). W = K752.974.T6.AK76 (10H).
	pbn := `[Dealer "E"]
[Deal "E:Q86.AKQ53.QJ3.52 AT9..AK9752.QT94 K752.974.T6.AK76 J43.JT862.84.J83"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1 // seats: 0=N, 1=E, 2=S, 3=W
	// East's second call (over West's 2H raise) must not be a game bid.
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got == "4C" {
				t.Fatalf("East jumped to game %s (%s); combined values are short of game\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "2C" {
		t.Fatalf("final contract = %s, want 2C (heart partscore)\nauction: %s", got, formatAuction(calls))
	}
}
