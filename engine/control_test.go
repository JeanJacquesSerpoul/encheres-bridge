package main

import (
	"strings"
	"testing"
)

// scaffoldControlEngine builds a minimal auction where South opened 1S and
// North raised to 3S (fit agreed below game, room left for 4-level cues),
// ready for South's control-bidding turn.
func scaffoldControlEngine(southHand, northHand *Hand, northShownMin int) *Engine {
	e := &Engine{opener: 2}
	e.ps[0] = &playerState{seat: 0, hand: northHand, shownMax: 40, shownMin: northShownMin}
	e.ps[1] = &playerState{seat: 1, hand: &Hand{}, shownMax: 40}
	e.ps[2] = &playerState{seat: 2, hand: southHand, shownMax: 40}
	e.ps[3] = &playerState{seat: 3, hand: &Hand{}, shownMax: 40}
	e.calls = []SeatCall{
		{Seat: 2, Call: bid(1, SSpades)},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bidSuit(3, Spades)},
		{Seat: 1, Call: passCall},
	}
	return e
}

// TestControlBidCheapest checks that the cheapest new suit holding a control
// (here the ace of clubs) is shown first (docs/addon_4.md).
func TestControlBidCheapest(t *testing.T) {
	south := hand("AKQJ2", "32", "32", "A32")
	e := scaffoldControlEngine(south, hand("543", "876", "876", "8765"), 14)
	c, mn, ok := e.controlBid(e.ps[2], Spades)
	if !ok {
		t.Fatalf("expected a control bid, got none")
	}
	if got := c.Format("fr"); got != "4T" {
		t.Fatalf("call = %s, want 4T (cheapest control, the ace of clubs)", got)
	}
	if !strings.Contains(mn.fr, "l'As") {
		t.Fatalf("comment %q should mention the ace", mn.fr)
	}
	if mn.controlSuit != Clubs || !mn.forcing {
		t.Fatalf("meaning = %+v, want controlSuit=Clubs and forcing", mn)
	}
}

// TestControlBidSkipDenies checks that a suit without any control is skipped
// (and recorded as denied), and that the next real control is bid instead.
func TestControlBidSkipDenies(t *testing.T) {
	south := hand("AKQJ2", "32", "A32", "432") // no club control, ace of diamonds
	e := scaffoldControlEngine(south, hand("543", "876", "876", "8765"), 14)
	c, mn, ok := e.controlBid(e.ps[2], Spades)
	if !ok {
		t.Fatalf("expected a control bid, got none")
	}
	if got := c.Format("fr"); got != "4K" {
		t.Fatalf("call = %s, want 4K (ace of diamonds, clubs denied)", got)
	}
	if !mn.deniedCtrl[Clubs] {
		t.Fatalf("meaning should deny clubs (skipped over), got %+v", mn.deniedCtrl)
	}
	if mn.controlSuit != Diamonds {
		t.Fatalf("controlSuit = %v, want Diamonds", mn.controlSuit)
	}
}

// TestControlBidNoneLeft checks that a hand with no control outside trump
// does not produce a control bid.
func TestControlBidNoneLeft(t *testing.T) {
	south := hand("AKQJ2", "432", "432", "432")
	e := scaffoldControlEngine(south, hand("543", "876", "876", "8765"), 14)
	if _, _, ok := e.controlBid(e.ps[2], Spades); ok {
		t.Fatalf("expected no control bid for a hand with no side-suit control")
	}
}

// TestContinueControlBidToBlackwood checks that once no further control can
// be shown, a combined count in the slam zone moves on to Blackwood (4NT is
// still reachable below the agreed trump's game level). With a major trump
// the zone is what makes the ask affordable: every answer sits above 4S, so
// a short one strands the pair in five [S-4].
func TestContinueControlBidToBlackwood(t *testing.T) {
	south := hand("AKQJ2", "432", "432", "432") // no more controls to show
	e := scaffoldControlEngine(south, hand("543", "876", "876", "8765"), 22)
	c, mn := e.continueControlBid(e.ps[2], Spades)
	if got := c.Format("fr"); got != "4SA" {
		t.Fatalf("call = %s, want 4SA (Blackwood)", got)
	}
	if !mn.blackwood {
		t.Fatalf("meaning should be flagged blackwood, got %+v", mn)
	}
}

// TestContinueControlBidSignsOff checks that with no more controls and
// insufficient combined values, the pair signs off in the agreed trump suit.
func TestContinueControlBidSignsOff(t *testing.T) {
	south := hand("AKQJ2", "432", "432", "432")
	e := scaffoldControlEngine(south, hand("543", "876", "876", "8765"), 8)
	c, mn := e.continueControlBid(e.ps[2], Spades)
	if got := c.Format("fr"); got != "4P" {
		t.Fatalf("call = %s, want 4P (sign-off in the agreed trump)", got)
	}
	if strings.Contains(mn.fr, "clefs") {
		t.Fatalf("sign-off should not ask for keycards, got %q", mn.fr)
	}
}
