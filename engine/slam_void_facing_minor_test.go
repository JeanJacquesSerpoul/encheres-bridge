package engine

import "testing"

// TestSlamTryVoidFacingMinorOpening checks two slam-valuation points. A void
// facing the four diamonds of partner's 1D opening keeps its three points
// [E-1c]: only a five-card suit makes the shortness waste. And over partner's
// jump to game (20-23 HLD, a narrow range) a combined 34 is cushion enough to
// probe above game [S-3]. 1D - 1H - 4H with QJ5 Q9762 - A7542: 14 HLD + 20 =
// 34, West cue-bids and the pair reaches 6H.
func TestSlamTryVoidFacingMinorOpening(t *testing.T) {
	const pbn = `[Dealer "N"]
[Vulnerable "None"]
[Deal "N:A8762.T3.KQ97.J9 K3.AKJ5.AJ864.KT T94.84.T532.Q863 QJ5.Q9762..A7542"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(6, SHearts) {
		t.Fatalf("final contract = %s, want 6C\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}
