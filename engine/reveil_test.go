package engine

import (
	"strings"
	"testing"
)

// reopenEngine builds the auction 1x - Passe - Passe with South (seat 2) as
// dealer and opener, leaving East (seat 1) in the classic balancing seat
// (docs/regles_moteur.md §9.3).
func reopenEngine(opening Suit) *Engine {
	e := &Engine{}
	e.opener = 2
	e.openCall = bidSuit(1, opening)
	e.calls = []SeatCall{
		{Seat: 2, Call: bidSuit(1, opening)},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: passCall},
	}
	return e
}

// TestReopenZones checks the balancing table of docs/regles_moteur.md §9.3,
// well below the direct-seat ranges: a suit at 8-13 HL denying an opening,
// 1SA at 9-13 HL, 2SA natural at 17-19 HL, the double from 8 H with the ideal
// three-suited shape (compulsory from 14 HL, and followed by a notrump bid on
// a balanced 14-16), the major barrage jumps, and the shifted two-suiter
// scheme (2SA being natural).
func TestReopenZones(t *testing.T) {
	cases := []struct {
		name    string
		opening Suit
		h       *Hand
		want    string
		hint    string
	}{
		{
			// Exact hand from the reference document (section 2).
			name:    "couleur simple : 8H et 5 belles cartes",
			opening: Diamonds,
			h:       hand("KQT64", "J9", "Q53", "T32"),
			want:    "1P", hint: "dénie l'ouverture",
		},
		{
			// Exact hand from the reference document (section 3).
			name:    "1SA : 9-13HL régulier avec arrêt",
			opening: Hearts,
			h:       hand("Q2", "KT5", "AT743", "J95"),
			want:    "1SA", hint: "9-13HL",
		},
		{
			// Exact hand from the reference document (section 4).
			name:    "contre dès 8H : tricolore court dans l'ouverture",
			opening: Hearts,
			h:       hand("AT83", "2", "KJ72", "J973"),
			want:    "Contre", hint: "dès 8H",
		},
		{
			// Balanced 14-16 with the opened suit held: too strong for the
			// limited 1SA (9-13 HL) and too weak for 2SA, so it doubles and
			// names notrump afterwards -- the only way to show the zone.
			name:    "contre 14-16H régulier : le Sans-Atout suivra",
			opening: Diamonds,
			h:       hand("AQ3", "KJ6", "AT75", "QT2"),
			want:    "Contre", hint: "14-16H",
		},
		{
			name:    "2SA naturel : 17-19HL régulier",
			opening: Diamonds,
			h:       hand("AQ3", "KJ6", "AT75", "KT2"),
			want:    "2SA", hint: "17-19",
		},
		{
			// Exact hand from the reference document (section 5b).
			name:    "saut constructif : une dizaine de points, 6 belles cartes",
			opening: Diamonds,
			h:       hand("KQT972", "2", "AJ4", "853"),
			want:    "2P", hint: "saut",
		},
		{
			name:    "bicolore sur 1T : 2K montre les majeures",
			opening: Clubs,
			h:       hand("AQJ85", "KQT85", "43", "7"),
			want:    "2K", hint: "majeures",
		},
		{
			name:    "bicolore sur 1C : cue-bid, autre majeure et Trèfle",
			opening: Hearts,
			h:       hand("AKJ72", "3", "63", "AQJ85"),
			want:    "2C", hint: "bicolore",
		},
		{
			name:    "bicolore sur 1T : cue-bid, les deux moins chères",
			opening: Clubs,
			h:       hand("7", "AKQJ9", "QJT84", "96"),
			want:    "2T", hint: "bicolore",
		},
		{
			// The 2D jump over 1C shows the majors, so the six-card diamond
			// hand reopens with a simple 1D.
			name:    "unicolore Carreau sur 1T : pas de saut, 2K = majeures",
			opening: Clubs,
			h:       hand("85", "93", "KQJT84", "A72"),
			want:    "1K", hint: "couleur",
		},
		{
			// Short in the opened suit: 8 H is enough to reopen.
			name:    "contre à 8H : singleton dans l'ouverture",
			opening: Diamonds,
			h:       hand("9832", "K872", "3", "KQ73"),
			want:    "Contre", hint: "réveil",
		},
		{
			// One point short of the 8 H floor, and four cards in the opened
			// suit besides: partner could not act despite his shortness
			// there, so prudence commands a pass.
			name:    "passe : 7H et 4 cartes dans l'ouverture",
			opening: Diamonds,
			h:       hand("983", "K72", "QJ53", "J73"),
			want:    "Passe", hint: "",
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
			if tc.hint != "" && !strings.Contains(mn.fr, tc.hint) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.hint)
			}
		})
	}
}

// TestReopenNotInSandwich checks that the balancing ranges never leak into
// the sandwich position (docs/regles_moteur.md §9.3): after 1H - Passe -
// 1S, at least one opponent is unlimited, so the ideal-shape hand that would
// double for takeout in the balancing seat (9H, where 8 is enough) is far too
// weak for the direct-seat ranges and must pass.
func TestReopenNotInSandwich(t *testing.T) {
	e := &Engine{}
	e.opener = 2
	e.openCall = bidSuit(1, Hearts)
	e.calls = []SeatCall{
		{Seat: 2, Call: bidSuit(1, Hearts)},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bidSuit(1, Spades)},
	}
	p := &playerState{seat: 1, hand: hand("AT83", "2", "KJ72", "J973")}
	c, mn := e.overcall(p)
	if c.Kind != KindPass {
		t.Fatalf("in the sandwich position the 9H hand called %s (%s), want Passe", c.Format("fr"), mn.fr)
	}
}

// TestReopenFullAuction plays a whole deal: South's 1D opening comes back to
// East after two passes. East (8H, a good five-card spade suit) is too weak
// for anything in the direct seat but must reopen with 1S in the balancing
// seat -- West, long in diamonds behind the opener, holds the missing
// strength and had no direct call available (docs/regles_moteur.md §9.3).
func TestReopenFullAuction(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			north: hand("9853", "QT542", "6", "J95"), // 3 HL: passes
			east:  hand("KQT64", "J9", "Q53", "T83"), // 8H: reopens 1S
			south: hand("A72", "A63", "K872", "Q42"), // 13H: opens 1D
			west:  hand("J", "K87", "AJT94", "AK76"), // 16H, long in diamonds: no direct call
		},
	}
	calls := NewEngine(d).Run()
	if len(calls) < 4 {
		t.Fatalf("auction too short: %s", formatAuction(calls))
	}
	if calls[0].Seat != south || calls[0].Call.Format("fr") != "1K" {
		t.Fatalf("South should open 1K, got %s\nauction: %s", calls[0].Call.Format("fr"), formatAuction(calls))
	}
	if calls[1].Call.Kind != KindPass || calls[2].Call.Kind != KindPass {
		t.Fatalf("West and North should both pass\nauction: %s", formatAuction(calls))
	}
	if got := calls[3].Call.Format("fr"); calls[3].Seat != east || got != "1P" {
		t.Fatalf("East's reopening = %s (%s), want 1P\nauction: %s", got, calls[3].M.fr, formatAuction(calls))
	}
	if !strings.Contains(calls[3].M.fr, "réveil") {
		t.Fatalf("comment %q does not mention the balancing seat", calls[3].M.fr)
	}
}

// TestReopenNeedsAnOpponentBid guards the entry condition of the balancing
// seat (docs/regles_moteur.md §9.3): the seat exists because an opposing
// opening would otherwise buy the contract cheaply. Three passes buy nothing,
// so the fourth player is not reopening anything -- he is opening, under the
// opening ranges, and the 8-13 HL balancing zones must never reach him.
func TestReopenNeedsAnOpponentBid(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	// West holds 8 H, a five-card diamond suit and the ideal takeout shape
	// against clubs: everything the balancing seat asks for, and none of it
	// worth a call when nobody has opened.
	w := hand("AQ3", "984", "Q9874", "86")
	d := &Deal{
		Dealer: north,
		Hands: [4]*Hand{
			north: hand("9852", "T3", "AK", "A9752"),
			east:  hand("JT7", "KQJ62", "T3", "KJ4"),
			south: hand("K64", "A75", "J652", "QT3"),
			west:  w,
		},
	}
	calls := NewEngine(d).Run()
	if len(calls) != 4 {
		t.Fatalf("auction should be passed out in four calls: %s", formatAuction(calls))
	}
	for _, sc := range calls {
		if sc.Call.Kind != KindPass {
			t.Fatalf("expected four passes, got %s\nauction: %s", sc.Call.Format("fr"), formatAuction(calls))
		}
		if sc.M.reopen {
			t.Fatalf("call by seat %d marked as a reopening with no opponent bid: %s", sc.Seat, sc.M.fr)
		}
	}
	// The predicate itself: three passes are not a balancing position.
	e := &Engine{opener: -1}
	e.calls = []SeatCall{
		{Seat: north, Call: passCall},
		{Seat: east, Call: passCall},
		{Seat: south, Call: passCall},
	}
	if e.isClassicReopen() {
		t.Fatal("isClassicReopen accepted an auction with no bid in it")
	}
	// The very same hand does reopen once the opponents have opened.
	re := reopenEngine(Clubs)
	c, mn := re.overcall(&playerState{seat: 1, hand: w})
	if c.Kind == KindPass {
		t.Fatalf("the same 8 H hand passed over 1T - Passe - Passe (%s)", mn.fr)
	}
}
