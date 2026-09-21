package main

import (
	"strings"
	"testing"
)

// TestSlamExplorationOverGameRaise reproduces the deal where South
// (J32.KJ754.AKQ8.4, 14H / 17 HLD) passed North's direct 4H raise
// ("soutien à la manche, 20HLD et plus") with a combined minimum of 37 —
// far into the slam zone. Blackwood is barred (spades unguarded from South's
// side), but the control bids can run above game once the count carries a
// real cushion: South cues 5C (singleton), North 5D (void), South signs off
// 5H denying a spade control, North shows the spade ace with 5S, and South
// bids the small slam. Par is 13 tricks in hearts.
func TestSlamExplorationOverGameRaise(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AQ74.AT82.-.AKJT8 T9865..T7632.Q65 J32.KJ754.AKQ8.4 K.Q963.J954.9732"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	// South's second call (after the 1H response) must start the control bids.
	const south = 2
	n := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if n == 1 {
			if sc.Call.Kind == KindPass || !strings.Contains(sc.M.fr, "contrôle") {
				t.Fatalf("South over 4H = %s (%s), want a control bid toward slam\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "6C" && got != "7C" {
		t.Fatalf("final contract = %s, want a heart slam (par: 13 tricks)\nauction: %s",
			got, formatAuction(calls))
	}
}
