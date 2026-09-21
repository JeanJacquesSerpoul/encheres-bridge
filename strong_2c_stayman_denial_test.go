package main

import (
	"strings"
	"testing"
)

// TestStrong2CStaymanDenial reproduces a reported deal where, after North's
// strong 2C opening rebid a balanced 2NT, South's Stayman (3C) went unanswered:
// North -- holding neither four hearts nor four spades (2-3-4-4) -- concluded a
// flat 3NT instead of denying the majors with 3D. The Stayman answer must be
// wired for the 2C opening's 2NT rebid exactly as for the 2D one; with the
// denial in place the quantitative machinery goes on to find the cold 6NT.
func TestStrong2CStaymanDenial(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("K7", "Q94", "AKQ8", "AKQT"), // 23 HCP, 2-3-4-4, no four-card major
		south: hand("AJ62", "K853", "JT4", "J6"), // 10 HCP, four hearts and four spades
		east:  hand("954", "A7", "9732", "8542"),
		west:  hand("QT83", "JT62", "65", "973"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2T", "fort indéterminé"}, // strong 2C opening
		{south, 0, "2K", "relais"},           // forced 2D relay
		{north, 1, "2SA", "22-23"},           // balanced rebid
		{south, 1, "3T", "Stayman"},          // Stayman, four-card major(s)
		{north, 2, "3K", "pas de majeure"},   // denies both majors, NOT 3NT
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

	// North's second call must be the Stayman denial, never a flat 3NT.
	if got, _ := southsCall(calls, north, 2); got == "3SA" {
		t.Fatalf("North answered the Stayman with a flat 3SA\nauction: %s", formatAuction(calls))
	}
}
