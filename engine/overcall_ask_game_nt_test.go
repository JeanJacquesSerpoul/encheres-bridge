package engine

import (
	"strings"
	"testing"
)

// TestOvercallAskOpeningValuesThenNT checks that the advancer, told by the
// overcaller's answer to his strength-asking cue-bid that the side holds
// opening values, bids the game -- and 3NT ahead of the minor game when the
// opponents' suit is held and his hand is balanced [A-6]. 1C - 1D - 2C - 2NT:
// North, AK KJ85 952 T852, bids 3NT, not a 3D invitation.
func TestOvercallAskOpeningValuesThenNT(t *testing.T) {
	const north = 0
	const pbn = `[Dealer "E"]
[Vulnerable "EW"]
[Deal "E:JT75.QT2.QT4.AQJ Q3.A3.AKJ86.K743 98642.9764.73.96 AK.KJ85.952.T852"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var south2NT bool
	n := 0
	for _, sc := range calls {
		if sc.Seat == 2 && sc.Call.Format("fr") == "2SA" {
			south2NT = true
			if !sc.M.stops[Clubs] {
				t.Fatalf("South's 2SA does not record the club stopper it announces (%s)", sc.M.fr)
			}
		}
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if !south2NT {
				t.Fatalf("South did not answer 2SA\nauction: %s", formatAuction(calls))
			}
			if got := sc.Call.Format("fr"); got != "3SA" {
				t.Fatalf("North's rebid = %s (%s), want 3SA\nauction: %s", got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "3SA") {
				t.Fatalf("North's comment %q does not name 3SA", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("North made fewer than 2 calls\nauction: %s", formatAuction(calls))
}
