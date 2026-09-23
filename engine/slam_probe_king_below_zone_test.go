package engine

import "testing"

// TestSlamProbeOpensAKingBelowTheZone: with 18 HLD in support of the heart
// fit facing a "soutien simple, 12-16HLD", South's combined minimum is 30 --
// a king below the slam zone [S-1] -- so the pair must look before settling
// for game. The probe used to need 31 opposite a narrow raise and the hand
// blasted 4H without a question asked.
func TestSlamProbeOpensAKingBelowTheZone(t *testing.T) {
	pbn := `[Dealer "W"]
[Vulnerable "EW"]
[Deal "W:98.J73.KT72.9732 Q542.AK65.54.KT5 AKJT63.8.9863.J8 7.QT942.AQJ.AQ64"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	cued := false
	for _, sc := range calls {
		if sc.Seat == south && sc.M.controlBid {
			cued = true
		}
	}
	if !cued {
		t.Fatalf("South never explored the slam by controls\nauction: %s", formatAuction(calls))
	}
	// North holds no diamond or spade control, so the exchange must come back
	// to game rather than bid a slam on faith.
	contract, _, _ := finalContract(calls)
	if contract.Level != 4 || contract.Strain != SHearts {
		t.Fatalf("final contract = %s, want 4C (4H) after the exchange\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
