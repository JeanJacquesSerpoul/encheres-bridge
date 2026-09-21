package main

import (
	"strings"
	"testing"
)

// TestStrongTwoInterference checks the defense after an opponent intervenes
// directly over the strong, artificial 2C/2D opening, before the forced
// relay (docs/bidings.md, "LA DÉFENSE APRÈS UNE INTERVENTION DU N°2"). The
// opening promises nothing about its own suit, so responder's suit bids here
// are fully natural (5+ good cards, 5H+); the double shows the same floor
// without a good natural bid; below 5H there is nothing to say.
//
// East = A 7 . A J 9 5 3 2 . 6 . A K Q 9 (18H, strong 2C) opens 2C; South
// overcalls 2H.
func TestStrongTwoInterference(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3

	newEngine := func() *Engine {
		e := &Engine{opener: east, openCall: bid(2, SClubs)}
		e.calls = []SeatCall{
			{Seat: east, Call: bid(2, SClubs)},
			{Seat: south, Call: bid(2, SHearts)},
		}
		return e
	}

	t.Run("cinq belles cartes : naturelle, pas un soutien", func(t *testing.T) {
		// West = Q T 5 . (void) . J T 6 4 2 . Q J T 4 2: the deal that
		// exposed the bug -- 6H, two five-card minors, void in the
		// opponent's suit. Must not be read as "supporting clubs".
		h := hand("QT5", "", "JT642", "QJT42")
		e := newEngine()
		p := &playerState{seat: west, hand: h}
		c, mn := e.respond(p)

		if strings.Contains(mn.fr, "soutien") {
			t.Fatalf("comment %q reads as support for the artificial 2C suit", mn.fr)
		}
		if !strings.Contains(mn.fr, "naturelle") {
			t.Fatalf("comment %q does not read as a natural bid", mn.fr)
		}
		if !mn.forcing {
			t.Fatalf("natural response over a strong 2C opening must stay forcing")
		}
		if c.Kind != KindBid || c.Strain == SHearts {
			t.Fatalf("West bid %s, want a natural minor (not the opponents' hearts)", c.Format("fr"))
		}
	})

	t.Run("5H et plus sans belle couleur : Contre", func(t *testing.T) {
		h := hand("KJ32", "7", "QJ32", "9865")
		if got := h.H(); got < 5 {
			t.Fatalf("test hand has %dH, want 5+", got)
		}
		e := newEngine()
		p := &playerState{seat: west, hand: h}
		c, mn := e.respond(p)

		if c.Kind != KindDouble {
			t.Fatalf("West responds %s (%s), want Contre (no good natural suit)", c.Format("fr"), mn.fr)
		}
		if !mn.forcing {
			t.Fatalf("the double must stay forcing opposite a strong 2C opening")
		}
	})

	t.Run("moins de 5H : passe", func(t *testing.T) {
		h := hand("432", "432", "432", "432")
		if got := h.H(); got >= 5 {
			t.Fatalf("test hand has %dH, want under 5", got)
		}
		e := newEngine()
		p := &playerState{seat: west, hand: h}
		c, _ := e.respond(p)

		if c.Kind != KindPass {
			t.Fatalf("West responds %s, want Passe below 5H", c.Format("fr"))
		}
	})
}
