package engine

import (
	"strings"
	"testing"
)

// TestSixMajorTexas2NTSlam reproduces a reported deal where responder held a
// six-card major and 14 HCP opposite a 20-21 2SA opening (34 combined, cold
// for slam), yet the afterTransfer "oMin >= 20" branch signed off in 4S the
// moment it saw six trumps -- the h.Len(M) >= 6 case fired before any point
// gate. A six-card suit is a reason to bid slam, never to stop in game, so a
// slam-zone hand must explore keycards instead of concluding at the four level.
//
//	South (opener) = J953.AQT.AKJ.AJ2 : 20 HCP, balanced, opens 2SA.
//	North (responder) = AKQ842.432.Q63.K : 14 HCP, six spades. Texas to spades
//	(3H), then must drive to slam rather than settle for 4S.
func TestSixMajorTexas2NTSlam(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(south, map[int]*Hand{
		north: hand("AKQ842", "432", "Q63", "K"),
		south: hand("J953", "AQT", "AKJ", "AJ2"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{south, 0, "2SA", "20-21"},        // strong 2NT opening
		{north, 0, "3C", "Texas"},         // 3H, Texas for spades
		{south, 1, "3P", "rectification"}, // 3S, completes the transfer
		{north, 1, "4SA", "Blackwood"},    // 4NT: slam try, NOT a flat 4S sign-off
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
					t.Fatalf("comment %q does not mention %q", sc.M.fr, w.hint)
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
	if got := contract.Format("fr"); got != "6SA" {
		t.Fatalf("final contract = %s, want 6SA (slam)\nauction: %s", got, formatAuction(calls))
	}
}
