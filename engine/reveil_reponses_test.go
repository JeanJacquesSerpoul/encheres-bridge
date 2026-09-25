package engine

import (
	"strings"
	"testing"
)

// Seats used by the balancing tests: South opens, West and North pass, East
// reopens and West answers.
const (
	rvNorth = 0
	rvEast  = 1
	rvSouth = 2
	rvWest  = 3
)

// reveilSeat plays "1x - Passe - Passe" and lets East, in the classic
// balancing seat, choose its call. South's opening is recorded rather than
// decided, so the hand under test is the only one that matters.
func reveilSeat(t *testing.T, opening Suit, east, west *Hand) (*Engine, Call, meaning) {
	t.Helper()
	// Only the two defenders ever decide anything here -- the opening and the
	// opponents' passes are recorded, not chosen -- so the table is seated
	// directly rather than dealt: the hands under test are then free to hold
	// whatever the case needs, without the cards having to add up to a deal.
	e := &Engine{dealer: rvSouth, opener: -1}
	for i := range 4 {
		e.ps[i] = &playerState{seat: i, hand: hand("", "", "", ""), shownMax: 40}
	}
	e.ps[rvEast].hand, e.ps[rvWest].hand = east, west
	n := 5
	if !opening.IsMajor() {
		n = 3
	}
	e.record(rvSouth, bidSuit(1, opening), m(12, 23, "ouverture", "opening").withLen(opening, n))
	e.record(rvWest, passCall, noInfo())
	e.record(rvNorth, passCall, noInfo())
	c, mn := e.decide(rvEast)
	e.record(rvEast, c, mn)
	return e, c, mn
}

// reveilAnswer passes for South and returns West's answer to East's balancing
// call.
func reveilAnswer(t *testing.T, e *Engine) (Call, meaning) {
	t.Helper()
	e.record(rvSouth, passCall, noInfo())
	c, mn := e.decide(rvWest)
	e.record(rvWest, c, mn)
	return c, mn
}

// TestReopenJumps checks the three jump reopenings of docs/regles_moteur.md
// §9.3 [V-5]. In a major the jump is a pure barrage — six cards to the two
// level (a good weak two), seven to the three level (a three-level opening) —
// while in a minor it is the opposite, a good six-card suit at the very limit
// of an opening, which is why it takes 11 HL rather than denying them.
func TestReopenJumps(t *testing.T) {
	cases := []struct {
		name    string
		opening Suit
		h       *Hand
		want    string
		hint    string
	}{
		{
			name:    "majeure sixième : barrage au palier de 2",
			opening: Diamonds,
			h:       hand("64", "KQJ982", "T3", "J75"),
			want:    "2C", hint: "6 cartes",
		},
		{
			name:    "majeure septième : barrage au palier de 3",
			opening: Hearts,
			h:       hand("KQJ9743", "5", "842", "96"),
			want:    "3P", hint: "7 cartes",
		},
		{
			name:    "mineure sixième : saut à la limite de l'ouverture",
			opening: Spades,
			h:       hand("83", "94", "AKJ973", "Q52"),
			want:    "3K", hint: "limite de l'ouverture",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := reopenEngine(tc.opening)
			p := &playerState{seat: 1, hand: tc.h}
			c, mn := e.overcall(p)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("réveil sur 1%s = %s (%s), want %s", suitNameFR[tc.opening], got, mn.fr, tc.want)
			}
			if !strings.Contains(mn.fr, tc.hint) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.hint)
			}
		})
	}
}

// TestReopenDoubleThenNotrump plays the reference sequence of the balanced
// 14-16 balancing hand: 1H - Passe - Passe - Contre / Passe - 1S - Passe -
// 1SA. The double alone would be read as the 8 H takeout shape, so the zone
// only reaches partner with the notrump bid that follows it, at the cheapest
// level over a minimum answer (docs/regles_moteur.md §9.3 [V-3]).
func TestReopenDoubleThenNotrump(t *testing.T) {
	east := hand("AQ3", "T74", "KJ85", "AQ4") // 16H régulier, trois petites à Coeur
	west := hand("J9642", "K83", "T96", "82") // 4H : réponse minimale
	e, c, mn := reveilSeat(t, Hearts, east, west)
	if c.Kind != KindDouble {
		t.Fatalf("East's reopening = %s (%s), want Contre", c.Format("fr"), mn.fr)
	}
	if !strings.Contains(mn.fr, "14-16H") {
		t.Fatalf("comment %q does not announce the balanced 14-16 zone", mn.fr)
	}
	c, mn = reveilAnswer(t, e)
	if got := c.Format("fr"); got != "1P" {
		t.Fatalf("West's answer = %s (%s), want 1P (4 cartes, 0-10H)", got, mn.fr)
	}
	e.record(rvNorth, passCall, noInfo())
	c, mn = e.decide(rvEast)
	if got := c.Format("fr"); got != "1SA" {
		t.Fatalf("East's rebid = %s (%s), want 1SA (14-16H)\nauction: %s", got, mn.fr, formatAuction(e.calls))
	}
	if mn.minPts != 14 || mn.maxPts != 16 {
		t.Fatalf("East's 1SA shows %d-%d, want 14-16", mn.minPts, mn.maxPts)
	}
}

// TestAnswerReopenDouble walks the reference ladder for the answers to a
// balancing double (docs/regles_moteur.md §9.5), on the document's own
// example auction 1T - Passe - Passe - Contre - Passe. One answers as though
// partner were weak — the double may hold 8 H — so the whole scale sits a full
// zone below the direct-seat answers of §9.2.
func TestAnswerReopenDouble(t *testing.T) {
	cases := []struct {
		name string
		h    *Hand
		want string
	}{
		{"4 cartes et 0 à 10H : palier de 1", hand("K843", "752", "J964", "83"), "1P"},
		{"4 cartes et 11-12H : saut au palier de 2", hand("A83", "KQ54", "QJ62", "74"), "2C"},
		{"5 cartes et 11-12H : double saut au palier de 3", hand("A83", "74", "QJ962", "KQ4"), "3K"},
		{"9-12H et l'arrêt : 1SA", hand("A83", "J74", "Q962", "K54"), "1SA"},
		{"13-14H et l'arrêt : 2SA", hand("AQ8", "J74", "KJ62", "K54"), "2SA"},
		{"l'ouverture sans enchère évidente : cue-bid", hand("AQ8", "AJ4", "KJ62", "954"), "2T"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// East holds the ideal three-suited shape short in clubs: the
			// 8 H reopening double.
			east := hand("KJ97", "Q863", "K852", "6")
			e, c, mn := reveilSeat(t, Clubs, east, tc.h)
			if c.Kind != KindDouble {
				t.Fatalf("East's reopening = %s (%s), want Contre", c.Format("fr"), mn.fr)
			}
			c, mn = reveilAnswer(t, e)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("réponse au contre de réveil = %s (%s), want %s\nauction: %s", got, mn.fr, tc.want, formatAuction(e.calls))
			}
		})
	}
}

// TestAnswerReopenSuit checks the answers to a suit réveil
// (docs/regles_moteur.md §9.5) on the auction 1K - Passe - Passe - 1P -
// Passe. The balancing bid is capped at 13 HL and has denied an opening, so
// the notrump answers carry higher zones than they would over a direct
// overcall, a new suit is a non-forcing misfit call, and the cue-bid of the
// opener's suit is a game try rather than a question about partner's strength.
func TestAnswerReopenSuit(t *testing.T) {
	cases := []struct {
		name string
		h    *Hand
		want string
		hint string
	}{
		{"9-12H avec l'arrêt : 1SA", hand("74", "KJ83", "Q962", "A75"), "1SA", "9-12H"},
		{"13-15H avec l'arrêt : 2SA", hand("74", "KJ83", "AQ92", "AJ5"), "2SA", "13-15H"},
		{"fit et l'ouverture : cue-bid espoir de manche", hand("KJ85", "A94", "73", "KQ86"), "2K", "espoir de manche"},
		{"fit et 7-10HLD : soutien simple", hand("KQ7", "J642", "A53", "T64"), "2P", "soutien simple"},
		{"fit et 11-12HLD : soutien à saut", hand("KQ7", "J642", "A53", "Q64"), "3P", "soutien à saut"},
		// Four trumps opposite the five the réveil promises: the ninth trump
		// is worth a point, and 10 H becomes the 11 HLD of the jump raise.
		{"fit de 9 : le neuvième atout compte", hand("KQ72", "J64", "A53", "T64"), "3P", "soutien à saut"},
		{"misfit avec sa propre couleur : non forcing", hand("6", "KQJ84", "95", "K7432"), "2C", "misfit"},
		{"rien à dire : passe", hand("83", "9642", "T74", "J952"), "Passe", ""},
	}
	// East: 8H and a good five-card spade suit -- the plain 1S reopening.
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			east := hand("AQ964", "T7", "J84", "983")
			e, c, mn := reveilSeat(t, Diamonds, east, tc.h)
			if got := c.Format("fr"); got != "1P" {
				t.Fatalf("East's reopening = %s (%s), want 1P", got, mn.fr)
			}
			c, mn = reveilAnswer(t, e)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("réponse au réveil = %s (%s), want %s\nauction: %s", got, mn.fr, tc.want, formatAuction(e.calls))
			}
			if tc.hint != "" && !strings.Contains(mn.fr, tc.hint) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.hint)
			}
		})
	}
}

// TestReopenCueBidAnswer checks the coded reply to the advancer's cue-bid over
// a suit réveil (docs/regles_moteur.md §9.5): the réveil spans 8-13 HL, and
// the reply splits it -- the minimum returns to its own suit, the maximum
// describes. It is the same machinery as the cue-bid over a direct overcall
// [A-6], one rung lower, since the réveil has already denied an opening.
func TestReopenCueBidAnswer(t *testing.T) {
	// West holds the fit and 13 HLD: it cue-bids 2D over East's 1S réveil.
	west := hand("KJ85", "A94", "73", "KQ86")

	t.Run("réveil minimum : retour à sa couleur", func(t *testing.T) {
		east := hand("AQ964", "T7", "J84", "983") // 8HL : le bas de la fourchette
		e, _, _ := reveilSeat(t, Diamonds, east, west)
		c, _ := reveilAnswer(t, e)
		if got := c.Format("fr"); got != "2K" {
			t.Fatalf("West's advance = %s, want the 2K cue-bid", got)
		}
		e.record(rvNorth, passCall, noInfo())
		c, mn := e.decide(rvEast)
		if got := c.Format("fr"); got != "2P" {
			t.Fatalf("East's answer = %s (%s), want 2P\nauction: %s", got, mn.fr, formatAuction(e.calls))
		}
		if !strings.Contains(mn.fr, "réveil minimum") {
			t.Fatalf("comment %q does not announce a minimum reopening", mn.fr)
		}
	})

	t.Run("réveil maximum : description", func(t *testing.T) {
		east := hand("AQJ64", "T7", "J84", "KJ3") // 13H : le haut de la fourchette
		e, _, _ := reveilSeat(t, Diamonds, east, west)
		c, _ := reveilAnswer(t, e)
		if got := c.Format("fr"); got != "2K" {
			t.Fatalf("West's advance = %s, want the 2K cue-bid", got)
		}
		e.record(rvNorth, passCall, noInfo())
		c, mn := e.decide(rvEast)
		if got := c.Format("fr"); got == "2P" {
			t.Fatalf("East's answer = 2P (%s): the plain return is reserved for the minimum\nauction: %s", mn.fr, formatAuction(e.calls))
		}
		if !strings.Contains(mn.fr, "maximum du réveil") {
			t.Fatalf("comment %q does not announce the top of the reopening range", mn.fr)
		}
	})
}
