package main

import "testing"

// TestStrong2DSlamOnCombinedFloors checks that the 2D opener adds the two
// announced floors before rebidding: 25 H facing the 2NT ace answer, which
// guarantees 8, is 33 between the hands. The default 3NT rebid would bury the
// deal in a game the pair was already beyond, so the small slam is named.
func TestStrong2DSlamOnCombinedFloors(t *testing.T) {
	const north = 0
	// N = AJ82 KQ2 AKQ AQ7 (25H, no five-card suit), S = KT4 JT93 JT42 K4 (8H,
	// no ace, no short suit): 2D - 2NT - 6NT.
	const pbn = `[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:AJ82.KQ2.AKQ.AQ7 Q63.854.86.J9853 KT4.JT93.JT42.K4 975.A76.9753.T62"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		n++
		if n == 2 {
			if got := sc.Call.Format("fr"); got != "6SA" {
				t.Fatalf("second call of North = %s (%s), want 6SA\nauction: %s", got, sc.M.fr, formatAuction(calls))
			}
			return
		}
	}
	t.Fatalf("North made fewer than two calls\nauction: %s", formatAuction(calls))
}
