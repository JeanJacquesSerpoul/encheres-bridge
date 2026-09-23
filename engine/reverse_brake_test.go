package engine

import (
	"strings"
	"testing"
)

// afterReverse puts responder in front of opener's reverse: 1C - 1S - 2D,
// the "bicolore cher" being auto-forcing and worth 18 HL and up.
func afterReverse(h *Hand) (Call, meaning) {
	e := &Engine{opener: 0, openCall: bid(1, SClubs)}
	for seat := range 4 {
		e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
	}
	e.ps[2].hand = h
	e.ps[0].bids = 2
	e.ps[0].shownMin, e.ps[0].shownMax = 18, 23
	e.ps[0].shownLens[Clubs], e.ps[0].shownLens[Diamonds] = 4, 4
	e.ps[2].bids = 1
	e.ps[2].responded = true
	e.ps[2].shownMin, e.ps[2].shownMax = 6, 40
	e.ps[2].shownLens[Spades] = 4
	rev := m(18, 23, "bicolore cher, forcing", "reverse, forcing").
		withLen(Diamonds, 4).withLen(Clubs, 4).asForcing()
	rev.reverse = true
	e.calls = []SeatCall{
		{Seat: 0, Call: bid(1, SClubs), M: m(12, 23, "ouverture mineure", "").withLen(Clubs, 3)},
		{Seat: 1, Call: passCall},
		{Seat: 2, Call: bid(1, SSpades), M: m(6, 40, "réponse 1 sur 1", "").withLen(Spades, 4).asForcing()},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bid(2, SDiamonds), M: rev},
		{Seat: 1, Call: passCall},
	}
	e.ps[0].lastM = &e.calls[4].M
	return e.decide(2)
}

// TestReverseBrake pins the two answers to opener's reverse that are not game
// forcing [RO-19b]. The reverse announces 18 as a floor, and the generic count
// reads a floor as licence to bid the game: a hand of 5-7 H had no way to say
// otherwise and drove the side to 3SA on nothing.
func TestReverseBrake(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hcp        int
		want       string
		forcing    bool
		hint       string
	}{
		{"5 H : le coup de frein", "AJ72", "643", "765", "543", 5, "2SA", true, "modérateur"},
		{"7 H : plafond du frein", "AJ72", "643", "765", "Q54", 7, "2SA", true, "modérateur"},
		{"5 H, majeure cinquième : la priorité", "AJ762", "643", "765", "54", 5, "2P", true, "cinquième"},
		{"8 H : au-dessus du frein, forcing de manche", "AJ72", "K43", "765", "543", 8, "3SA", false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.H() != tc.hcp {
				t.Fatalf("la main vaut %d H, le cas en annonce %d", h.H(), tc.hcp)
			}
			c, mn := afterReverse(h)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("réponse au bicolore cher = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if mn.forcing != tc.forcing {
				t.Fatalf("forcing = %v, attendu %v (%s)", mn.forcing, tc.forcing, mn.fr)
			}
			if tc.hint != "" && !strings.Contains(mn.fr, tc.hint) {
				t.Fatalf("le commentaire %q ne mentionne pas %q", mn.fr, tc.hint)
			}
			if tc.want == "2SA" && mn.maxPts != 7 {
				t.Fatalf("plafond annoncé %d, attendu 7 (%s)", mn.maxPts, mn.fr)
			}
		})
	}
}

// TestReverseBrakeAnswered replays the reference auction whole: 1C - 1S - 2D -
// 2SA - 3C, passed out. A brake nobody answers is worse than no brake at all,
// so opener must retreat when the combined maximum -- partner capped at 7 --
// does not reach the game.
func TestReverseBrakeAnswered(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "None"]
[Deal "N:65.5.AK98.AKQT92 KJ9.KJ98.Q32.J87 AT72.Q64.765.543 Q843.AT732.JT4.6"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	want := []string{"1T", "Passe", "1P", "Passe", "2K", "Passe", "2SA", "Passe", "3T", "Passe", "Passe", "Passe"}
	if len(calls) != len(want) {
		t.Fatalf("%d enchères, attendu %d\nauction: %s", len(calls), len(want), formatAuction(calls))
	}
	for i, w := range want {
		if got := calls[i].Call.Format("fr"); got != w {
			t.Fatalf("enchère #%d = %s (%s), attendu %s\nauction: %s",
				i, got, calls[i].M.fr, w, formatAuction(calls))
		}
	}
	c, decl, _ := finalContract(calls)
	if got := c.Format("fr"); got != "3T" || decl != 0 {
		t.Fatalf("contrat = %s par %s, attendu 3T par N\nauction: %s",
			got, seatNames[decl], formatAuction(calls))
	}
}
