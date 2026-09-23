package main

import "testing"

// TestResponderInvitesOverWideMinorRebid checks that responder invites with 2NT
// rather than jumping to 3NT when the combined count only just reaches game and
// opener's shown range is wide (a simple minor rebid, 13-17HL).
//
// South (passed hand, 11 HCP + a five-card spade suit) responds 1S to North's
// 1C, North rebids 2C (13-17, six clubs). The eight-card club fit made the
// engine value South by HLD and land cMin exactly on the 25-point notrump game
// threshold, so it blasted 3NT. But opposite opener's 13 floor that game is
// thin: South should invite with 2NT and let North bid game on his maximum
// (here 15 -> 3NT).
func TestResponderInvitesOverWideMinorRebid(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "N:QJ3.A.KQ74.K9654 754.KT62.J53.AT8 AK986.Q54.982.Q3 T2.J9873.AT6.J72"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2 // seats: 0=N, 1=E, 2=S, 3=W
	// South's second bid (after 1S) must be the 2NT invitation, not 3NT.
	n := 0
	for _, sc := range calls {
		if sc.Seat != south || sc.Call.Kind != KindBid {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "2SA" {
				t.Fatalf("South's rebid = %s (%s), want 2SA (invitation)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}
}
