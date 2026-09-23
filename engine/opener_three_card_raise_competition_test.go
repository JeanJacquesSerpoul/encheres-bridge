package engine

import "testing"

// TestOpenerThreeCardRaiseInCompetition checks that opener does not pass a
// forcing 1-level response once RHO has bid over it. Sequence: S 1D, W 1H,
// N 1S (forcing), E 2H, S ? — South (A62.K.A9542.A543, 15H, singleton in the
// enemy suit, three-card spade support) has no shape-showing rebid left (the
// cheap second suit and the 1NT rebid are both gone), but 15 points opposite a
// forcing response cannot pass: it raises the known 4-3 spade fit, here a jump
// to 3S (game try) that carries N-S to 4S.
func TestOpenerThreeCardRaiseInCompetition(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "All"]
[Deal "S:A62.K.A9542.A543 K43.AQT96.QT8.87 QJT975.J2.KJ.T92 8.87543.763.KQJ6"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south, north = 2, 0
	var southRebid *SeatCall
	seen := 0
	for i := range calls {
		if calls[i].Seat != south {
			continue
		}
		if seen == 1 { // 1D opening is the first call
			southRebid = &calls[i]
			break
		}
		seen++
	}
	if southRebid == nil {
		t.Fatalf("South never made a second call\nauction: %s", formatAuction(calls))
	}
	if southRebid.Call.Kind == KindPass {
		t.Fatalf("South passed a forcing response with 15 points\nauction: %s", formatAuction(calls))
	}
	if got := southRebid.Call.Format("fr"); got != "3P" {
		t.Fatalf("South rebid = %s (%s), want 3P (jump raise of the 4-3 spade fit)\nauction: %s",
			got, southRebid.M.fr, formatAuction(calls))
	}

	// North, with six spades and useful cards, must accept: N-S belong in 4S.
	reached4S := false
	for _, sc := range calls {
		if sc.Seat == north && sc.Call == bidSuit(4, Spades) {
			reached4S = true
		}
	}
	if !reached4S {
		t.Fatalf("North never bid 4S over the game try\nauction: %s", formatAuction(calls))
	}
}
