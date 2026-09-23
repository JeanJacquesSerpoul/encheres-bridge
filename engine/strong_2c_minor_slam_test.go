package main

import (
	"strings"
	"testing"
)

// TestStrong2CMinorTwoSuiterSlam reproduces the reported deal where North's
// strong 2C opening rebid a balanced 2NT and South held a slam-going 5-5 minor
// two-suiter (0-3-5-5, void in spades). The old relay structure gave South no
// way to show both minors over the 2NT rebid, so it buried the huge double fit
// under a flat 3NT. South must now transfer through the minor Texas (3S), name
// the second minor (4D) to confirm the 5-5 shape, and let the generic slam
// engine drive to the minor slam.
func TestStrong2CMinorTwoSuiterSlam(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("A", "KQ97", "AKQ4", "KQ86"), // 23 HCP, 1-4-4-4
		south: hand("", "A52", "J9872", "AT742"), // 0-3-5-5, both minors
		east:  hand("KQ765432", "T4", "", "J53"),
		west:  hand("JT98", "J863", "T653", "9"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2T", "fort indéterminé"},     // strong 2C opening
		{south, 0, "2K", "relais"},               // forced 2D relay
		{north, 1, "2SA", "sans couleur longue"}, // balanced rebid, no long suit
		{south, 1, "3P", "Texas Trèfle"},         // minor Texas (transfer via clubs)
		{north, 2, "4T", "rectification"},        // opener rectifies to clubs
		{south, 2, "4K", "bicolore 5T-5K"},       // reveals the 5-5 double fit
	}
	for _, w := range seq {
		n := 0
		found := false
		for _, sc := range calls {
			if sc.Seat != w.seat {
				continue
			}
			if n == w.nth {
				if got := sc.Call.Format("fr"); got != w.call {
					t.Fatalf("seat %s call #%d = %s (%s), want %s\nauction: %s",
						seatNames[w.seat], w.nth, got, sc.M.fr, w.call, formatAuction(calls))
				}
				if w.hint != "" && !strings.Contains(sc.M.fr, w.hint) {
					t.Fatalf("seat %s call #%d comment %q does not mention %q",
						seatNames[w.seat], w.nth, sc.M.fr, w.hint)
				}
				found = true
				break
			}
			n++
		}
		if !found {
			t.Fatalf("seat %s never made call #%d\nauction: %s", seatNames[w.seat], w.nth, formatAuction(calls))
		}
	}

	// South must no longer bury the two-suiter under a flat 3NT.
	for _, sc := range calls {
		if sc.Seat == south && sc.Call.Format("fr") == "3SA" {
			t.Fatalf("South concluded a flat 3SA, missing the minor slam\nauction: %s", formatAuction(calls))
		}
	}

	// The partnership must reach at least a small slam in a minor.
	contract, _, _ := finalContract(calls)
	if !(contract.Level >= 6 && (contract.Strain == SClubs || contract.Strain == SDiamonds)) {
		t.Fatalf("final contract = %s, want a minor slam (6C/6D or higher)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
