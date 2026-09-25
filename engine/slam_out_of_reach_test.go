package engine

import "testing"

// TestSlamOutOfReachAfterTheirOpening checks that the opponents' opening caps
// our side's honours: 1C - 1D - 2C - 2NT - 3NT leaves North-South 30 H at most
// (40 minus East's 12 HL opening, at least 10 H), short of the 33 a slam needs
// [S-1]. South, Q3 A3 AKJ86 K743 with no singleton, passes 3NT instead of
// opening control bids that would land the pair in 5D.
func TestSlamOutOfReachAfterTheirOpening(t *testing.T) {
	const pbn = `[Dealer "E"]
[Vulnerable "EW"]
[Deal "E:JT75.QT2.QT4.AQJ Q3.A3.AKJ86.K743 98642.9764.73.96 AK.KJ85.952.T852"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}
