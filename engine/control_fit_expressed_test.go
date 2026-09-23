package engine

import (
	"strings"
	"testing"
)

// TestControlsWaitForTheExpressedFit is a regression test for a reported
// auction: 2C – 2D – 2H – 2S, and East, holding AKQ5 opposite the five spades
// West has just promised, opened the control bids with 3C. The fit was real —
// nine trumps — but East knew it alone. West had heard no spade support, so at
// his end 3C was a natural club bid, and the "controls" he answered were shown
// for a trump nobody had named ([S-0]: "le fit doit d'abord avoir été exprimé").
//
// East must name the trump first. 3S is forcing, the controls run from there,
// and the pair reaches the same 6S — this time on a suit both hands have bid.
func TestControlsWaitForTheExpressedFit(t *testing.T) {
	const pbn = `[Dealer "N"]
[Deal "N:7.Q9652.A8542.K6 AKQ5.AKJT43.9.AT 982.7.JT7.987532 JT643.8.KQ63.QJ4"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 2 {
			if got := sc.Call.Format("fr"); got != "3P" {
				t.Fatalf("East's third call = %s (%s), want 3P naming the trump\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if strings.Contains(sc.M.fr, "contrôle") && !strings.Contains(sc.M.fr, "avant les contrôles") {
				t.Fatalf("East cued before the fit was expressed: %q\nauction: %s",
					sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	// Every control in this auction must come after both hands have shown
	// spades: the raise is East's third call, so nothing before it may cue.
	for i, sc := range calls {
		if !strings.Contains(sc.M.fr, "enchère de contrôle") {
			continue
		}
		if i < 8 {
			t.Fatalf("control bid at call %d (%s) before the fit was expressed\nauction: %s",
				i, sc.M.fr, formatAuction(calls))
		}
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "6P" {
		t.Fatalf("final contract = %s, want 6P\nauction: %s", got, formatAuction(calls))
	}
}
