package main

import (
	"strings"
	"testing"
)

// TestControlBidAllControlsDrivesSlam checks that a strong hand does not fall
// back on the game with "plus de contrôle à montrer" when the cue-bid exchange
// has already shown a control in every side suit. East opens 1H, South makes a strong takeout
// double (18H), and after North's minimum answer the pair cue-bids: North shows
// the club king and a heart singleton, South a diamond singleton. With controls
// in clubs, diamonds and hearts and AKQ of the trump spades, all four suits are
// controlled, so South (21 HLD) must ask for keycards and bid the cold 6S rather
// than pass out in 4S. The bug had the low takeout-double floor (0-7H) keep the
// combined count under the Blackwood gate.
func TestControlBidAllControlsDrivesSlam(t *testing.T) {
	// Donneur East. S = AKQ95.KT5.T.AQ53 (18H, singleton diamond).
	// N = 8763.6.A952.KT92 (7H, singleton heart, the diamond ace and club king).
	pbn := `[Dealer "E"]
[Deal "E:4.AJ98432.KQ8.84 AKQ95.KT5.T.AQ53 JT2.Q7.J7643.J76 8763.6.A952.KT92"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	// The exchange must not die on a "plus de contrôle à montrer" retreat.
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "plus de contrôle à montrer") {
			t.Fatalf("signed off short of slam with all controls present: %q\nauction: %s",
				sc.M.fr, formatAuction(calls))
		}
	}

	contract, _, _ := finalContract(calls)
	if !(contract.Level >= 6 && contract.Strain == SSpades) {
		t.Fatalf("final contract = %s, want a spade small slam (6P)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
