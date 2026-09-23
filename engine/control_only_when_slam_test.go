package engine

import "testing"

// TestNoControlBidWithoutSlamInView: "ne démarrer les enchères de contrôle que
// si on envisage un chelem". West opens 1D, North overcalls 2C, East raises to
// 2D, South bids his own spades (2S), North proposes game (3S). South holds
// QJ753 AJT32 76 7 -- a maximum for his non-forcing 2S, but the combined
// maximum is only 29 HLD, four short of the slam zone. There is no slam to
// explore: South accepts the game (4S) instead of launching a cue exchange.
func TestNoControlBidWithoutSlamInView(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(east, map[int]*Hand{
		north: hand("A42", "Q874", "Q", "AJ843"),
		east:  hand("T96", "K9", "T532", "KQT2"),
		south: hand("QJ753", "AJT32", "76", "7"),
		west:  hand("K8", "65", "AKJ984", "965"),
	})

	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.M.controlBid {
			t.Fatalf("started a cue exchange with no slam in view: %s %s (%s)\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P (game, no slam try)\nauction: %s", got, formatAuction(calls))
	}
}
