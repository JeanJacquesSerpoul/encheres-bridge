package engine

import (
	"strings"
	"testing"
)

// fscStep is one expected call of a full-deal auction: the nth call of a
// seat, and a fragment its comment must carry.
type fscStep struct {
	seat int
	nth  int
	call string
	hint string
}

func checkFSCSeq(t *testing.T, calls []SeatCall, seq []fscStep) {
	t.Helper()
	for _, w := range seq {
		n := 0
		found := false
		for _, sc := range calls {
			if sc.Seat != w.seat {
				continue
			}
			if n == w.nth {
				if got := sc.Call.Format("fr"); got != w.call {
					t.Fatalf("seat %s call #%d = %s (%s), want %s\nauction: %s",
						seatNames[w.seat], w.nth, got, sc.M.fr, w.call, formatAuction(calls))
				}
				if w.hint != "" && !strings.Contains(sc.M.fr, w.hint) {
					t.Fatalf("comment %q does not mention %q\nauction: %s", sc.M.fr, w.hint, formatAuction(calls))
				}
				found = true
				break
			}
			n++
		}
		if !found {
			t.Fatalf("seat %s never made call #%d\nauction: %s", seatNames[w.seat], w.nth, formatAuction(calls))
		}
	}
}

// TestFourthSuitForcingSupport runs the convention's first answer through a
// full deal: 1H - 1S - 2C leaves diamonds unbid, responder holds exactly five
// spades and no diamond stopper, so 2D asks. Opener's three spades come back
// as 2S and the 5-3 fit is played in game.
func TestFourthSuitForcingSupport(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			south: hand("Q83", "AQJ64", "4", "KT52"), // 12H, 5-4 hearts and clubs, three spades
			west:  hand("T94", "K85", "AJ76", "983"), // 8H
			north: hand("AKJ62", "72", "983", "Q64"), // 10H, five spades, no diamond stopper
			east:  hand("75", "T93", "KQT52", "AJ7"), // 10H
		},
	}
	calls := NewEngine(d).Run()
	checkFSCSeq(t, calls, []fscStep{
		{south, 0, "1C", ""},
		{north, 0, "1P", ""},
		{south, 1, "2T", "bicolore"},
		{north, 1, "2K", "quatrième couleur forcing"},
		{south, 2, "2P", "3 cartes dans votre majeure"},
	})
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P (the 5-3 spade fit)\nauction: %s", got, formatAuction(calls))
	}
}

// TestFourthSuitForcingImpossibleSpade covers the answer the ask is really
// made for when the asker's own suit is a minor: 1C - 1D - 1H leaves spades
// unbid, responder cannot guard them himself, and the "impossible" 2S asks
// opener for the stopper that makes notrump playable. A natural spade suit
// would have been shown at the one level, which is why the convention has to
// climb a level here and 1S stays natural.
func TestFourthSuitForcingImpossibleSpade(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			south: hand("Q83", "AQJ4", "4", "KT952"), // 12H, five clubs and four hearts, spade stopper
			west:  hand("KT95", "K65", "J76", "873"), // 7H
			north: hand("762", "72", "AKQ983", "Q4"), // 11H, six diamonds, no spade stopper
			east:  hand("AJ4", "T983", "T52", "AJ6"), // 10H
		},
	}
	calls := NewEngine(d).Run()
	checkFSCSeq(t, calls, []fscStep{
		{south, 0, "1T", ""},
		{north, 0, "1K", ""},
		{south, 1, "1C", "changement de couleur au palier de 1"},
		{north, 1, "2P", "quatrième couleur forcing"},
		{south, 2, "2SA", "arrêt à Pique"},
	})
}

// fourthSuitAuction rebuilds the engine state of 1H - 1S - 2C, the sequence
// the unit tests below examine one call at a time.
func fourthSuitAuction(responder, opener *Hand) *Engine {
	e := &Engine{ps: [4]*playerState{}, opener: 2, openCall: bidSuit(1, Hearts)}
	for i := range 4 {
		e.ps[i] = &playerState{seat: i, shownMax: 40}
	}
	e.ps[0].hand, e.ps[2].hand = responder, opener
	e.ps[0].bids, e.ps[0].responded, e.ps[0].shownMin = 1, true, 6
	e.ps[0].shownLens[Spades] = 4
	e.ps[2].bids, e.ps[2].shownMin, e.ps[2].shownMax = 2, 12, 19
	e.ps[2].shownLens[Hearts], e.ps[2].shownLens[Clubs] = 5, 4
	e.calls = []SeatCall{
		{Seat: 2, Call: bidSuit(1, Hearts), M: m(12, 23, "", "").withLen(Hearts, 5)},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bidSuit(1, Spades), M: m(6, 40, "", "").withLen(Spades, 4)},
		{Seat: 1, Call: passCall},
		{Seat: 2, Call: bidSuit(2, Clubs), M: m(12, 19, "", "").withLen(Clubs, 4)},
		{Seat: 3, Call: passCall},
	}
	return e
}

// TestFourthSuitForcingPerimeter checks when the ask is *not* made: with a
// known fit there is nothing to look for, a stopper of our own answers the
// only other question, and ten points are the floor.
func TestFourthSuitForcingPerimeter(t *testing.T) {
	cases := []struct {
		name      string
		responder *Hand
		want      bool
	}{
		{
			name:      "cinq piques, sans arrêt à Carreau : la question se pose",
			responder: hand("AKJ62", "72", "983", "Q64"), // 10H
			want:      true,
		},
		{
			name:      "arrêt à Carreau : la main se décrit seule",
			responder: hand("AKJ6", "72", "KJ83", "Q64"),
			want:      false,
		},
		{
			name:      "moins de 10H : rien à demander",
			responder: hand("AJ862", "72", "9843", "64"), // 5H
			want:      false,
		},
		{
			name:      "fit Trèfle connu : le soutien passe avant la demande",
			responder: hand("AKJ62", "7", "83", "Q6432"),
			want:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := fourthSuitAuction(tc.responder, hand("Q83", "AQJ64", "4", "KT52"))
			c, _, ok := e.fourthSuitAsk(e.ps[0])
			if ok != tc.want {
				t.Fatalf("fourthSuitAsk = (%s, %v), want ok=%v", c.Format("fr"), ok, tc.want)
			}
			if ok && c.Format("fr") != "2K" {
				t.Fatalf("the fourth suit of 1C - 1P - 2T is Carreau: got %s", c.Format("fr"))
			}
		})
	}
}

// TestFourthSuitForcingNotOverArtificialAuctions guards the deduction itself:
// the convention reads three *natural* suits and concludes that nobody holds
// the fourth. After a strong 2C opening and its 2D relay, neither call
// promises the suit it names, so there is no fourth suit to deduce.
func TestFourthSuitForcingNotOverArtificialAuctions(t *testing.T) {
	e := &Engine{ps: [4]*playerState{}, opener: 2, openCall: bid(2, SClubs)}
	for i := range 4 {
		e.ps[i] = &playerState{seat: i, shownMax: 40}
	}
	e.ps[0].hand, e.ps[0].bids, e.ps[0].responded = hand("AKJ62", "72", "983", "Q64"), 1, true
	e.ps[2].hand, e.ps[2].bids = hand("Q83", "AQJ64", "A4", "AK5"), 2
	e.calls = []SeatCall{
		{Seat: 2, Call: bid(2, SClubs), M: m(18, 23, "", "").asForcing()}, // artificial: no club length
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bid(2, SDiamonds), M: m(-1, -1, "", "")}, // artificial relay
		{Seat: 1, Call: passCall},
		{Seat: 2, Call: bidSuit(2, Hearts), M: m(-1, -1, "", "").withLen(Hearts, 5)},
		{Seat: 3, Call: passCall},
	}
	if c, _, ok := e.fourthSuitAsk(e.ps[0]); ok {
		t.Fatalf("the fourth suit forcing fired (%s) over an artificial 2T - 2K auction", c.Format("fr"))
	}
}

// TestFourthSuitAnswerShapes drives opener's answers, in the convention's own
// order, over the same 1H - 1S - 2C - 2D ask.
func TestFourthSuitAnswerShapes(t *testing.T) {
	cases := []struct {
		name   string
		opener *Hand
		call   string
		hint   string
	}{
		{
			name:   "trois cartes dans sa majeure, minimum",
			opener: hand("Q83", "AQJ64", "4", "KT52"),
			call:   "2P",
			hint:   "3 cartes dans votre majeure",
		},
		{
			name:   "trois cartes et 17HLD et plus : le saut",
			opener: hand("KQ8", "AQJ64", "4", "AK52"),
			call:   "3P",
			hint:   "17-19HLD",
		},
		{
			name:   "l'arrêt dans la quatrième couleur, minimum",
			opener: hand("Q8", "AQJ64", "K42", "T952"),
			call:   "2SA",
			hint:   "arrêt à Carreau",
		},
		{
			name:   "l'arrêt, 15H et plus",
			opener: hand("Q8", "AQJ64", "KQ2", "AT5"),
			call:   "3SA",
			hint:   "15H et plus",
		},
		{
			name:   "5-4-3-1 avec l'As troisième : le soutien conventionnel",
			opener: hand("8", "AQJ64", "A42", "KT52"),
			call:   "3K",
			hint:   "soutien conventionnel",
		},
		{
			name:   "bicolore 5-5 sans arrêt",
			opener: hand("84", "AQJ64", "2", "KT952"),
			call:   "3T",
			hint:   "5-5",
		},
		{
			name:   "couleur sixième sans arrêt",
			opener: hand("8", "AQJ642", "72", "KT52"),
			call:   "2C",
			hint:   "sixième",
		},
		{
			// 6-5 is both shapes at once, and the branch that fires must be
			// the long suit's: announcing 5-5 buries the sixth card of the
			// opening suit, which is what turns a possible 6-2 into a real
			// trump suit (audit du par, donne 404 -- the 5-5 answer left a
			// nine-card diamond fit unfound and the side in a six-card one).
			name:   "bicolore 6-5 : la sixième prime sur le 5-5",
			opener: hand("8", "AQJ642", "7", "KT952"),
			call:   "2C",
			hint:   "sixième",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := fourthSuitAuction(hand("AKJ62", "72", "983", "Q64"), tc.opener)
			ask := m(10, 40, "", "").asForcing()
			ask.fourthSuit, ask.fourthSuitSuit = true, Diamonds
			e.calls = append(e.calls,
				SeatCall{Seat: 0, Call: bidSuit(2, Diamonds), M: ask},
				SeatCall{Seat: 1, Call: passCall})
			c, mn := e.fourthSuitAnswer(e.ps[2], Diamonds)
			if got := c.Format("fr"); got != tc.call {
				t.Fatalf("answer = %s (%s), want %s", got, mn.fr, tc.call)
			}
			if !strings.Contains(mn.fr, tc.hint) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.hint)
			}
		})
	}
}
