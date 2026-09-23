package main

import (
	"strings"
	"testing"
)

// TestStrong2DBalancedRebidStayman reproduces a reported deal where North's
// strong 2D opening (game forcing, 24HL+) rebid a balanced 2NT, and South,
// holding four hearts, simply concluded 3NT -- missing the 4-4 heart fit
// (North holds AKQ2). Opener's default balanced rebid is treated like a 2NT
// opening: South must be able to look for the major fit through Stayman (3C),
// which here lands the far superior 4H game.
func TestStrong2DBalancedRebidStayman(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AQ85", "AKQ2", "KQ74", "A"), // 24 HCP, 4-4-4-1, opens 2D
		south: hand("J43", "J864", "63", "K832"), // 5 HCP, four hearts
		east:  hand("KT62", "97", "J95", "QJ74"),
		west:  hand("97", "T53", "AT82", "T965"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2K", "forcing de manche"}, // strong 2D opening
		{south, 0, "2C", "pas d'As"},          // ace-step: no ace, weak
		{north, 1, "2SA", "régulier"},         // default balanced rebid
		{south, 1, "3T", "Stayman"},           // Stayman, not a flat 3NT
		{north, 2, "3SA", "Cœur et 4"},        // both four-card majors
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
	if got := contract.Format("fr"); got != "4C" {
		t.Fatalf("final contract = %s, want 4C (heart game via the 4-4 fit)\nauction: %s", got, formatAuction(calls))
	}
}
