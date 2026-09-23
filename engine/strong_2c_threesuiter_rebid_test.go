package main

import (
	"strings"
	"testing"
)

// TestStrong2CThreeSuiterRebid reproduces a reported deal where North's strong
// 2C opening, holding a 4-4-4-1 three-suiter (23 HCP, no suit longer than four
// cards), rebid 3H over the 2D relay -- a call that promises a good six-card
// suit. A three-suiter has no long suit to name: opener must rebid 2NT on
// strength rather than lie about a six-card major.
func TestStrong2CThreeSuiterRebid(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("A", "KQ97", "AKQ4", "KQ86"), // 23 HCP, 1-4-4-4
		south: hand("", "A52", "J9872", "AT742"), // 0-3-5-5
		east:  hand("KQ765432", "T4", "", "J53"),
		west:  hand("JT98", "J863", "T653", "9"),
	})

	calls := NewEngine(d).Run()

	// North must not claim a long major with a 4-4-4-1: no 3H rebid.
	for _, sc := range calls {
		if sc.Seat == north && sc.Call.Format("fr") == "3C" {
			t.Fatalf("North rebid 3C (3H) with a 4-4-4-1, falsely promising six cards\nauction: %s",
				formatAuction(calls))
		}
	}

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2T", "fort indéterminé"},     // strong 2C opening
		{south, 0, "2K", "relais"},               // forced 2D relay
		{north, 1, "2SA", "sans couleur longue"}, // three-suiter: 2NT, not a phantom suit
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
}
