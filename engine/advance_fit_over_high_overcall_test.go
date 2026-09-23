package engine

import (
	"strings"
	"testing"
)

// TestAdvanceEightCardFitOverThreeLevelOvercall reproduces the reported deal.
// South preempts 3C (seven cards), West overcalls 3H -- which promises six
// cards and 12 HL [I-5] -- and North passes. East holds KJT65 76 A93 A86:
// 12 H, 13 HLD, and a heart doubleton that makes an eight-card fit opposite
// the promised six.
//
// East used to pass, on two counts: [A-5]'s "3 atouts" was read as three cards
// in East's own hand (a count taken against the five a one-level overcall
// promises), and the strength ask [A-6] refused to cue-bid above the three
// level. Two opening hands facing each other went unheard and the pair played
// 3H. East must speak, and the cue-bid is affordable precisely because the
// coded sign-off -- West's return to hearts -- is the game and nothing more.
func TestAdvanceEightCardFitOverThreeLevelOvercall(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:93.82.Q2.QJ95432 8.AKJT953.KJ54.7 AQ742.Q4.T876.KT KJT65.76.A93.A86"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var ask *SeatCall
	for i := range calls {
		if calls[i].Seat == east && calls[i].M.overcallAsk {
			ask = &calls[i]
			break
		}
	}
	if ask == nil {
		t.Fatalf("East never asked how strong the overcall was\nauction: %s", formatAuction(calls))
	}
	if ask.Call != bidSuit(4, Clubs) {
		t.Fatalf("East's ask = %s, want 4T (cue-bid of the opener's suit)\nauction: %s",
			ask.Call.Format("fr"), formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract.Level < 4 || contract.Strain != SHearts {
		t.Fatalf("final contract = %s, want at least the heart game\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestAdvanceNoAskOnSevenCardFit guards the other side of the same rule: the
// eight-card fit is measured, not waived. Opposite a one-level overcall --
// five cards promised -- a doubleton still makes only seven, and the ask stays
// off however strong the advancing hand is [A-6]. Raising the promise to six
// makes the same hand ask.
func TestAdvanceNoAskOnSevenCardFit(t *testing.T) {
	// West opens 1C, North overcalls 1H, East passes, South advances.
	const north, east, south, west = 0, 1, 2, 3
	ask := func(shownLen int) (Call, meaning) {
		e := &Engine{opener: west}
		for i := 0; i < 4; i++ {
			e.ps[i] = &playerState{seat: i, hand: &Hand{}, shownMax: 40}
		}
		// 13 HLD, two hearts -- and no club stopper, so the stopper-based
		// no-fit probe of [A-5] cannot fire and muddy the reading.
		e.ps[south].hand = hand("AQJ2", "76", "KQJ4", "932")
		e.ps[north].shownMin, e.ps[north].shownMax = 9, 18
		e.ps[north].shownLens[Hearts] = shownLen
		e.openCall = bidSuit(1, Clubs)
		e.calls = []SeatCall{
			{Seat: west, Call: bidSuit(1, Clubs)},
			{Seat: north, Call: bidSuit(1, Hearts)},
			{Seat: east, Call: passCall},
		}
		return e.advance(e.ps[south])
	}

	if c, mn := ask(5); mn.overcallAsk {
		t.Fatalf("asked the overcall's strength (%s) on a seven-card fit", c.Format("fr"))
	}
	// The very same hand, opposite a promised sixth card, holds the eight-card
	// fit the rule is really about -- and must ask.
	c, mn := ask(6)
	if !mn.overcallAsk {
		t.Fatalf("did not ask on an eight-card fit: %s (%s)", c.Format("fr"), mn.fr)
	}
	if c != bidSuit(2, Clubs) {
		t.Fatalf("ask = %s, want 2T (cue-bid of the opener's suit)", c.Format("fr"))
	}
	if !strings.Contains(mn.fr, "fit") {
		t.Fatalf("comment %q should be the with-a-fit ask, not the no-fit probe", mn.fr)
	}
}
