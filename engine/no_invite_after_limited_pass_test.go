package main

import "testing"

// TestNoInviteAfterLimitedPass reproduces the deal where South (J3.7642.K754.QJ6,
// 7H) passed over East's 1H intervention (showing 0-7) and then raised North's
// 2C repetition to 3C labelled "proposition de manche". An invitation facing the
// 12-17 repetition promises at least 8 points — more than the pass already
// denied — so South must simply pass 2C.
func TestNoInviteAfterLimitedPass(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:K95.KQ.93.AKT752 A642.JT853.AT8.3 J3.7642.K754.QJ6 QT87.A9.QJ62.984"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	// Every South call must be a pass: 0-7 facing 12-17 never invites game.
	const south = 2
	for _, sc := range calls {
		if sc.Seat == south && sc.Call.Kind != KindPass {
			t.Fatalf("South bid %s (%s); 7H after a 0-7 pass must keep passing\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "2T" {
		t.Fatalf("final contract = %s, want 2T\nauction: %s", got, formatAuction(calls))
	}
}
