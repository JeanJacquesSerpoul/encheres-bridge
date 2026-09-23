package engine

import (
	"strings"
	"testing"
)

// TestStrong2DSecondSuitOverThreeNTFindsMinorSlam reproduces the reported
// deal. South opens 2K on a 5-1-2-5 black two-suiter (22 H), names the spades,
// and North, with no ace but a singleton spade, signs off in 3SA. The old
// auction then carried on as if South were balanced: a quantitative 4SA, a
// partial acceptance at 5SA, and the pair stopped in 5SA played by North.
//
// Two things were missing. South had not finished describing the hand: the
// clubs are named at the four level [S-13c], and North's four of them make the
// fit. Then, over North's 5T, the ace step has already said North holds no
// ace: South's four keys (three aces and the trump king) with the slam count
// reached are [S-7]'s small slam, with no room left for any ask [S-7b].
//
// South must declare: from North, an East diamond lead through the 975 into
// K8 sets up two diamond losers before the hearts can be run.
func TestStrong2DSecondSuitOverThreeNTFindsMinorSlam(t *testing.T) {
	pbn := `[Dealer "W"]
[Deal "N:9.KQJ97.975.QT94 Q53.T6543.J42.63 AKT72.A.K8.AKJ52 J864.82.AQT63.87"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north, south = 0, 2
	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{south, 0, "2K", "forcing de manche"},
		{north, 1, "2C", "pas d'As"},
		{south, 1, "2P", "couleur cinquième"},
		{north, 2, "3SA", ""},
		{south, 2, "4T", "deuxième couleur"},
		{north, 3, "5T", ""},
		{south, 3, "6T", "petit chelem"},
	}
	for _, w := range seq {
		n, found := 0, false
		for _, sc := range calls {
			if sc.Seat != w.seat {
				continue
			}
			if n == w.nth {
				if got := sc.Call.Format("fr"); got != w.call {
					t.Fatalf("seat %s call #%d = %s (%s), want %s\nauction: %s",
						seatNames[w.seat], w.nth, got, sc.M.fr, w.call, formatAuction(calls))
				}
				if w.hint != "" && !strings.Contains(sc.M.fr, w.hint) {
					t.Fatalf("seat %s call #%d comment %q does not mention %q\nauction: %s",
						seatNames[w.seat], w.nth, sc.M.fr, w.hint, formatAuction(calls))
				}
				found = true
				break
			}
			n++
		}
		if !found {
			t.Fatalf("seat %s never made call #%d\nauction: %s", seatNames[w.seat], w.nth, formatAuction(calls))
		}
	}

	contract, declarer, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "6T" || declarer != south {
		t.Fatalf("final contract = %s by %s, want 6T by S\nauction: %s",
			got, seatNames[declarer], formatAuction(calls))
	}
}

// TestSecondSuitNoFitAnswersTheCount guards the other branch of [S-13c]: the
// partner who signed off in 3SA has no fit for the second suit, so he answers
// the count in notrump. East's 2K, West's club ace, East's hearts, West's 3SA,
// then East names the diamonds: West holds 11 H facing a 24 floor, the 33 are
// there, and the answer is 6SA -- the slam the quantitative 4SA used to reach
// on this deal -- not the flat 4SA the generic endgame bid.
func TestSecondSuitNoFitAnswersTheCount(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "N:J753.52.K863.T32 AK.AKQJT.AJ542.9 62.9864.T7.K8754 QT984.73.Q9.AQJ6"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east, west = 1, 3
	var sawSecondSuit bool
	for _, sc := range calls {
		if sc.Seat == east && sc.M.secondSuitTry {
			sawSecondSuit = true
		}
	}
	if !sawSecondSuit {
		t.Fatalf("East never named the second suit over 3SA\nauction: %s", formatAuction(calls))
	}
	contract, declarer, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "6SA" || declarer != west {
		t.Fatalf("final contract = %s by %s, want 6SA by W\nauction: %s",
			got, seatNames[declarer], formatAuction(calls))
	}
}
