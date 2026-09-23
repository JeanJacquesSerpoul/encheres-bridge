package engine

import (
	"strings"
	"testing"
)

// TestQuantitative4NTAfterJump2NT checks that when the Checkback shape
// doesn't apply (no 5-card responder major, no 4-card other major), a
// balanced 14 HL responder may still try a quantitative 4SA over opener's
// jump 2NT rebid (18-19 HL) instead of settling short (docs/addon_4.md).
func TestQuantitative4NTAfterJump2NT(t *testing.T) {
	const north, south = 0, 2
	d := dealWith(south, map[int]*Hand{
		north: hand("A32", "KQJ9", "J43", "QJ4"),
		south: hand("K54", "32", "AKQ2", "AK32"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{south, 0, "1K", ""},             // 1D opening
		{north, 0, "1C", ""},             // 1H response
		{south, 1, "2SA", "Checkback"},   // jump 2NT rebid
		{north, 1, "4SA", "quantitatif"}, // quantitative try
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
}
