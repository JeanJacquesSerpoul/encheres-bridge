package main

import (
	"strings"
	"testing"
)

// TestGrandSlamViaControlsOverStrong2D reproduces a deal where the ace-step
// convention over a strong 2D opening (docs/bidings.md) leaves responder's
// monster 6-card trump suit undescribed. Responder naturally shows the long
// suit, the pair agrees diamonds as trump, and opener asks for slam. Because
// the ace-step response has already located every ace, opener's 4NT is a KING
// ask (the trump queen standing in for the trump king as the fifth key); the
// answer plus overwhelming combined strength carry the auction to the laydown
// grand slam in diamonds.
func TestGrandSlamViaControlsOverStrong2D(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AQ3", "AK3", "K3", "AKT94"),
		east:  hand("KT94", "7652", "94", "J82"),
		south: hand("J72", "Q8", "AQJ762", "63"),
		west:  hand("865", "JT94", "T85", "Q75"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2K", "24HL"},            // strong 2D opening, game forcing
		{south, 0, "3K", "As de Carreau"},   // ace-step response: the ace of diamonds
		{north, 1, "4T", "cinquième"},       // natural rebid, opener's own long suit
		{south, 1, "4K", "l'atout"},         // responder finally shows the long, strong suit
		{north, 2, "4SA", "appel aux Rois"}, // 4NT is a king ask (aces already located)
		{south, 2, "5K", "clefs"},           // king-key answer (Rois + Dame d'atout)
		{north, 3, "7K", "grand chelem"},    // king answer + power: bids the grand slam
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
