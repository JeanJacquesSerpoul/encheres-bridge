package main

import "testing"

// rebidOver1S1H sets up 1H - 1S and returns opener's rebid. Opener's second
// suit is clubs, cheaper than the opening: the "bicolore économique" seat.
func rebidOver1H1S(h *Hand, gameForce bool) (Call, meaning) {
	e := &Engine{opener: 0, openCall: bid(1, SHearts)}
	for seat := range 4 {
		e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
	}
	e.ps[0].hand = h
	e.ps[0].bids = 1
	e.ps[2].bids = 1
	e.ps[2].responded = true
	e.ps[2].shownMin, e.ps[2].shownMax = 6, 40
	e.ps[2].shownLens[Spades] = 4
	e.calls = []SeatCall{
		{Seat: 0, Call: bid(1, SHearts), M: m(12, 23, "ouverture majeure", "").withLen(Hearts, 5)},
		{Seat: 1, Call: passCall},
		{Seat: 2, Call: bid(1, SSpades), M: m(6, 40, "réponse 1 sur 1", "").withLen(Spades, 4).asForcing()},
		{Seat: 3, Call: passCall},
	}
	e.ps[2].lastM = &e.calls[2].M
	e.gameForce[0] = gameForce
	return e.openerRebid(e.ps[0])
}

// TestBicoloreZones pins [RO-19]: the cheap second suit stops at 17, and from
// 18 the same two suits are shown one level higher. The cheap bid is one
// partner may pass, so it cannot also carry the hands that want to hear from
// him -- before the split, a nineteen-count and a twelve-count made the same
// call and the responder had no way to tell them apart.
func TestBicoloreZones(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hl         int
		want       string
		forcing    bool
	}{
		{"15 HL : économique", "4", "KQ932", "K73", "AQ86", 15, "2T", false},
		{"17 HL : plafond de l'économique", "4", "KQJ32", "KQ3", "AJ86", 17, "2T", false},
		{"18 HL : le saut", "4", "KQJ32", "KQ3", "AQ86", 18, "3T", true},
		{"19 HL : le saut", "4", "AQJ32", "KQ3", "AQ86", 19, "3T", true},
		{"22 HL : le saut", "4", "AKQ32", "AQ3", "AQ86", 22, "3T", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.HL() != tc.hl {
				t.Fatalf("la main vaut %d HL, le cas en annonce %d", h.HL(), tc.hl)
			}
			c, mn := rebidOver1H1S(h, false)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("redemande = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if mn.forcing != tc.forcing {
				t.Fatalf("forcing = %v, attendu %v (%s)", mn.forcing, tc.forcing, mn.fr)
			}
			if tc.want == "2T" && (mn.minPts != 12 || mn.maxPts != 17) {
				t.Fatalf("zone annoncée %d-%d, attendue 12-17 (%s)", mn.minPts, mn.maxPts, mn.fr)
			}
			if tc.want == "3T" && mn.minPts != 18 {
				t.Fatalf("plancher annoncé %d, attendu 18 (%s)", mn.minPts, mn.fr)
			}
		})
	}
}

// TestBicoloreNoJumpUnderGameForce guards the exception: once the auction is
// forced to game, partner cannot pass the cheap bid, so it already carries the
// whole range and the level the jump would spend is room the slam exploration
// needs. The bid must then announce no ceiling -- a 12-17 label on a hand of
// 18 would tell partner the opposite of the truth.
func TestBicoloreNoJumpUnderGameForce(t *testing.T) {
	h := hand("4", "AQJ32", "KQ3", "AQ86") // 19 HL
	c, mn := rebidOver1H1S(h, true)
	if got := c.Format("fr"); got != "2T" {
		t.Fatalf("redemande = %s (%s), attendu 2T : le saut brûle le palier dont le chelem a besoin",
			got, mn.fr)
	}
	if mn.maxPts >= 0 {
		t.Fatalf("l'enchère annonce un plafond de %d (%s) alors qu'elle ne limite rien",
			mn.maxPts, mn.fr)
	}
}

// TestBicoloreCherUnchanged: the reverse, where the second suit ranks above
// the opening, keeps its own threshold of 18 HL.
func TestBicoloreCherUnchanged(t *testing.T) {
	rebidOver1D1S := func(h *Hand) (Call, meaning) {
		e := &Engine{opener: 0, openCall: bid(1, SDiamonds)}
		for seat := range 4 {
			e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
		}
		e.ps[0].hand = h
		e.ps[0].bids = 1
		e.ps[2].bids = 1
		e.ps[2].responded = true
		e.ps[2].shownMin, e.ps[2].shownMax = 6, 40
		e.ps[2].shownLens[Spades] = 4
		e.calls = []SeatCall{
			{Seat: 0, Call: bid(1, SDiamonds), M: m(12, 23, "ouverture mineure", "").withLen(Diamonds, 3)},
			{Seat: 1, Call: passCall},
			{Seat: 2, Call: bid(1, SSpades), M: m(6, 40, "réponse 1 sur 1", "").withLen(Spades, 4).asForcing()},
			{Seat: 3, Call: passCall},
		}
		e.ps[2].lastM = &e.calls[2].M
		return e.openerRebid(e.ps[0])
	}
	cases := []struct {
		name       string
		s, h, d, c string
		hl         int
		want       string
	}{
		{"17 HL : pas encore", "4", "AQ86", "AQJ32", "K73", 17, "2K"},
		{"18 HL : le bicolore cher", "4", "AQ86", "AKJ32", "K73", 18, "2C"},
		{"19 HL", "4", "AQ86", "AKQ32", "K73", 19, "2C"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.HL() != tc.hl {
				t.Fatalf("la main vaut %d HL, le cas en annonce %d", h.HL(), tc.hl)
			}
			c, mn := rebidOver1D1S(h)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("redemande = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
		})
	}
}

// TestBicoloreOverNotrumpResponse: the same rule at the other seat. Opener's
// rebid after 1S - 1NT ("poubelle", 6-10) goes through a different branch,
// which capped the cheap bid at 17 but had no jump above it -- a 5-4 hand of
// 18-19 HL announced 12-17 while holding more (audit du par, donne 22).
func TestBicoloreOverNotrumpResponse(t *testing.T) {
	rebidOver1S1NT := func(h *Hand) (Call, meaning) {
		e := &Engine{opener: 0, openCall: bid(1, SSpades)}
		for seat := range 4 {
			e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
		}
		e.ps[0].hand = h
		e.ps[0].bids = 1
		e.ps[2].bids = 1
		e.ps[2].responded = true
		e.ps[2].shownMin, e.ps[2].shownMax = 6, 10
		e.calls = []SeatCall{
			{Seat: 0, Call: bid(1, SSpades), M: m(12, 23, "ouverture majeure", "").withLen(Spades, 5)},
			{Seat: 1, Call: passCall},
			{Seat: 2, Call: bid(1, SNoTrump), M: m(6, 10, "1SA \"poubelle\"", "")},
			{Seat: 3, Call: passCall},
		}
		e.ps[2].lastM = &e.calls[2].M
		return e.openerRebid(e.ps[0])
	}
	cases := []struct {
		name       string
		s, h, d, c string
		hl         int
		want       string
	}{
		{"15 HL : économique", "KQ864", "AQ83", "4", "K73", 15, "2C"},
		{"19 HL : le saut", "AQ864", "AJ83", "A", "K73", 19, "3C"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.HL() != tc.hl {
				t.Fatalf("la main vaut %d HL, le cas en annonce %d", h.HL(), tc.hl)
			}
			c, mn := rebidOver1S1NT(h)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("redemande = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if tc.want == "2C" && mn.maxPts != 17 {
				t.Fatalf("plafond annoncé %d, attendu 17 (%s)", mn.maxPts, mn.fr)
			}
			if tc.want == "3C" && mn.minPts != 18 {
				t.Fatalf("plancher annoncé %d, attendu 18 (%s)", mn.minPts, mn.fr)
			}
		})
	}
}
