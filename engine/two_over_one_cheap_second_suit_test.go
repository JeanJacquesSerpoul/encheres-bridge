package engine

import "testing"

// TestNoJumpSecondSuitOverFittedTwoOverOne checks that opener shows his cheap
// second suit at the two level over a two-over-one response, even with 18 HL,
// when that response is the fitted "changement de couleur avant soutien".
//
// North opens 1S on KJ9762.Q4.AKJT.Q (16 H, 6-2-4-1, 18 HL) and South answers
// 2C on QT4.A96.53.AK542 -- three spades and 13+ HLD, the new suit before
// supporting [RM-4]. The engine jumped to 3D ("saut dans la seconde couleur"),
// but [RO-19] keeps the economical 2D after a two-over-one: partner cannot
// pass it, so it already carries the whole range, and the level the jump
// spends belongs to the slam exploration. The fitted two-over-one did not set
// the game force the rule tested, so the exception was skipped.
func TestNoJumpSecondSuitOverFittedTwoOverOne(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "All"]
[Deal "N:KJ9762.Q4.AKJT.Q 3.T852.Q762.9863 QT4.A96.53.AK542 A85.KJ73.984.JT7"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0 // seats: 0=N, 1=E, 2=S, 3=W
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "2K" {
				t.Fatalf("North's rebid = %s (%s), want 2K (bicolore économique)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	// The cheap bid carries no ceiling here: the side must still reach game
	// (18 HL facing 13+ HLD with a spade fit).
	var last Call
	for _, sc := range calls {
		if sc.Call.Kind == KindBid {
			last = sc.Call
		}
	}
	if last.Level < 4 && !(last.Level == 3 && last.Strain == SNoTrump) {
		t.Fatalf("final contract %s stays below game\nauction: %s",
			last.Format("fr"), formatAuction(calls))
	}
}

// TestDelayedSupportThenControls follows the same deal one round further.
// Over 2D, South -- fitted two-over-one, 13+ HLD -- must name the trump at the
// three level, forcing ("fit différé" [RM-4b]) rather than jump to 4S, which
// would shut opener out. North, 20 HLD with a slam in view, then starts the
// control bids: his singleton is the club queen, which fills South's clubs,
// so neither the 3NT "oui mais" [S-2d] nor the slam valuation holds it
// against him.
func TestDelayedSupportThenControls(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "All"]
[Deal "N:KJ9762.Q4.AKJT.Q 3.T852.Q762.9863 QT4.A96.53.AK542 A85.KJ73.984.JT7"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	if len(calls) < 9 {
		t.Fatalf("auction too short: %s", formatAuction(calls))
	}
	south := calls[6] // N 1S, E -, S 2C, W -, N 2D, E -, S ?
	if south.Call != bidSuit(3, Spades) || !south.M.forcing {
		t.Fatalf("South's second call = %s (%s), want a forcing 3P (fit différé)\nauction: %s",
			south.Call.Format("fr"), south.M.fr, formatAuction(calls))
	}
	north := calls[8]
	if north.Call.Kind != KindBid || north.Call.Strain == SNoTrump || north.Call == bidSuit(4, Spades) {
		t.Fatalf("North's call over 3P = %s (%s), want a control bid\nauction: %s",
			north.Call.Format("fr"), north.M.fr, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 {
		t.Fatalf("final contract = %s, want the small slam\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
