package engine

import "testing"

// TestHelpSuitTryNeedsTwoCovers: the help-suit try asks partner to cover the
// losers of a suit with no ace or king -- two of them at least -- so the
// help must cover two [C-19]. 1H - (2C) - 2H - 3D: South holds A2 in
// diamonds, one honour and a doubleton, a single loser covered. South
// declines with 3H instead of accepting the game.
func TestHelpSuitTryNeedsTwoCovers(t *testing.T) {
	d, err := ParsePBN([]byte(`[Vulnerable "None"]
[Deal "S:754.T94.A2.K8753 QT862.53.JT87.T9 KJ9.AKQJ2.9643.Q A3.876.KQ5.AJ642"]
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

// TestHasHelpCountsCovers pins the definition: two honours, or a singleton or
// void; one honour or a doubleton alone is not enough.
func TestHasHelpCountsCovers(t *testing.T) {
	cases := []struct {
		holding string
		want    bool
	}{
		{"AK2", true},   // deux honneurs
		{"AQ", true},    // deux honneurs, doubleton
		{"KQJ3", true},  // trois honneurs
		{"Q32", false},  // un seul honneur
		{"K532", false}, // un seul honneur, quatre cartes
		{"A2", false},   // un honneur et un doubleton
		{"85", false},   // doubleton sans honneur
		{"7654", false}, // rien
		{"2", true},     // singleton
		{"", true},      // chicane
	}
	e := &Engine{}
	for _, c := range cases {
		p := &playerState{hand: hand("", "", c.holding, "")}
		if got := e.hasHelp(p, Diamonds); got != c.want {
			t.Errorf("%q : hasHelp = %v, want %v", c.holding, got, c.want)
		}
	}
}
