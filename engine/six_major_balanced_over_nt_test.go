package engine

import (
	"strings"
	"testing"
)

// TestSixCardMajorBalancedOverNTRebid checks that a responder with a six-card
// major, even balanced, plays game in it once partner has rebid notrump.
// South opens 1C, North answers 1S with AKQJ54 864 KJ 73 and hears 1NT
// (12-14 regular): a balanced opener holds at least two spades, the
// eight-card fit is known, and 14 H opposite 12-14 leaves no slam. North
// bids 4S, not 3NT.
func TestSixCardMajorBalancedOverNTRebid(t *testing.T) {
	const north = 0
	const pbn = `[Dealer "S"]
[Vulnerable "EW"]
[Deal "E:T976.Q953.T64.Q6 82.AKJT.Q2.AT842 3.72.A98753.KJ95 AKQJ54.864.KJ.73"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "4P" {
				t.Fatalf("North's rebid = %s (%s), want 4P\nauction: %s", got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "manche") {
				t.Fatalf("North's 4P comment %q does not mention the game", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("North made fewer than 2 calls\nauction: %s", formatAuction(calls))
}
