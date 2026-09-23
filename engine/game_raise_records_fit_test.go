package engine

import (
	"strings"
	"testing"
)

// TestGameRaiseRecordsTrumpLength reproduces the reported deal where, after a
// 2D opening, responder raised opener's hearts to game (4H) yet the generic
// "conclusion à la manche" recorded no trump length, so opener lost sight of
// the nine-card heart fit and fell back to a notrump slam try. Recording the
// real length lets opener see the fit and drive the (king-ask) slam in hearts
// instead: with all four kings and the trump queen between the hands, the cold
// grand slam at 7H is reached.
func TestGameRaiseRecordsTrumpLength(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("A", "AKQ862", "AK2", "A54"), // 24 HCP, six hearts, all four aces
		south: hand("K53", "J94", "J3", "KQT73"), // 10 HCP, three-card heart support
		east:  hand("QJ862", "7", "QT984", "62"),
		west:  hand("T974", "T53", "765", "J98"),
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
		{north, 1, "3C", "cinquième"},       // shows the long heart suit
		{south, 1, "4C", "manche"},          // raises to game in hearts (records the fit)
		{north, 2, "4SA", "appel aux Rois"}, // heart fit seen: king-ask in hearts, not notrump
		{south, 2, "5C", "clefs"},           // king-key answer
		{north, 3, "7C", "grand chelem"},    // all keys held: grand slam in hearts
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

	// Opener must play in the heart fit, not fall back to notrump.
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "7C" {
		t.Fatalf("final contract = %s, want 7C (grand slam in the heart fit)\nauction: %s",
			got, formatAuction(calls))
	}
}
