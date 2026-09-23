package main

import (
	"strings"
	"testing"
)

// TestAceStepResponderDoesNotAsk checks who explores after a 2D opening. South
// has answered the ace step (2S: the ace of hearts or spades), so he has told
// North where his aces are and learns nothing by asking: his 4NT would draw a
// "1 or 4" whose ambiguity nothing in the auction can resolve, and the old
// engine read it as 1 and signed off in 5S with every keycard held.
//
// The exploration belongs to North, who knows his own hand and the ace the
// step named: holding all four aces between the hands, his 4NT is the king ask
// (docs/bidings.md), and the answer -- every king plus the trump queen -- is
// worth the grand slam. 7S makes on the count of top tricks.
//
// South raises straight to 4S rather than cueing on the way: with two spades
// and a fit nobody has named, [S-0] leaves him no control to bid. The round
// the old auction spent on those fitless cues bought nothing -- the king ask
// and the grand slam land exactly where they did before.
func TestAceStepResponderDoesNotAsk(t *testing.T) {
	const north, south = 0, 2
	const pbn = `[Dealer "W"]
[Vulnerable "None"]
[Deal "W:J6.T62.JT864.T94 AKQ53.KQ5.A.AK62 72.J43.Q7532.QJ8 T984.A987.K9.753"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat == south && sc.Call == bid(4, SNoTrump) {
			t.Fatalf("South asked keycards after answering the ace step\nauction: %s", formatAuction(calls))
		}
	}

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 2, "4SA", "appel aux Rois"},
		{south, 2, "5K", "clefs"},
		{north, 3, "7P", "grand chelem"},
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
					t.Fatalf("seat %s call #%d comment %q does not mention %q", seatNames[w.seat], w.nth, sc.M.fr, w.hint)
				}
				found = true
				break
			}
			n++
		}
		if !found {
			t.Fatalf("seat %s made fewer than %d calls\nauction: %s", seatNames[w.seat], w.nth+1, formatAuction(calls))
		}
	}
}
