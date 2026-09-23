package engine

import (
	"strings"
	"testing"
)

// TestQuantitative4NTOverPlain1NTRebid reproduces the reported deal: opener
// rebids a plain 1SA (12-14 balanced) and responder holds enough (20 HCP)
// that the combined count could be anywhere from 32 to 34 -- below the
// direct 6SA threshold (cMin >= 33) but above it opposite the top of
// opener's range. Responder should try a quantitative 4SA rather than settle
// for 3SA outright (docs/addon_4.md, "4SA quantitatif").
func TestQuantitative4NTOverPlain1NTRebid(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("Q852", "932", "QJ", "9872"),
		east:  hand("K974", "K85", "T6", "AK64"),
		south: hand("T6", "T64", "97532", "QT3"),
		west:  hand("AJ3", "AQJ7", "AK84", "J5"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{east, 0, "1T", ""},             // 1C opening
		{west, 0, "1C", ""},             // 1H response
		{east, 1, "1SA", "12-14"},       // 1NT rebid, 12-14 balanced
		{west, 1, "4SA", "quantitatif"}, // quantitative slam try, not a flat 3SA
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
