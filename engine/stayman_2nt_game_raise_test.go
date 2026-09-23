package engine

import "testing"

// TestStaymanOver2NTRaisesToFourNotFive checks that after a 2NT opening and a
// 3C Stayman ask, responder's natural game raise of opener's major answer lands
// in four of the major (game), not five.
//
// The bug: afterStaymanOneMajor bid the game raise as bidSuit(askLevel+2, M).
// Over a 1NT opening (askLevel 2) that is 4M, correct; but over a 2NT opening
// (askLevel 3) it produced 5M, overshooting the making game by a level.
//
// North = K6.AK984.AKQ5.J8 (20 HCP, 2NT opening). South = QT92.JT72.J93.AK
// (11 HCP, four hearts). Auction: 2NT - 3C(Stayman) - 3H - should be 4H.
func TestStaymanOver2NTRaisesToFourNotFive(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:K6.AK984.AKQ5.J8 J8.Q63.642.Q9532 QT92.JT72.J93.AK A7543.5.T87.T764"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4C" {
		t.Fatalf("final contract = %s, want 4C (heart game)\nauction: %s", got, formatAuction(calls))
	}
}
