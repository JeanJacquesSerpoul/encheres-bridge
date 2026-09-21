package main

import (
	"strings"
	"testing"
)

// TestPassedHandKeepsItsCeiling and TestFitBeforeControlsOnTheSecondDoor are
// two halves of one reported deal. North passed as dealer, answered 1S, and
// then — over South's reverse — opened the control bids with 3C on a fit
// nobody had named, before asking Blackwood on a floor of fourteen points his
// own pass had already denied.
const passedHandDeal = `[Dealer "N"]
[Vulnerable "None"]
[Deal "N:A8762.QJ83.A985. KT43.62.KJ.T6543 Q.AK.T763.AQJ872 J95.T9754.Q42.K9"]`

// Three faults on this deal, and each one alone would still have bid the
// slam. North's club void faced South's own club suit, so the three points
// HLD grants for it were waste, not ruffing value [E-1c] -- and the reverse
// never recorded the four clubs that would have shown it [RO-19]. The control
// machinery's second door obeyed only the weaker trumpAgreed, so North cued a
// suit he had never supported [S-0]. And 4NT with a minor agreed commits the
// side: the answer 5H sits above 5D, so the sign-off bumps into the slam it
// meant to refuse [S-1].
//
// Counted honestly the pair holds 30, not 33. The probe never opens, and
// North bids the game his count actually supports.
func TestPassedHandDoesNotDriveToSlam(t *testing.T) {
	d, err := ParsePBN([]byte(passedHandDeal))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	for _, sc := range calls {
		if sc.Seat == north && strings.Contains(sc.M.fr, "enchère de contrôle") {
			t.Fatalf("North cued before naming the fit: %q\nauction: %s", sc.M.fr, formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if contract.Level >= 6 {
		t.Fatalf("final contract = %s: slam on a 4-4 trump fit missing KQJ\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// A pass before anyone opened caps the hand at 11 [E-1b]; no later bid may
// raise that ceiling to match its own announced floor.
func TestPassedHandKeepsItsCeiling(t *testing.T) {
	d, err := ParsePBN([]byte(passedHandDeal))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	e := NewEngine(d)
	calls := e.Run()

	const north = 0
	p := e.ps[north]
	if !p.passedOpening {
		t.Fatalf("North's opening pass was not recorded\nauction: %s", formatAuction(calls))
	}
	if p.shownMax > 11 {
		t.Fatalf("North showed a ceiling of %d after passing as dealer\nauction: %s",
			p.shownMax, formatAuction(calls))
	}
	if p.shownMin > p.shownMax {
		t.Fatalf("North's shown range is inverted: [%d..%d]\nauction: %s",
			p.shownMin, p.shownMax, formatAuction(calls))
	}
}
