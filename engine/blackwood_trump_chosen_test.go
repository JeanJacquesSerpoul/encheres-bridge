package main

import (
	"strings"
	"testing"
)

// TestBlackwoodNeedsTheTrumpChosen replays a reported deal where East answered
// "two keycards and the trump queen" about a suit he had never named. West
// overcalled 2C over South's 1H opening; East cue-bid 2H, the [A-6] question
// about the overcall's strength, which does promise three clubs -- and West
// answered it by describing a second suit, 2S. From there the side had two
// suits of its own on the table and East had chosen neither, yet the control
// exchange ran on to 4NT and East's 5S counted the club queen as the trump
// queen of a suit nobody had agreed [S-4b].
//
// The promise a cue-bid carries is real but it proposes nothing: it answers
// the question it asked, not the question of where to play. Until partner
// names his preference the trump is not chosen, so no keycard ask -- West
// names the suit himself instead.
func TestBlackwoodNeedsTheTrumpChosen(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "None"]
[Deal "S:KQ.KQ763.KQT5.J3 JT73.A8.3.AK9642 98642.J542.J2.T5 A5.T9.A98764.Q87"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.M.blackwood {
			t.Fatalf("%s asked %s (%s) with the trump still unchosen\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		if sc.M.keyResp || strings.Contains(sc.M.fr, "Dame d'atout") {
			t.Fatalf("%s answered %s (%s) about a trump nobody named\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}

	// The exploration still ends where it should: West, out of controls to
	// show, names the club game himself -- putting the trump on the table at
	// last -- instead of a slam bid on an imaginary agreement.
	const west = 3
	named := false
	for _, sc := range calls {
		if sc.Seat == west && sc.Call.IsBid() && sc.Call.Strain == SClubs && sc.Call.Level >= 5 {
			named = true
		}
	}
	if !named {
		t.Fatalf("West never named his own suit to settle the trump\nauction: %s", formatAuction(calls))
	}
}
