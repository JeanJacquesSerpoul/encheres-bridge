package main

import (
	"strings"
	"testing"
)

// TestTwoOverOneGameForcing reproduces a reported deal where South's
// two-over-one response (2H, 11HL+, forcing) over North's 1S opening got
// passed out at North's minimum 2SA rebid (12-14H) instead of continuing to
// game. A two-over-one response commits the side to game regardless of
// opener's rebid; it must not be passed below game. It also must not
// overshoot into a slam neither hand can justify: a forcing response
// deliberately leaves its own range wide open (11-40 here), and the
// quantitative-4NT machinery must not mistake that width for real slam
// values once South settles for 3NT (26 combined HCP total).
func TestTwoOverOneGameForcing(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AQT93", "A6", "T864", "A8"),
		south: hand("65", "KQT832", "Q73", "KQ"),
		east:  hand("K742", "J5", "KJ", "T7542"),
		west:  hand("J8", "974", "A952", "J963"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "1P", ""},                  // 1S opening
		{south, 0, "2C", "forcing de manche"}, // 2H, two-over-one, game forcing
		{north, 1, "2SA", "12-14"},            // minimum notrump rebid
		{south, 1, "3SA", "manche"},           // must not pass: bids game
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
	if got := contract.Format("fr"); got != "3SA" {
		t.Fatalf("final contract = %s, want 3SA (game, not overboard)\nauction: %s", got, formatAuction(calls))
	}
}
