package main

import (
	"strings"
	"testing"
)

// TestCheapestControlOnHeartFit is the reported deal. North opens 1D, South
// answers 1H and North's jump raise to 3H opens the slam exploration. South
// holds QJ8 in spades -- no spade control -- so the principle of the controls
// gives him 4C, the cheapest control he actually holds: "S doit dire le
// contrôle le plus économique". The engine used to read 3S there as a relay
// asking for the spade control, which inverts the convention -- 3S on a heart
// fit says "I hold the spade control" [S-2b]. Nothing is lost by stepping over
// it: North is void in spades, and South's 4C tells him the control is not
// opposite.
func TestCheapestControlOnHeartFit(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "None"]
[Deal "N:.KT752.AKQT85.84 K954.Q43.742.KT7 QJ8.AJ96.96.AQJ3 AT7632.8.J3.9652"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	cue, ok := nthCallBy(calls, seatS, 1)
	if !ok || cue.Call.Format("fr") != "4T" {
		t.Fatalf("South's rebid = %s, want 4T (le contrôle le plus économique qu'il détient)\nauction: %s",
			cue.Call.Format("fr"), formatAuction(calls))
	}
	if !strings.Contains(cue.M.fr, "l'As") {
		t.Fatalf("comment %q does not show the club ace", cue.M.fr)
	}
	if !cue.M.deniedCtrl[Spades] {
		t.Fatalf("4C steps over 3S and must deny the spade control\nauction: %s", formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "6C" {
		t.Fatalf("final contract = %s, want 6C\nauction: %s", got, formatAuction(calls))
	}
}
