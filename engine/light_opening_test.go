package engine

import (
	"strings"
	"testing"
)

// TestLightOpeningThirdSeat pins the third-seat opening below the [O-6]
// threshold: partner has passed, so the hand no longer bids for a game it
// cannot reach, but for the lead and for the level it costs the opponents.
// The price is a real five-card suit holding the honours of the hand.
func TestLightOpeningThirdSeat(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hcp        int
		want       string
		wantMin    int
	}{
		{"11 H, ADxxx à Pique : l'ouverture légère", "AQ843", "A4", "986", "J53", 11, "1P", 10},
		{"10 H, AVTxx : la couleur porte la main", "AJT43", "K4", "Q86", "543", 10, "1P", 10},
		{"9 H : sous le plancher", "AQ843", "K4", "9862", "54", 9, "Passe", 0},
		{"11 H mais les points dehors", "KJ843", "K4", "Q86", "Q53", 11, "Passe", 0},
		{"11 H, couleur cinquième médiocre", "98432", "AK4", "Q86", "Q5", 11, "Passe", 0},
		{"11 H sans couleur cinquième", "AQ83", "K42", "986", "Q53", 11, "Passe", 0},
		{"belle mineure cinquième : elle s'ouvre aussi", "K4", "986", "AQJ43", "J53", 11, "1K", 10},
		{"12 H : l'ouverture normale [O-6] passe devant", "AQ843", "A42", "986", "Q5", 12, "1P", 12},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hd := hand(tc.s, tc.h, tc.d, tc.c)
			if hd.H() != tc.hcp {
				t.Fatalf("la main vaut %d H, le cas en annonce %d", hd.H(), tc.hcp)
			}
			c, mn := openingWith(hd, 2)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("ouverture = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if tc.want != "Passe" && mn.minPts != tc.wantMin {
				t.Fatalf("plancher annoncé %d, attendu %d (%s)", mn.minPts, tc.wantMin, mn.fr)
			}
		})
	}
}

// TestLightOpeningThirdSeatZone pins what the light opening promises: the
// 10-11 zone it really holds, not the 12-23 of [O-6], and the five cards of
// the suit named. Partner, capped by his own opening pass [E-1b], must read a
// combined maximum that cannot invite.
func TestLightOpeningThirdSeatZone(t *testing.T) {
	c, mn := openingWith(hand("AQ843", "A4", "986", "J53"), 2)
	if got := c.Format("fr"); got != "1P" {
		t.Fatalf("ouverture = %s, attendu 1P", got)
	}
	if mn.minPts != 10 || mn.maxPts != 11 {
		t.Fatalf("zone annoncée %d-%d, attendue 10-11 (%s)", mn.minPts, mn.maxPts, mn.fr)
	}
	if mn.lens[Spades] != 5 {
		t.Fatalf("longueur promise à Pique = %d, attendue 5", mn.lens[Spades])
	}
}

// TestLightOpeningFourthSeat pins the rule of 15: in fourth seat there is no
// lead to direct and no partner to reach, only a partscore to buy -- and the
// spade length says who buys it. Below 15, passing the hand out scores zero,
// which beats a minus.
func TestLightOpeningFourthSeat(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hcp        int
		want       string
	}{
		{"11 H et 5 piques : 16, on ouvre", "AQ843", "A4", "986", "J53", 11, "1P"},
		{"11 H et 4 piques : 15, on ouvre", "KQ54", "A32", "J432", "J2", 11, "1K"},
		{"11 H et 3 piques : 14, on passe", "KQ5", "A32", "J4322", "J2", 11, "Passe"},
		{"10 H et 5 piques : 15, on ouvre", "AJT43", "K4", "Q862", "54", 10, "1P"},
		{"10 H et 4 piques : 14, on passe", "AJT4", "K43", "Q862", "54", 10, "Passe"},
		{"9 H et 6 piques : sous le plancher de 10 H", "AJT432", "K43", "986", "J", 9, "Passe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hd := hand(tc.s, tc.h, tc.d, tc.c)
			if hd.H() != tc.hcp {
				t.Fatalf("la main vaut %d H, le cas en annonce %d", hd.H(), tc.hcp)
			}
			c, mn := openingWith(hd, 3)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("ouverture = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
		})
	}
}

// TestLightOpeningOnlyLateSeats pins that the rule is a seat rule and nothing
// else: the very hand that opens in third seat passes in first and in second,
// where partner is still to speak and the règle des 20 governs.
func TestLightOpeningOnlyLateSeats(t *testing.T) {
	hd := hand("AQ843", "A4", "986", "J53")
	for _, passes := range []int{0, 1} {
		c, mn := openingWith(hd, passes)
		if got := c.Format("fr"); got != "Passe" {
			t.Fatalf("après %d passe(s) : ouverture = %s (%s), attendu Passe", passes, got, mn.fr)
		}
	}
}

// TestLightOpeningKeepsPreemptPriority pins the order: a hand that has a
// preempt keeps it. The barrage describes six cards at a level the light
// opening of one cannot buy, and [O-7a] already tuned its conditions.
func TestLightOpeningKeepsPreemptPriority(t *testing.T) {
	hd := hand("AK9432", "K54", "87", "65") // 10 H, six good cards
	c, mn := openingWith(hd, 2)
	if got := c.Format("fr"); got != "2P" {
		t.Fatalf("ouverture = %s (%s), attendu 2P", got, mn.fr)
	}
}

// TestLightOpenerStaysQuiet pins the discipline the light opening pays for:
// having borrowed a king it does not hold, the hand never bids again of its
// own accord. Partner is a passed hand, there is no game behind his answer,
// and a second bid is where the borrowed king turns a plus into a minus.
func TestLightOpenerStaysQuiet(t *testing.T) {
	// The deal that raised the question: West holds ADxxx to Pique and 11 H
	// in third seat, opens 1P for the lead, and passes East's 2K.
	pbn := `[Dealer "E"]
[Vulnerable "EW"]
[Deal "N:JT65.QJ763.AJ.Q7 9.KT52.KQT72.K84 K72.98.543.AT962 AQ843.A4.986.J53"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatal(err)
	}
	calls := NewEngine(d).Run()
	const want = "E:Passe S:Passe W:1P N:Passe E:2K S:Passe W:Passe N:Passe"
	if got := formatAuction(calls); got != want {
		t.Fatalf("enchères = %s, attendu %s", got, want)
	}
}

// TestPassedHandTwoOverOneNotForcing pins the response side of the same coin:
// a hand that has already passed cannot commit the side to game -- it is
// capped at 11 [E-1b] and the opening it faces may be light [O-7b]. Its
// change of suit at the two level names a playable contract and leaves the
// opening free to pass it there.
func TestPassedHandTwoOverOneNotForcing(t *testing.T) {
	// East deals and passes with 11 H and a five-card diamond suit, West
	// opens 1P light in third seat, East answers 2K.
	pbn := `[Dealer "E"]
[Vulnerable "EW"]
[Deal "N:JT65.QJ763.AJ.Q7 9.KT52.KQT72.K84 K72.98.543.AT962 AQ843.A4.986.J53"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatal(err)
	}
	calls := NewEngine(d).Run()
	got, comment := nthCallText(t, calls, 1, 1) // East's second call
	if got != "2K" {
		t.Fatalf("réponse d'Est = %s (%s), attendu 2K ; enchères : %s", got, comment, formatAuction(calls))
	}
	for _, sc := range calls {
		if sc.Seat == 1 && sc.Call == bid(2, SDiamonds) {
			if sc.M.forcing {
				t.Fatalf("le 2 sur 1 de la main passée est annoncé forcing (%s)", sc.M.fr)
			}
			if sc.M.maxPts != 11 {
				t.Fatalf("plafond annoncé %d, attendu 11 (%s)", sc.M.maxPts, sc.M.fr)
			}
		}
	}
}

// TestLightOpenerAnswersDrury pins the one question a passed hand still gets
// to ask [RM-2b], and the answer the light opening owes it: no, the points
// are not there. That answer is the whole reason the convention exists.
func TestLightOpenerAnswersDrury(t *testing.T) {
	const north, south = 0, 2
	// North passes, South opens 1P light in third seat with 11 H, North asks
	// with the Drury 2T on a three-card fit and 11 HLD.
	pbn := `[Dealer "N"]
[Deal "N:K72.A65.Q9843.76 T9.QJ32.A72.KJ54 AQJ63.K4.J65.982 854.T987.KT.AQT3"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatal(err)
	}
	calls := NewEngine(d).Run()
	if got, comment := nthCallText(t, calls, south, 0); got != "1P" || !strings.Contains(comment, "légère") {
		t.Fatalf("ouverture = %s (%s), attendue 1P légère ; enchères : %s", got, comment, formatAuction(calls))
	}
	if got, comment := nthCallText(t, calls, north, 1); got != "2T" || !strings.Contains(comment, "Drury") {
		t.Fatalf("réponse = %s (%s), attendu 2T Drury ; enchères : %s", got, comment, formatAuction(calls))
	}
	if got, comment := nthCallText(t, calls, south, 1); got != "2P" || !strings.Contains(comment, "légère") {
		t.Fatalf("réponse au Drury = %s (%s), attendu 2P légère ; enchères : %s", got, comment, formatAuction(calls))
	}
}
