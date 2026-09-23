package engine

import (
	"strings"
	"testing"
)

// TestControlBidTransitionsToBlackwood reproduces the reported deal where the
// control-bid exchange over a 2D opening signed off at 5D "contrôles
// insuffisants", missing an easy slam. With a minor trump the game level (5D)
// leaves almost no room: a control bid in clubs would climb to 5C, past 4NT,
// stranding the pair above Blackwood. Rather than cue past the ask, opener must
// launch Blackwood while 4NT is still available; the pair then reaches the
// small slam in diamonds instead of stopping in game.
func TestControlBidTransitionsToBlackwood(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AQ", "K4", "AQT64", "AKQJ"), // 25 HCP, all four aces, diamond suit
		south: hand("J87", "A86", "K72", "9642"), // 8 HCP, the heart ace and the trump king
		east:  hand("KT9532", "53", "953", "85"),
		west:  hand("64", "QJT972", "J8", "T73"),
	})

	calls := NewEngine(d).Run()

	// Opener must not cue 5C past the ask, and must not let the exchange fall
	// back on 5D with "plus de contrôle à montrer".
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if sc.Call == bid(5, SClubs) {
			t.Fatalf("North cued 5C past 4NT instead of asking Blackwood\nauction: %s", formatAuction(calls))
		}
	}
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "plus de contrôle à montrer") {
			t.Fatalf("the exchange stopped short of slam: %q\nauction: %s", sc.M.fr, formatAuction(calls))
		}
	}

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 1, "3K", "cinquième"},       // shows the diamond suit
		{south, 1, "4K", "soutien forcing"}, // raises: the trump before the controls
		{north, 2, "4SA", "appel aux Rois"}, // asks rather than cue 5C past 4NT
		{south, 2, "5K", "clefs"},           // king-key answer
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

	contract, _, _ := finalContract(calls)
	if !(contract.Level >= 6 && contract.Strain == SDiamonds) {
		t.Fatalf("final contract = %s, want at least a diamond small slam (6D)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
