package engine

import (
	"strings"
	"testing"
)

// TestReopenMajorBeforeNT checks that a good five-card major is named in the
// balancing seat even at the two level, ahead of the 1NT reopening [V-4].
// South holds J93 KJT53 AKT 64 after 1S - Pass - Pass: 13 HL, balanced, a
// spade hold -- all 1NT asks -- but the hearts are the side's suit. South
// reopens 2H.
func TestReopenMajorBeforeNT(t *testing.T) {
	const south = 2
	const pbn = `[Dealer "W"]
[Vulnerable "EW"]
[Deal "E:Q2.2.8764.KT9753 J93.KJT53.AKT.64 KT864.Q97.Q2.AJ2 A75.A864.J953.Q8"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if got := sc.Call.Format("fr"); got != "2C" {
			t.Fatalf("South's reopening = %s (%s), want 2C\nauction: %s", got, sc.M.fr, formatAuction(calls))
		}
		if !strings.Contains(sc.M.fr, "réveil") {
			t.Fatalf("South's 2C comment %q does not say it is a reopening", sc.M.fr)
		}
		return
	}
	t.Fatalf("South never called\nauction: %s", formatAuction(calls))
}
