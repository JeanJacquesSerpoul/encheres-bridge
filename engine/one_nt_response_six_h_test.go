package engine

import (
	"strings"
	"testing"
)

// TestOneNTResponseNeedsSixH checks that the 1NT "poubelle" response to a
// major opening needs 6 H of its own [RM-6]. East holds Q2 2 8764 KT9753 over
// West's 1S: 7 HL, but only 5 H -- the two length points belong to a club
// suit that notrump will never use. East passes.
func TestOneNTResponseNeedsSixH(t *testing.T) {
	const east = 1
	const pbn = `[Dealer "W"]
[Vulnerable "EW"]
[Deal "E:Q2.2.8764.KT9753 J93.KJT53.AKT.64 KT864.Q97.Q2.AJ2 A75.A864.J953.Q8"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if got := sc.Call.Format("fr"); got != "Passe" {
			t.Fatalf("East's response = %s (%s), want Passe\nauction: %s", got, sc.M.fr, formatAuction(calls))
		}
		if !strings.Contains(sc.M.fr, "6 points H") {
			t.Fatalf("East's pass comment %q does not say fewer than 6 H", sc.M.fr)
		}
		return
	}
	t.Fatalf("East never called\nauction: %s", formatAuction(calls))
}
