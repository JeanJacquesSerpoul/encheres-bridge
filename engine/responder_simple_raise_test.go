package engine

import (
	"strings"
	"testing"
)

// TestResponderSimpleRaiseBelowInviteZone checks the zone that governs
// responder's rebid facing an opening that has limited nothing. West opens
// 1D, East answers 1H, West names spades at the one level -- which says
// nothing at all about his force [RO-19]. East holds SJ854 HK9642 D- CJT53:
// four spades, but 5 H and 9 HLD. The scheme gives 6-10 the simple raise and
// keeps the jump for 11-12 [RM-4], so the answer is 2S, not the 3S "game
// invitation" the generic count used to produce off a 12-23 maximum.
func TestResponderSimpleRaiseBelowInviteZone(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "N"]
[Vulnerable "EW"]
[Deal "N:Q63.Q87.7543.AK6 J854.K9642..JT53 T2.AJT5.J8.Q8742 AK97.3.AKQT962.9"]
`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	n := 0
	for _, sc := range calls {
		if sc.Seat != east || sc.Call.Kind == KindPass && n == 0 {
			continue // East dealt in third seat: the opening pass is not a response
		}
		n++
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "1C" {
				t.Fatalf("East's response = %s, want 1C\nauction: %s", got, formatAuction(calls))
			}
			continue
		}
		if got := sc.Call.Format("fr"); got != "2P" {
			t.Fatalf("East's rebid = %s (%s), want 2P: 9 HLD is under the invitational floor\nauction: %s",
				got, sc.M.fr, formatAuction(calls))
		}
		if strings.Contains(sc.M.fr, "proposition") {
			t.Fatalf("comment %q reads as an invitation, which 9 HLD does not carry", sc.M.fr)
		}
		return
	}
	t.Fatalf("East never rebid\nauction: %s", formatAuction(calls))
}
