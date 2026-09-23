package engine

import "testing"

// TestWeakTwoShape pins what a weak two major must look like, independently of
// the point count: exactly six cards of real quality, no five-card side suit,
// no four cards in the other major, and at most one defensive trick outside.
func TestWeakTwoShape(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		want       bool
	}{
		{"DV9xxx, rien à côté", "QJ9432", "543", "876", "2", true},
		{"VT9xxx : la forme sans la couleur", "JT9432", "543", "876", "2", false},
		{"Axxxxx : un seul honneur", "A98432", "543", "876", "2", false},
		{"mineure quatrième : tolérée", "QJ9432", "54", "8", "K543", true},
		{"mineure cinquième : bicolore", "QJ9432", "5", "8", "K5432", false},
		{"autre majeure quatrième", "QJ9432", "K543", "8", "54", false},
		{"une levée de défense extérieure", "QJ9432", "A54", "876", "2", true},
		{"deux levées de défense extérieures", "QJ9432", "A54", "K76", "2", false},
		{"septième : ce n'est plus un 2 faible", "QJ97432", "54", "876", "2", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if got := weakTwoShape(h, Spades); got != tc.want {
				t.Fatalf("weakTwoShape = %v, attendu %v", got, tc.want)
			}
		})
	}
}

// openingWith returns the opening call of a hand sitting after n passes.
func openingWith(h *Hand, passes int) (Call, meaning) {
	e := &Engine{opener: -1}
	for seat := range 4 {
		e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
	}
	seat := passes % 4
	e.ps[seat].hand = h
	for i := range passes {
		e.calls = append(e.calls, SeatCall{Seat: i, Call: passCall})
	}
	return e.opening(e.ps[seat])
}

// TestWeakTwoOpeningZone pins the strength band and the seat rule. The floor
// moves from 5 H to 6, and fourth seat never preempts: three passes have gone
// round, there is nobody left to bar.
func TestWeakTwoOpeningZone(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hcp        int
		passes     int
		want       string
	}{
		{"5 H : sous le plancher", "QJ9432", "Q54", "876", "2", 5, 0, "Passe"},
		{"6 H : le 2 faible", "KJ9432", "Q54", "876", "2", 6, 0, "2P"},
		{"6 H en deuxième position", "KJ9432", "Q54", "876", "2", 6, 1, "2P"},
		{"6 H en troisième position", "KJ9432", "Q54", "876", "2", 6, 2, "2P"},
		{"6 H en quatrième position : personne à barrer", "KJ9432", "Q54", "876", "2", 6, 3, "Passe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.H() != tc.hcp {
				t.Fatalf("la main vaut %d H, le cas en annonce %d", h.H(), tc.hcp)
			}
			c, mn := openingWith(h, tc.passes)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("ouverture = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if tc.want == "2P" && (mn.minPts != 6 || mn.maxPts != 11) {
				t.Fatalf("zone annoncée %d-%d, attendue 6-11 (%s)", mn.minPts, mn.maxPts, mn.fr)
			}
		})
	}
}
