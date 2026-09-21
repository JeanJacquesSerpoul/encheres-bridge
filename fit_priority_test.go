package main

import (
	"strings"
	"testing"
)

// TestThreeCardFitBeforeFourCardSpades checks that a genuine fit (three-card
// support, adequate per the sup >= 3 block below) is never delayed by a
// four-card spade suit over a 1H opening: with a weak hand there is no
// reason to detour through the new suit first when the direct raise already
// values the hand correctly (soutien simple, 6-10 HLD). Only a real shortage
// of support (0-2 cards) still routes through spades first.
//
// South = Q T 9 3 2 . Q 8 5 . 6 3 . J 7 6 (5H, 6 HLD, three-card heart
// support and a five-card spade suit) must raise to 2H, not detour through
// 1S.
func TestThreeCardFitBeforeFourCardSpades(t *testing.T) {
	const north, south = 0, 2

	h := hand("QT932", "Q85", "63", "J76")
	if got := h.Len(Hearts); got != 3 {
		t.Fatalf("test hand has %d hearts, want exactly 3", got)
	}
	if got := h.HLD(Hearts); got < 6 || got > 10 {
		t.Fatalf("test hand HLD(hearts) = %d, want 6-10 (soutien simple zone)", got)
	}

	e := &Engine{opener: north, openCall: bid(1, Hearts.Strain()), calls: []SeatCall{{Seat: north, Call: bid(1, Hearts.Strain())}}}
	p := &playerState{seat: south, hand: h}
	c, mn := e.respondMajor(p, Hearts)

	if got := c.Format("fr"); got != "2C" {
		t.Fatalf("South responds %s (%s), want 2C (raise, no reason to delay the fit)\n", got, mn.fr)
	}
	if !strings.Contains(mn.fr, "soutien simple") {
		t.Fatalf("comment %q does not mention the simple raise", mn.fr)
	}
}

// TestShortOfSupportStillShowsSpadesFirst checks that the four-card spade
// priority still applies when support is genuinely short (0-2 cards, no
// fit to speak of).
func TestShortOfSupportStillShowsSpadesFirst(t *testing.T) {
	const north, south = 0, 2

	h := hand("QT932", "Q8", "6432", "J76")
	if got := h.Len(Hearts); got != 2 {
		t.Fatalf("test hand has %d hearts, want exactly 2", got)
	}

	e := &Engine{opener: north, openCall: bid(1, Hearts.Strain()), calls: []SeatCall{{Seat: north, Call: bid(1, Hearts.Strain())}}}
	p := &playerState{seat: south, hand: h}
	c, mn := e.respondMajor(p, Hearts)

	if got := c.Format("fr"); got != "1P" {
		t.Fatalf("South responds %s (%s), want 1P (no fit, shows the four-card spade suit)\n", got, mn.fr)
	}
}
