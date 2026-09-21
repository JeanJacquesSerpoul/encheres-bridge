package main

import (
	"strings"
	"testing"
)

// TestDeclineInviteVoidRetreatsToOwnSuit checks that an opener declining a
// suit invitation does not pass with a void in partner's suit. East opens 1C
// with .Q94.A93.KQJT764 and rebids 2C; West (6 spades, invitational) proposes
// 2S. East must not pass -- that strands the side in a 6-0 spade "fit" -- but
// retreat to 3C on its seven-card suit.
func TestDeclineInviteVoidRetreatsToOwnSuit(t *testing.T) {
	pbn := `[Dealer "W"]
[Deal "W:987653.AK87.JT.2 AKJ2.T32.Q864.93 .Q94.A93.KQJT764 QT4.J65.K752.A85"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1 // seats: 0=N, 1=E, 2=S, 3=W
	// East's calls: 1T (open), 2T (rebid), then the call over West's 2P invite.
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 2 {
			if got := sc.Call.Format("fr"); got != "3T" {
				t.Fatalf("East third call = %s (%s), want 3T (declines, void in spades)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "chicane") {
				t.Fatalf("comment %q does not mention the void", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("East never made a third call\nauction: %s", formatAuction(calls))
}
