package main

import (
	"strings"
	"testing"
)

// TestSixMajorControlsOverNTRebid reproduces the deal where South, holding
// AQT543.KT5.AK5.9 (16H, six spades, singleton club — about 20 points once the
// fit is counted), must not blast 4P over North's 1SA rebid (12-14 balanced,
// hence at least a doubleton spade: an eight-card fit is certain). With a
// revalued minimum of 32 facing 12-14, the slam zone is in view and the hand
// must start the control bids; on this layout the machinery reaches 6P, the
// par contract.
func TestSixMajorControlsOverNTRebid(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:K9.9764.Q6.AKQ54 8.J83.JT943.J872 AQT543.KT5.AK5.9 J762.AQ2.872.T63"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	// South's second call (after the 1P response) must be a control bid, not 4P.
	const south = 2
	n := 0
	var second *SeatCall
	for i := range calls {
		if calls[i].Seat != south {
			continue
		}
		if n == 1 {
			second = &calls[i]
			break
		}
		n++
	}
	if second == nil {
		t.Fatalf("South never made a second call\nauction: %s", formatAuction(calls))
	}
	if second.Call.Format("fr") == "4P" || !strings.Contains(second.M.fr, "contrôle") {
		t.Fatalf("South rebid = %s (%s), want a control bid toward slam\nauction: %s",
			second.Call.Format("fr"), second.M.fr, formatAuction(calls))
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "6P" {
		t.Fatalf("final contract = %s, want 6P (par)\nauction: %s", got, formatAuction(calls))
	}
}
