package engine

import (
	"testing"
)

// After the Drury singleton ask (2SA - 3T - 3C), the responder's shortness is
// already on the table. When the auction then reaches the slam-zone control
// exchange, the responder must not "cue" that same shortness again: it tells
// partner nothing and points the auction at the wrong suit. With no other
// control below game he signs off in the fit [S-3b].
func TestDruryShortSuitNotRecuedAsControl(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "EW"]
[Deal "E:T7.AJ987.Q762.K4 AQ832.5.JT5.J763 .QT642.K9843.T95 KJ9654.K3.A.AQ82"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	var southSpadeSignoff bool
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if sc.M.controlBid && sc.Call.Strain == SHearts {
			t.Fatalf("South cue-bid %s to re-show a singleton partner already knows about\nauction: %s",
				sc.Call.Format("fr"), formatAuction(calls))
		}
		if sc.Call == bidSuit(4, Spades) {
			southSpadeSignoff = true
		}
	}
	if !southSpadeSignoff {
		t.Fatalf("South never signed off in 4P after running out of fresh controls\nauction: %s", formatAuction(calls))
	}
}
