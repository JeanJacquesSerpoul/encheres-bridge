package engine

import (
	"strings"
	"testing"
)

// TestLawRaiseNeedsResponseFloor replays a reported deal where the law of
// total tricks talked a hand into a raise it had no right to make. North holds
// 98642.J542.J2.T5 -- 2 H, 5 HLD with the heart fit -- and over South's 1H
// opening and West's 2C overcall the nine-card fit alone pushed him to 3H,
// which South then carried to a hopeless 4H.
//
// The law fixes the *level* of a support bid, not the right to speak: a
// response needs six points whatever the trump length [L-1]/[RC-3]. Below the
// floor North passes. The fit is not lost -- [L-1b] picks it up later, in the
// partscore battle, on trump length alone -- but that bid promises nothing, so
// South must not read it as an invitation and must pass it too.
func TestLawRaiseNeedsResponseFloor(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "None"]
[Deal "S:KQ.KQ763.KQT5.J3 JT73.A8.3.AK9642 98642.J542.J2.T5 A5.T9.A98764.Q87"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north, south = 0, 2
	byNorth := make([]SeatCall, 0, 4)
	for _, sc := range calls {
		if sc.Seat == north {
			byNorth = append(byNorth, sc)
		}
	}
	if len(byNorth) == 0 {
		t.Fatalf("North never called\nauction: %s", formatAuction(calls))
	}
	if got := byNorth[0].Call; got.Kind != KindPass {
		t.Fatalf("North's first call = %s (%s), want Passe: 5 HLD is under the six-point response floor\nauction: %s",
			got.Format("fr"), byNorth[0].M.fr, formatAuction(calls))
	}

	// Whatever North says later must not be read as values: South, a bare
	// minimum opener, must never reach game opposite a hand that denied the
	// floor to respond.
	for _, sc := range calls {
		if sc.Seat == south && sc.Call.IsBid() && isGame(sc.Call) {
			t.Fatalf("South bid the game %s (%s) opposite a hand that could not respond\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}

	// The law still gets its say in the partscore battle, and it promises
	// nothing in points [L-1b].
	law := false
	for _, sc := range byNorth {
		if strings.Contains(sc.M.fr, "on ne leur laisse pas la partielle") {
			law = true
			if sc.M.minPts > 0 {
				t.Fatalf("the partscore law bid promised %d points; it must promise nothing\nauction: %s",
					sc.M.minPts, formatAuction(calls))
			}
		}
	}
	if !law {
		t.Fatalf("North never contested the partscore with the known nine-card fit\nauction: %s", formatAuction(calls))
	}
}
