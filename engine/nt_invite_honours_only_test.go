package engine

import "testing"

// TestNoNTInviteOnFitShortness checks that a notrump invitation is measured on
// high cards alone, never on the shortness points a supposed minor fit adds.
//
// East opens 1C and rebids 1NT (12-14H) over West's 1H response. West holds
// 2.J987.Q87.KQ753 -- 8 HCP, 9 HL. With five clubs facing opener's three the
// engine agreed a club fit, revalued the hand as HLD (11, singleton spade) and
// invited with 2NT, which East accepted: 3NT on a 21-point pack. Opposite a
// 14-point maximum West can never bring the side to game, and the singleton
// buys no trick in notrump. West must pass the 1NT rebid.
func TestNoNTInviteOnFitShortness(t *testing.T) {
	pbn := `[Dealer "E"]
[Deal "S:AJ.KQ62.A964.J86 2.J987.Q87.KQ753 T87653.43.KJ32.4 KQ94.AT5.T5.AT92"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const west = 3 // seats: 0=N, 1=E, 2=S, 3=W
	// West's second call (after 1H) must be a pass, not an invitation.
	n := 0
	for _, sc := range calls {
		if sc.Seat != west {
			continue
		}
		if n == 1 {
			if sc.Call.Kind != KindPass {
				t.Fatalf("West's second call = %s (%s), want Passe\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}
}
