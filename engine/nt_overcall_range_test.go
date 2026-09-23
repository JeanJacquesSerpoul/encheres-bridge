package main

import "testing"

// overcallOver1D sets up the direct seat over an opposing 1D opening and
// returns what the engine bids.
func overcallOver1D(h *Hand) (Call, meaning) {
	e := &Engine{opener: 3, openCall: bid(1, SDiamonds)}
	for seat := range 4 {
		e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
	}
	e.ps[0].hand = h
	e.ps[3].bids = 1
	e.calls = []SeatCall{{Seat: 3, Call: bid(1, SDiamonds),
		M: m(12, 23, "ouverture mineure", "minor opening").withLen(Diamonds, 3)}}
	return e.decide(0)
}

// TestNTOvercallRange pins [I-4]: over a one-level opening, 1NT is 16-18 H,
// regular, with a stopper in their suit. The band sits one point above the
// 1NT opening's own 15-17, the overcall being made in front of a hand that has
// already announced an opening.
//
// The 18 is the point of the ordering. The "any shape" takeout double starts
// at 18 too, and being tried first it used to swallow every 18-count: the band
// announced a maximum the bid could never hold. 1NT is now asked first, so a
// hand that fits its three conditions gets the precise call; 19 and up go back
// to the double.
func TestNTOvercallRange(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hcp        int
		want       string
	}{
		{"15 H : sous la zone", "AQ32", "KJ4", "Q76", "QJ8", 15, "Contre"},
		{"16 H : plancher", "AQ32", "KJ4", "QJ6", "QJ8", 16, "1SA"},
		{"17 H", "AQ32", "KQ4", "QJ6", "QJ8", 17, "1SA"},
		{"18 H : plafond, avant le contre", "AQ32", "KQ4", "KJ6", "QJ8", 18, "1SA"},
		{"19 H : le contre reprend la main", "AQ32", "KQ4", "KQ6", "QJ8", 19, "Contre"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.H() != tc.hcp {
				t.Fatalf("la main vaut %d H, le cas en annonce %d", h.H(), tc.hcp)
			}
			c, mn := overcallOver1D(h)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("intervention = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if tc.want == "1SA" {
				if mn.minPts != 16 || mn.maxPts != 18 {
					t.Fatalf("zone annoncée %d-%d, attendue 16-18 (%s)", mn.minPts, mn.maxPts, mn.fr)
				}
				if !mn.stops[Diamonds] {
					t.Fatalf("l'arrêt à Carreau n'est pas enregistré (%s)", mn.fr)
				}
			}
		})
	}
}

// TestNTOvercallConditions pins the two conditions the band does not carry:
// the stopper in their suit, and the regular shape. Both are checked in the
// middle of the zone, where the point count alone would say yes.
func TestNTOvercallConditions(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		want       string
	}{
		{"arrêt = Dame troisième", "AQ32", "KJ4", "QJ6", "QJ8", "1SA"},
		{"arrêt = Valet quatrième", "AQ32", "AJ4", "J643", "KJ", "1SA"},
		{"trois petites : pas d'arrêt", "AQ32", "AQ4", "643", "KJ98", "Contre"},
		{"chicane dans leur couleur : main irrégulière", "AQ432", "KJ42", "", "AQ98", "1P"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := hand(tc.s, tc.h, tc.d, tc.c)
			if h.H() != 16 {
				t.Fatalf("la main vaut %d H, le cas veut 16", h.H())
			}
			c, mn := overcallOver1D(h)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("intervention = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
		})
	}
}
