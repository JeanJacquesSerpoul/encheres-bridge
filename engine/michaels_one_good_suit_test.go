package engine

import (
	"strings"
	"testing"
)

// TestMichaelsWithOneGoodSuit checks that a 5-5 major two-suiter over a minor
// opening is shown by the 2D Michaels even when one suit is thin. South holds
// AKT65 J9762 J5 4 over East's 1C: the hearts are poor, but the spades are
// good and the honours sit in the two suits (8 H). A natural 1S would bury the
// hearts; 2D shows both and lets partner pick the trump.
func TestMichaelsWithOneGoodSuit(t *testing.T) {
	const south = 2
	const pbn = `[Dealer "S"]
[Vulnerable "EW"]
[Deal "E:7.KQ4.842.AKJ532 AKT65.J9762.J5.4 QJ982.5.KT73.QT8 43.AT83.AQ96.976"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	n := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "2K" {
				t.Fatalf("South's overcall = %s (%s), want 2K\nauction: %s", got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "Michaël") {
				t.Fatalf("South's 2K comment %q does not mention Michaels", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("South made fewer than 2 calls\nauction: %s", formatAuction(calls))
}
