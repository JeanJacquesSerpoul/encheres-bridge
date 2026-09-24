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
