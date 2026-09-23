package engine

import (
	"strings"
	"testing"
)

// TestKingAskOverStrong2D checks the king-ask Blackwood introduced over a 2D
// opening: because the ace-step responses already pinpoint every ace, opener's
// 4NT asks for kings, the trump queen replacing the trump king as the fifth
// key (docs/bidings.md). Here a king-key is missing and the combined strength
// is short of the overwhelming zone, so the auction stops in the small slam
// rather than blasting a grand -- proving 4NT was read as a king ask (a plain
// keycard ask, with every keycard held, would have driven to seven).
func TestKingAskOverStrong2D(t *testing.T) {
	const north, south = 0, 2
	d := dealWithQuietOpponents(north, map[int]*Hand{
		north: hand("AQ3", "AK3", "K3", "AKT94"), // 23 HCP, all four aces
		south: hand("J72", "82", "AQJ762", "63"), // long diamonds, the trump queen, no king
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2K", "24HL"},               // strong 2D opening
		{south, 0, "3K", "As de Carreau"},      // ace-step: the ace of diamonds
		{north, 1, "4T", "cinquième"},          // natural rebid
		{south, 1, "4K", "l'atout"},            // shows the long diamond suit
		{north, 2, "4SA", "appel aux Rois"},    // 4NT is a king ask, not a keycard ask
		{south, 2, "5K", "clefs"},              // king-key answer
		{north, 3, "6K", "une clef manquante"}, // a king-key missing: stops in the small slam
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
	if got := contract.Format("fr"); got != "6K" {
		t.Fatalf("final contract = %s, want 6K (small slam, a king-key missing)\nauction: %s",
			got, formatAuction(calls))
	}
}
