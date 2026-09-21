package main

import (
	"strings"
	"testing"
)

// TestStrongAdvanceCueBidsInsteadOf1NT reproduces the screenshot deal: South,
// dealer, passes; West opens 1C; North overcalls 1D; East passes. South holds
// QJ75 A987 Q QT43 — 11H, 4-4-1-4, a club stopper but a singleton in partner's
// diamond suit and no five-card suit of its own. The engine used to advance
// with a plain 1NT, but the 1NT advance is limited to 8-10HL. Too strong for
// that, South must cue-bid the opener's suit (2C), a forcing probe that asks
// North to describe further.
func TestStrongAdvanceCueBidsInsteadOf1NT(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(south, map[int]*Hand{
		north: hand("AT4", "KJ2", "AJT92", "82"),
		east:  hand("K98", "T53", "86543", "96"),
		south: hand("QJ75", "A987", "Q", "QT43"),
		west:  hand("632", "Q64", "K7", "AKJ75"),
	})

	calls := NewEngine(d).Run()

	var southAdvance *SeatCall
	for i := range calls {
		if calls[i].Seat != south {
			continue
		}
		// South's first turn is the dealer pass; its advance is the next one.
		if calls[i].Call.Kind == KindPass {
			continue
		}
		southAdvance = &calls[i]
		break
	}
	if southAdvance == nil {
		t.Fatalf("South never made an advancing bid\nauction: %s", formatAuction(calls))
	}

	if southAdvance.Call == bid(1, SNoTrump) {
		t.Fatalf("South advanced with 1SA despite 11H (above the 8-10HL cap)\nauction: %s", formatAuction(calls))
	}
	if southAdvance.Call != bid(2, SClubs) {
		t.Fatalf("South's advance = %s, want 2T (cue-bid of the opener's suit)\nauction: %s",
			southAdvance.Call.Format("fr"), formatAuction(calls))
	}
	if !strings.Contains(southAdvance.M.fr, "cue-bid") {
		t.Fatalf("South's 2T comment %q does not mention the cue-bid", southAdvance.M.fr)
	}
}
