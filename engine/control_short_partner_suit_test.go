package main

import (
	"strings"
	"testing"
)

// TestControlBidShortInPartnerSuitStillCheapestFirst locks in that economic
// order has no exception for a short-suit control sitting in partner's own
// suit. South, spades agreed, holds a singleton club opposite North's shown
// club length plus the heart ace. The cheapest control is the club singleton:
// South must cue 4C (4♣) first. Cueing 4H straight away would skip clubs and
// thereby deny the very control South holds [S-2b].
func TestControlBidShortInPartnerSuitStillCheapestFirst(t *testing.T) {
	south := hand("AKQJ2", "A32", "432", "3") // heart control, singleton club
	north := hand("543", "K76", "A76", "KJ982")
	e := scaffoldControlEngine(south, north, 16)
	e.ps[0].shownLens[Clubs] = 5 // North has named his clubs

	c, mn, ok := e.controlBid(e.ps[2], Spades)
	if !ok {
		t.Fatalf("expected a control bid, got none")
	}
	if got := c.Format("fr"); got != "4T" {
		t.Fatalf("first cue = %s (%s), want 4T (club singleton, the cheapest control)", got, mn.fr)
	}
	if mn.controlSuit != Clubs {
		t.Fatalf("controlSuit = %v, want Clubs", mn.controlSuit)
	}
	if !strings.Contains(mn.fr, "singleton") {
		t.Fatalf("comment %q should name the singleton", mn.fr)
	}
}
