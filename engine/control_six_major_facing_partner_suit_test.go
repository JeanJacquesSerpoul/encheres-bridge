package engine

import "testing"

// TestNoControlsSixMajorShortFacingPartnerSuit: "les enchères de contrôle sont
// utiles pour rechercher un chelem, ne pas les utiliser pour aller à une manche
// qui est évidente". 1H - 1S - 3D - 3NT: South (2 AKJ743 KQ42 K7) revalues
// the six hearts to 21 HLD, 31 facing North's floor -- but the stiff spade
// faces North's spades and buys no ruff, so the slam-valued count is 29. No
// slam in view: South bids 4H, not a 4C control.
func TestNoControlsSixMajorShortFacingPartnerSuit(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(east, map[int]*Hand{
		north: hand("Q874", "62", "AT9", "AJ64"),
		east:  hand("AJ53", "Q", "763", "QT953"),
		south: hand("2", "AKJ743", "KQ42", "K7"),
		west:  hand("KT96", "T985", "J85", "82"),
	})

	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.M.controlBid {
			t.Fatalf("control bid with no slam in view: %s %s (%s)\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4C" {
		t.Fatalf("final contract = %s, want 4C\nauction: %s", got, formatAuction(calls))
	}
}
