package engine

import (
	"strings"
	"testing"
)

// TestAceStepTwoHeartsIsNotCappedAtSeven guards the range the 2H ace-step
// answer announces. The step covers two different hands (docs/bidings.md,
// "L'OUVERTURE DE 2♦"): "Pas d'As, 0-7H, ou 8H et plus avec une courte".
// Announcing a 0-7 ceiling is a lie about the second kind, and an
// irreversible one -- a later bid can raise a floor but never lift a ceiling
// already announced.
//
// Here South holds KJ732.K43.J.T532: 8 HCP and a singleton diamond, so 2H is
// the right call. Opposite North's 25 the pair holds 33 and belongs in a slam
// try; with the false ceiling North could never count past 32 and the auction
// died in 3NT.
func TestAceStepTwoHeartsIsNotCappedAtSeven(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:A4.AQJ.AKQ2.AJ87 QT65.652.T87.KQ9 KJ732.K43.J.T532 98.T987.96543.64"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	ans, ok := nthCallBy(calls, seatS, 0)
	if !ok || ans.Call.Format("fr") != "2C" {
		t.Fatalf("South's response = %s, want 2C (pas d'As, courte)\nauction: %s",
			ans.Call.Format("fr"), formatAuction(calls))
	}
	if ans.M.maxPts >= 0 && ans.M.maxPts <= 7 {
		t.Fatalf("2H announced a ceiling of %d: the step also covers 8H+ with a short suit\nauction: %s",
			ans.M.maxPts, formatAuction(calls))
	}

	// With the ceiling gone, North can count the pair to the slam zone once
	// South's own 3NT narrows him, and makes the quantitative try.
	tried := false
	for _, sc := range calls {
		if sc.Seat == seatN && sc.Call == bid(4, SNoTrump) && strings.Contains(sc.M.fr, "quantitatif") {
			tried = true
		}
	}
	if !tried {
		t.Fatalf("North never tried the slam on 33 combined honours\nauction: %s", formatAuction(calls))
	}
}
