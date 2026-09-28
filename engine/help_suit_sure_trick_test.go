package engine

import "testing"

// TestHelpSuitTryNeedsSureTrick: accepting a help-suit try promises a sure
// trick in the try suit [C-19]. 1H - 2H - 3D: South holds Q2 in diamonds --
// no sure trick, the queen and the doubleton cover nothing for certain -- and
// declines with 3H instead of accepting the game.
func TestHelpSuitTryNeedsSureTrick(t *testing.T) {
	d, err := ParsePBN([]byte(`[Vulnerable "None"]
[Deal "S:754.T94.Q2.K8753 QT862.53.AJT8.T9 KJ9.AKQJ2.9643.Q A3.876.K75.AJ642"]
[Dealer "N"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	for i, sc := range calls {
		if !sc.M.helpSuitTry {
			continue
		}
		for _, next := range calls[i+1:] {
			if next.Seat != south {
				continue
			}
			if got := next.Call.Format("fr"); got != "3C" {
				t.Fatalf("South's answer to the try = %s (%s), want 3C\nauction: %s",
					got, next.M.fr, formatAuction(calls))
			}
			return
		}
	}
	t.Fatalf("no help-suit try in the auction: %s", formatAuction(calls))
}

// TestHasHelpSureTrick pins the definition: the ace, king-queen, or a
// singleton or void.
func TestHasHelpSureTrick(t *testing.T) {
	cases := []struct {
		holding string
		want    bool
	}{
		{"A2", true},    // l'As
		{"A432", true},  // l'As
		{"KQ3", true},   // Roi-Dame
		{"2", true},     // singleton
		{"", true},      // chicane
		{"K532", false}, // Roi seul : l'As peut être mal placé
		{"K7", false},   // Roi second
		{"Q2", false},   // Dame seconde
		{"Q965", false}, // Dame quatrième
		{"85", false},   // doubleton sans honneur
		{"7654", false}, // rien
	}
	e := &Engine{}
	for _, c := range cases {
		p := &playerState{hand: hand("", "", c.holding, "")}
		if got := e.hasHelp(p, Diamonds); got != c.want {
			t.Errorf("%q : hasHelp = %v, want %v", c.holding, got, c.want)
		}
	}
}
