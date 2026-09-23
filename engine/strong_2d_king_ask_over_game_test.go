package engine

import (
	"strings"
	"testing"
)

// TestStrong2DKingAskOverGameConclusion reproduces a reported deal where the
// strong 2D opening found its 4-4 heart fit through Stayman and then buried
// it in game. North holds A.AQJ8.AKQT.AKQ7 -- 29 H and all four aces -- and
// South, forced to answer whatever his count, showed none (2H) before raising
// the 3H fit to 4H on his ace-less 3 H.
//
// From there the point count alone can never move: South's floor is 0, so the
// combined minimum stops at 31 HLD, and the control road [S-1] has its ceiling
// at the trump game the auction already sits on. What breaks the deadlock is
// the information the ace step gave for free: every ace is accounted for, so
// opener's 4SA is the king ask [S-10], the trump queen standing in as the
// fifth key [S-11]. South's 5D shows one key (the heart king), which leaves
// one of the five missing and puts the pair in 6H instead of 4H.
func TestStrong2DKingAskOverGameConclusion(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "None"]
[Deal "S:T53.KT954.J8.862 J8642.6.974.T954 A.AQJ8.AKQT.AKQ7 KQ97.732.6532.J3"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north, south = 0, 2
	// South is the dealer: his pass in front of the 2D opening is call #0.
	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2K", "forcing de manche"},
		{south, 1, "2C", "pas d'As"},
		{north, 1, "2SA", "régulier"},
		{south, 2, "3T", "Stayman"},
		{north, 2, "3C", "4 cartes à Cœur"},
		{south, 3, "4C", "fit majeur"},
		{north, 3, "4SA", "appel aux Rois"},
		{south, 4, "5K", "clefs"},
		{north, 4, "6C", "petit chelem"},
	}
	for _, w := range seq {
		n, found := 0, false
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
