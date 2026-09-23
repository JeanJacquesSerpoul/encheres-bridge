package main

import "testing"

// TestDeclineNTInviteWithDistributionalMinimum checks that opener declines a
// notrump game invitation with a distributional minimum, rather than accepting
// on inflated length points.
//
// West opens 1D and rebids 1S (second suit at the one level, unlimited), East invites with
// 2NT. West is 4-5-0-4 with 13 HCP: 14 HL once the fifth diamond is counted,
// which cleared the shownMin+2 acceptance bar and blasted 3NT. But the void is
// worthless in notrump and 13 HCP is a dead minimum, so 3NT fails (the pair
// takes only eight tricks). West must decline and pass 2NT.
func TestDeclineNTInviteWithDistributionalMinimum(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "N:AT43.AQ97.T84.65 5.KJ852.KQ.J9873 K987.T643.976.A2 QJ62..AJ532.KQT4"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "2SA" {
		t.Fatalf("final contract = %s, want 2SA (invitation declined)\nauction: %s",
			got, formatAuction(calls))
	}
}
