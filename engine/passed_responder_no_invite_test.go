package engine

import (
	"testing"
)

// TestPassedResponderDoesNotInvite checks that a responder who has already
// passed partner's opening (denying 6+ HCP) does not later invite game. East
// opens 1C, West passes (4 HCP), North overcalls 1SA, and the auction is passed
// back to West. West must pass: the bug had it bid 2SA "proposition de manche"
// because the opener's wide 12-23 range pushed the combined maximum into the
// game zone, ignoring that West had limited itself below 6 points.
func TestPassedResponderDoesNotInvite(t *testing.T) {
	// Donneur East. W = J9842.KT3.86.873 : 4 HCP, passes 1C. N overcalls 1SA.
	pbn := `[Dealer "E"]
[Deal "E:75.AJ4.K92.AT542 QT63.Q965.QT54.Q J9842.KT3.86.873 AK.872.AJ73.KJ96"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	// West must never bid: with 4 HCP it can only pass throughout.
	const west = 3 // seats: 0=N, 1=E, 2=S, 3=W
	for _, sc := range calls {
		if sc.Seat == west && sc.Call.Kind != KindPass {
			t.Fatalf("West bid %s (%s) with 4 HCP after passing the opening\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}

	// North sits in the classic balancing seat (1C - Passe - Passe), where an
	// immediate 1SA would promise only 9-13 HL: with 16H the system is double
	// first, then the cheapest notrump over partner's minimum answer
	// (docs/regles_moteur.md §9.3, the balanced 14-16 double).
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "1SA" {
		t.Fatalf("final contract = %s, want 1SA (reopening double, then the cheapest notrump)\nauction: %s", got, formatAuction(calls))
	}
}
