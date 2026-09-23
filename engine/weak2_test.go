package main

import (
	"strings"
	"testing"
)

// TestRespondWeak2 exercises the responder's table over a weak 2H/2S
// opening in the absence of interference (docs/addon_7.md): four trumps or
// more is always a direct raise to game ("attaque-défense"), regardless of
// point count; with only 2-3 trumps, 18-20 HLD raises to game directly,
// while 15-17 HLD (game possible) or 21+ HLD (slam possible) go through the
// 2NT relay-and-fit-ask instead; below that, the fit-based barrage table
// applies.
func TestRespondWeak2(t *testing.T) {
	const opener, responder = 2, 0
	cases := []struct {
		name       string
		M          Suit
		respHand   *Hand
		want       string
		wantphrase string
	}{
		{
			// Four trumps and 17 HLD: previously this fell into the 2NT
			// relay (sup>=3 && hld>=15); the "attaque-défense" principle
			// requires a direct raise to game instead.
			name:       "4 atouts et 17HLD : 4C direct malgre les points",
			M:          Hearts,
			respHand:   hand("AKQ2", "KJ87", "95", "K54"),
			want:       "4C",
			wantphrase: "attaque-défense",
		},
		{
			name:       "3 atouts et 18HLD : 4P direct, manche acquise",
			M:          Spades,
			respHand:   hand("K76", "AKQ2", "AJ98", "54"),
			want:       "4P",
			wantphrase: "manche directe en attaque",
		},
		{
			name:       "2 atouts et 19HLD : 4C direct, manche acquise",
			M:          Hearts,
			respHand:   hand("AK987", "K5", "AQJ4", "32"),
			want:       "4C",
			wantphrase: "manche directe en attaque",
		},
		{
			name:       "3 atouts et 17HLD : 2SA relais, manche possible",
			M:          Spades,
			respHand:   hand("KQ7", "AJ98", "AK4", "432"),
			want:       "2SA",
			wantphrase: "relais-fitté",
		},
		{
			name:       "3 atouts et 21HLD : 2SA relais, chelem possible",
			M:          Hearts,
			respHand:   hand("AKQ2", "K76", "AKJ4", "32"),
			want:       "2SA",
			wantphrase: "relais-fitté",
		},
		{
			name:       "3 atouts et jeu faible : prolongement du barrage",
			M:          Spades,
			respHand:   hand("K76", "9542", "873", "432"),
			want:       "3P",
			wantphrase: "prolongement du barrage",
		},
		{
			name:       "2 atouts et jeu faible : passe",
			M:          Hearts,
			respHand:   hand("9542", "76", "8763", "432"),
			want:       "Passe",
			wantphrase: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := scaffoldRencontre(opener, bidSuit(2, tc.M))
			e.ps[responder].hand = tc.respHand
			c, mn := e.respondWeak2(e.ps[responder], tc.M)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("answer = %s (%s), want %s", got, mn.fr, tc.want)
			}
			if tc.wantphrase != "" && !strings.Contains(mn.fr, tc.wantphrase) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.wantphrase)
			}
		})
	}
}

// TestWeak2Feature exercises the opener's answer to the 2NT relay-and-fit-ask
// (docs/addon_7.md, `weak2Feature`): a minimum (9-11 HLD) repeats the suit;
// otherwise (12-14 HLD) an outside ace or king takes priority, a maximum
// with a singleton announces it directly at the 4-level (with the
// 2H-specific 4H = singleton spade quirk), and 3NT covers the rest.
func TestWeak2Feature(t *testing.T) {
	const opener, responder = 2, 0
	cases := []struct {
		name       string
		M          Suit
		openHand   *Hand
		want       string
		wantphrase string
	}{
		{
			name:       "9-11HLD : reponse minimale, repete la couleur",
			M:          Hearts,
			openHand:   hand("543", "QJ9876", "432", "3"),
			want:       "3C",
			wantphrase: "minimal",
		},
		{
			name:       "12-14HLD, force exterieure : priorite a la mention",
			M:          Hearts,
			openHand:   hand("43", "AQJ876", "K54", "32"),
			want:       "3K",
			wantphrase: "force extérieure",
		},
		{
			name:       "12-14HLD, maximum et singleton : annonce directe (exemple du texte)",
			M:          Hearts,
			openHand:   hand("532", "AKQ876", "4", "532"),
			want:       "4K",
			wantphrase: "singleton",
		},
		{
			name:       "12-14HLD, singleton Pique apres 2C : 4C special",
			M:          Hearts,
			openHand:   hand("4", "AKQ876", "532", "532"),
			want:       "4C",
			wantphrase: "singleton Pique",
		},
		{
			name:       "12-14HLD, ni force ni singleton : 3SA",
			M:          Hearts,
			openHand:   hand("543", "AKQ876", "54", "32"),
			want:       "3SA",
			wantphrase: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := scaffoldRencontre(opener, bidSuit(2, tc.M),
				SeatCall{Seat: responder, Call: bid(2, SNoTrump), M: noInfo()})
			e.ps[opener].hand = tc.openHand
			c, mn := e.weak2Feature(e.ps[opener], tc.M)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("answer = %s (%s), want %s", got, mn.fr, tc.want)
			}
			if tc.wantphrase != "" && !strings.Contains(mn.fr, tc.wantphrase) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.wantphrase)
			}
		})
	}
}
