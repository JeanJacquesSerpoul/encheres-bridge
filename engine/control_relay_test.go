package main

import (
	"strings"
	"testing"
)

// nthCallBy returns a seat's n-th call (0-based), passes included.
func nthCallBy(calls []SeatCall, seat, n int) (SeatCall, bool) {
	i := 0
	for _, sc := range calls {
		if sc.Seat != seat {
			continue
		}
		if i == n {
			return sc, true
		}
		i++
	}
	return SeatCall{}, false
}

const (
	seatN = 0
	seatS = 2
)

// TestControlStartsBelowTheUrgentControl is the document's headline example.
// South opens 1S on AT8654.93.A.AKQ2 and North's limit raise to 3S puts the
// pair in the slam zone. South needs the heart control and nothing else, so he
// names the control immediately BELOW hearts -- 4D -- making North's cheapest
// answer exactly the one card in question ("Commencer par nommer le contrôle
// immédiatement inférieur à celui qu'il est urgent de découvrir").
//
// The bid must also NOT deny the club control it skipped over: the player who
// opens the exchange chooses his step freely, and only suits skipped from that
// step onward are denied.
func TestControlStartsBelowTheUrgentControl(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:AT8654.93.A.AKQ2 2.765.KQJT4.J876 KJ97.AQ42.8532.3 Q3.KJT8.976.T954"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	sc, ok := nthCallBy(calls, seatS, 1)
	if !ok {
		t.Fatalf("South never made a second call\nauction: %s", formatAuction(calls))
	}
	if got := sc.Call.Format("fr"); got != "4K" {
		t.Fatalf("South's rebid = %s (%s), want 4K (contrôle immédiatement sous le Cœur)\nauction: %s",
			got, sc.M.fr, formatAuction(calls))
	}
	if !strings.Contains(sc.M.fr, "contrôle") {
		t.Fatalf("comment %q does not describe a control bid", sc.M.fr)
	}
	if sc.M.deniedCtrl[Clubs] {
		t.Fatalf("4D wrongly denied the club control: the opening step of the exchange denies nothing below it\nauction: %s",
			formatAuction(calls))
	}
}

// TestControlRelayAsksForClubs checks the 3SA "relais contrôle". South opens
// 1S on AK8652.KQJ2.A.T4, North raises to 3S, and the one control South cannot
// ask for economically is clubs -- no cue sits below it. 3SA asks for it.
// North, holding no club control, skips the step (which denies it) and names
// his diamond king instead; South then has his answer and stops in game.
func TestControlRelayAsksForClubs(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:AK8652.KQJ2.A.T4 .765.98743.QJ987 QJT9.A43.K652.32 743.T98.QJT.AK65"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	ask, ok := nthCallBy(calls, seatS, 1)
	if !ok || ask.Call.Format("fr") != "3SA" {
		t.Fatalf("South's rebid = %s, want 3SA (relais contrôle)\nauction: %s",
			ask.Call.Format("fr"), formatAuction(calls))
	}
	if !strings.Contains(ask.M.fr, "relais contrôle") || !strings.Contains(ask.M.fr, "Trèfle") {
		t.Fatalf("comment %q is not the club control relay", ask.M.fr)
	}

	ans, ok := nthCallBy(calls, seatN, 1)
	if !ok || ans.Call.Format("fr") != "4K" {
		t.Fatalf("North's answer = %s, want 4K (saute le Trèfle, nomme le Carreau)\nauction: %s",
			ans.Call.Format("fr"), formatAuction(calls))
	}
	if !ans.M.deniedCtrl[Clubs] {
		t.Fatalf("skipping the relay step must deny the club control\nauction: %s", formatAuction(calls))
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P: no club control, no slam\nauction: %s",
			got, formatAuction(calls))
	}
}

// TestControlRelayAnsweredPositively guards the other side of the same relay:
// with the club king, North answers by the immediately higher step, 4C, and
// the exchange runs on to the slam.
func TestControlRelayAnsweredPositively(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:AK8652.KQJ2.A.T4 .765.987432.QJ75 QJT9.A43.65.K632 743.T98.KQJT.A98"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	ans, ok := nthCallBy(calls, seatN, 1)
	if !ok || ans.Call.Format("fr") != "4T" {
		t.Fatalf("North's answer = %s, want 4T (réponse positive au relais)\nauction: %s",
			ans.Call.Format("fr"), formatAuction(calls))
	}
	if !strings.Contains(ans.M.fr, "positive") {
		t.Fatalf("comment %q is not the positive relay answer", ans.M.fr)
	}
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 {
		t.Fatalf("final contract = %s, want a slam once every control is located\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestNoSpadeRelayOnHeartFit guards the correction of the spade "relais": on
// a heart fit 3S is simply the cheapest control bid of the rotation, so it
// SHOWS the spade control and never asks for it. South here holds J3 and has
// no spade control to show, so the exchange opens at 4C -- and the step it
// stepped over denies the spade control, which is exactly what partner needs
// to read. (Deal of the document's ideal sequence, whose 3S ask this test
// used to assert.)
func TestNoSpadeRelayOnHeartFit(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AQ85.Q954.A.A765 K97.J62.KQJT.Q83 J3.AKT873.95.KJ2 T642..876432.T94"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.M.ctrlRelay && sc.M.ctrlRelaySuit == Spades {
			t.Fatalf("3S is a control bid on a heart fit, never a relay\nauction: %s", formatAuction(calls))
		}
	}
	cue, ok := nthCallBy(calls, seatS, 1)
	if !ok || cue.Call.Format("fr") != "4T" {
		t.Fatalf("South's rebid = %s, want 4T (le contrôle le moins cher qu'il détient)\nauction: %s",
			cue.Call.Format("fr"), formatAuction(calls))
	}
	if !cue.M.deniedCtrl[Spades] {
		t.Fatalf("stepping over 3S must deny the spade control\nauction: %s", formatAuction(calls))
	}
	if !strings.Contains(cue.M.fr, "sans contrôle à Pique") {
		t.Fatalf("comment %q does not spell out the denied spade control", cue.M.fr)
	}
}
