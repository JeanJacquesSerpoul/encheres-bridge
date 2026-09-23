package engine

import "testing"

// TestNTInviteAcceptedOnRunningLongSuit is the reported deal. North opens 1D
// on KT A94 AK7643 87, rebids 2D over South's 1S (13-17 HL) and hears 2NT.
// North used to pass as a minimum: without a fit the answer counted honours
// alone, 14 H, one short of the middle of the bracket. But the bracket was
// announced in HL, and North's 16 HL sits at its top. The length points of a
// six-card suit headed by the ace and king are the tricks 3NT plays for, not
// shape that may not run: North accepts.
func TestNTInviteAcceptedOnRunningLongSuit(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:A876.JT.QJT2.K65 QJ932.Q852..Q942 KT.A94.AK7643.87 54.K763.985.AJT3"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}
