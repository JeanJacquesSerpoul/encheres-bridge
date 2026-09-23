package main

import (
	"strings"
	"testing"
)

// TestKingAskNotrumpOverStrong2D checks the no-trump branch of the king-ask
// over a 2D opening, on a deal with genuinely no eight-card fit. Opener's 4NT
// is the notrump ask; because the ace-step response already located every ace,
// it must read as a KING ask (a plain king count, no trump queen involved)
// rather than the old ace ask. With all four kings held between the hands the
// auction reaches the grand slam at 7NT.
func TestKingAskNotrumpOverStrong2D(t *testing.T) {
	const north, south = 0, 2
	d := dealWithQuietOpponents(north, map[int]*Hand{
		north: hand("AK543", "AK", "AK2", "A76"), // 25 HCP, all four aces, three kings
		south: hand("J2", "Q65", "Q543", "K42"),  // 8 HCP, no ace, the fourth king
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2K", "24HL"},            // strong 2D opening
		{south, 0, "2SA", "pas d'As"},       // ace-step: no ace
		{north, 2, "4SA", "appel aux Rois"}, // notrump 4NT is a king ask, not an ace ask
		{south, 2, "5K", "Roi"},             // king count (here 1 king), never "As"
		{north, 3, "7SA", "grand chelem"},   // all four kings held: bids the grand slam
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

	// The stale "demande des As" wording must be gone on this side.
	for _, sc := range calls {
		if sc.Seat == north && sc.Call == bid(4, SNoTrump) && strings.Contains(sc.M.fr, "demande des As") {
			t.Fatalf("North's 4NT still reads as an ace ask after a 2D opening\nauction: %s", formatAuction(calls))
		}
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "7SA" {
		t.Fatalf("final contract = %s, want 7SA (grand slam, all kings held)\nauction: %s",
			got, formatAuction(calls))
	}
}
