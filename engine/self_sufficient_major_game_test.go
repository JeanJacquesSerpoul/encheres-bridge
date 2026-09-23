package engine

import "testing"

// TestSelfSufficientMajorBidsGame checks that opener, holding a self-sufficient
// seven-card major with a void, drives to game over the 1NT "poubelle" response
// rather than making the non-forcing invitational jump rebid.
//
// North = AKT7653.A93..AT8 (15 HCP, 18 HL, but HLD=21 counting the void).
// The auction 1S - (P) - 1NT - (2D): the old engine rebid a merely invitational
// 3S, which South (maximum but with the spade fit unknown, holding only Jx)
// passed, landing in 3S. Six spades makes (par), so stopping in 3S is a gross
// underbid. Opener's seven-card suit is worth game opposite the 6-10 range on
// its own, so it must conclude in 4S.
func TestSelfSufficientMajorBidsGame(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AKT7653.A93..AT8 Q942.642.J98.Q52 J8.QJ5.T432.KJ73 .KT87.AKQ765.964"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0 // seats: 0=N, 1=E, 2=S, 3=W
	// North's second bid must be 4S, not the invitational 3S.
	n := 0
	for _, sc := range calls {
		if sc.Seat != north || sc.Call.Kind != KindBid {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "4P" {
				t.Fatalf("North's rebid = %s (%s), want 4P (game)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	// East-West, holding a nine-card diamond fit against the announced 420,
	// may lawfully save at 5D (down two doubled costs 300); what matters
	// here is that North-South never stop below their game.
	contract, declarer, _ := finalContract(calls)
	got := contract.Format("fr")
	if got != "4P" && !(got == "5K" && sideOf(declarer) == 1) {
		t.Fatalf("final contract = %s, want 4P (or the EW 5K save over it)\nauction: %s", got, formatAuction(calls))
	}
}
