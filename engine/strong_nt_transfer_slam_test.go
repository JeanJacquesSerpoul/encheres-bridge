package engine

import (
	"strings"
	"testing"
)

// TestStrongNTTransferSlam reproduces a reported deal where responder's
// five-card major Texas transfer over a strong 2SA opening (20-21H) got
// stuck at a flat 3NT ("l'ouvreur choisit avec 3 atouts") regardless of how
// many points responder actually held. The afterTransfer "oMin >= 20"
// branch treated every hand from 4HL up to 40HL identically; with North's
// 20 and South's 14 (34 combined), the auction must explore keycards
// instead of settling for game.
func TestStrongNTTransferSlam(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AKQJ8", "A43", "QJ4", "K4"),
		south: hand("T2", "KQT62", "A5", "AJ52"),
		east:  hand("9753", "75", "K987", "763"),
		west:  hand("64", "J98", "T632", "QT98"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "2SA", "20-21"},        // strong 2NT opening
		{south, 0, "3K", "Texas"},         // 3D, Texas for hearts
		{north, 1, "3C", "rectification"}, // 3H, completes the transfer
		{south, 1, "4SA", "Blackwood"},    // 4NT ace ask, not a flat 3NT
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
	if got := contract.Format("fr"); got != "6SA" {
		t.Fatalf("final contract = %s, want 6SA\nauction: %s", got, formatAuction(calls))
	}
}
