package engine

import (
	"strings"
	"testing"
)

// TestAdvanceForcedOverTakeoutDouble checks that a takeout double forces the
// advancer to answer when its right-hand opponent has not bid over the double.
// West passed as dealer, North opened 1D, East doubled (any-shape 18H+), South
// passed: West holds only 2H but four hearts, so it must answer 1H (0-7) rather
// than pass and convert the double into a penalty pass. The bug had West read
// the opening 1D (made before the double) as an intervening bid and pass out.
func TestAdvanceForcedOverTakeoutDouble(t *testing.T) {
	// Donneur West. E = KQ9643.A9.AK6.K6 (19H) doubles N's 1D. W = 752.J742.J54.854.
	pbn := `[Dealer "W"]
[Deal "W:752.J742.J54.854 AT.K53.QT97.AJT7 KQ9643.A9.AK6.K6 J8.QT86.832.Q932"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const west = 3 // seats: 0=N, 1=E, 2=S, 3=W
	// West's *second* call (its first is the dealer pass) must be a bid, not a pass.
	n := 0
	for _, sc := range calls {
		if sc.Seat != west {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "1C" {
				t.Fatalf("West advance = %s (%s), want 1C (forced answer to the double)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "réponse au contre") {
				t.Fatalf("comment %q does not describe an answer to the double", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("West never made a second call\nauction: %s", formatAuction(calls))
}
