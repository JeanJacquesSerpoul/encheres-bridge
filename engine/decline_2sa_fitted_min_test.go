package engine

import (
	"strings"
	"testing"
)

// TestDecline2SAFittedBalancedMinimum checks that a balanced minimum opener
// declines the conventional 2NT fit-showing raise (2SA fitté, 11-12 HLD)
// instead of straining to a failing game. South opens 1H with 83.A8764.AJT.K42
// (12 HCP, 5-3-3-2): its fourteenth HLD point comes solely from the spade
// doubleton, which is worthless opposite North's limited three-card raise, so
// South must sign off in 3H, not accept with 4H. The deal makes only eight
// heart tricks.
//
// South deals: the conventional 2NT raise belongs to a responder who has not
// passed. The same North hand facing a third- or fourth-seat opening answers
// 2C Drury instead [RM-2b].
func TestDecline2SAFittedBalancedMinimum(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:83.A8764.AJT.K42 JT962.Q.K74.J976 KQ7.T92.Q95.A853 A54.KJ53.8632.QT"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2 // seats: 0=N, 1=E, 2=S, 3=W
	// South's calls: 1H (open), then the rebid over North's 2SA fitté.
	n := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "3C" {
				t.Fatalf("South rebid = %s (%s), want 3C (declines, minimum)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "refus") {
				t.Fatalf("comment %q does not describe a declined invitation", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("South never made a second call\nauction: %s", formatAuction(calls))
}
