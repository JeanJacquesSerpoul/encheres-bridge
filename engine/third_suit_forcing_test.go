package engine

import (
	"strings"
	"testing"
)

// TestThirdSuitForcingSupport runs the convention's first answer through a
// full deal: 1C - 1S - 2C caps opener and denies the four trumps he would have
// raised with, but says nothing about a third spade. Responder holds exactly
// five of them and the values for game, so 2D asks; opener's three spades come
// back as 3S and the 5-3 fit is played in game.
func TestThirdSuitForcingSupport(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			south: hand("K43", "83", "Q6", "AQJ932"), // 12H, six clubs, three spades
			west:  hand("82", "T9764", "AKJT", "54"), // 8H
			north: hand("AQJ76", "A52", "943", "T6"), // 11H, five spades, no diamond stopper
			east:  hand("T95", "KQJ", "8752", "K87"), // 9H
		},
	}
	calls := NewEngine(d).Run()
	checkFSCSeq(t, calls, []fscStep{
		{south, 0, "1T", ""},
		{north, 0, "1P", ""},
		{south, 1, "2T", "répétition"},
		{north, 1, "2K", "troisième couleur forcing"},
		{south, 2, "3P", "3 cartes dans votre majeure"},
	})
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P (the 5-3 spade fit)\nauction: %s", got, formatAuction(calls))
	}
}

// TestThirdSuitForcingStopper covers the convention's second question, the one
// the ask is really made for when no fit turns up: responder cannot guard the
// suit above opener's minor himself, and 3NT is unplayable unless opener can.
// Here he can, and the answer settles the contract.
func TestThirdSuitForcingStopper(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			south: hand("K4", "83", "KQ6", "AJ9432"), // 13H, six clubs, two spades, the diamond stopper
			west:  hand("32", "T976", "AJT2", "Q85"), // 7H
			north: hand("AQJ76", "A52", "943", "T6"), // 11H, five spades, no diamond stopper
			east:  hand("T985", "KQJ4", "875", "K7"), // 9H
		},
	}
	calls := NewEngine(d).Run()
	checkFSCSeq(t, calls, []fscStep{
		{south, 0, "1T", ""},
		{north, 0, "1P", ""},
		{south, 1, "2T", "répétition"},
		{north, 1, "2K", "troisième couleur forcing"},
		{south, 2, "3SA", "arrêt à Carreau"},
	})
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "3SA" {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", got, formatAuction(calls))
	}
}

// TestThirdSuitForcingDeniedPlaysTheMinor is the other side of that question.
// Opener has neither the third spade nor the diamond stopper and repeats his
// clubs; responder cannot guard diamonds either, so the notrump game the count
// would otherwise reach for is a fiction, and the side -- committed to game --
// plays the eleven-trick one in the fit it does hold.
func TestThirdSuitForcingDeniedPlaysTheMinor(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			south: hand("K4", "83", "Q6", "AQJ9432"), // 12H, seven clubs, no diamond stopper
			west:  hand("T98", "KQ74", "A85", "T76"), // 9H
			north: hand("AQJ76", "A52", "943", "K5"), // 14H, five spades, no diamond stopper
			east:  hand("532", "JT96", "KJT72", "8"), // 5H
		},
	}
	calls := NewEngine(d).Run()
	checkFSCSeq(t, calls, []fscStep{
		{north, 1, "2K", "troisième couleur forcing"},
		{south, 2, "3T", "ni soutien ni arrêt"},
		{north, 2, "5T", "manche dans le fit mineur"},
	})
}

// thirdSuitAuction rebuilds the engine state of 1m - 1M - 2m, the sequence the
// unit tests below examine one call at a time.
func thirdSuitAuction(responder, opener *Hand, minor, major Suit) *Engine {
	e := &Engine{ps: [4]*playerState{}, opener: 2, openCall: bidSuit(1, minor)}
	for i := range 4 {
		e.ps[i] = &playerState{seat: i, shownMax: 40}
	}
	e.ps[0].hand, e.ps[2].hand = responder, opener
	e.ps[0].bids, e.ps[0].responded, e.ps[0].shownMin = 1, true, 6
	e.ps[0].shownLens[major] = 4
	e.ps[2].bids, e.ps[2].shownMin, e.ps[2].shownMax = 2, 12, 16
	e.ps[2].shownLens[minor] = 6
	e.calls = []SeatCall{
		{Seat: 2, Call: bidSuit(1, minor), M: m(12, 23, "", "").withLen(minor, 3)},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bidSuit(1, major), M: m(6, 40, "", "").withLen(major, 4)},
		{Seat: 1, Call: passCall},
		{Seat: 2, Call: bidSuit(2, minor), M: m(12, 16, "", "").withLen(minor, 6)},
		{Seat: 3, Call: passCall},
	}
	return e
}

// TestThirdSuitForcingPerimeter checks when the ask is made and when it is
// not: it promises exactly five cards in the major and the values for game,
// and a hand that has already passed cannot promise the latter.
func TestThirdSuitForcingPerimeter(t *testing.T) {
	cases := []struct {
		name      string
		responder *Hand
		passed    bool
		want      bool
	}{
		{
			name:      "cinq piques et 11H : la question se pose",
			responder: hand("AQJ76", "A52", "943", "T6"), // 11H
			want:      true,
		},
		{
			name:      "majeure quatrième : l'enchère promettrait une cinquième carte",
			responder: hand("AQJ6", "A52", "9432", "T6"),
			want:      false,
		},
		{
			name:      "majeure sixième : elle se répète, elle ne demande rien",
			responder: hand("AQJ765", "A52", "943", "T"),
			want:      false,
		},
		{
			name:      "moins de 11H : rien à forcer",
			responder: hand("AQJ76", "852", "943", "T64"), // 7H
			want:      false,
		},
		{
			name:      "main déjà passée : elle ne peut pas engager la manche",
			responder: hand("AQJ76", "A52", "943", "T6"),
			passed:    true,
			want:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := thirdSuitAuction(tc.responder, hand("K43", "83", "Q6", "AQJ932"), Clubs, Spades)
			if tc.passed {
				e.calls = append([]SeatCall{{Seat: 0, Call: passCall}}, e.calls...)
			}
			c, _, ok := e.thirdSuitAsk(e.ps[0])
			if ok != tc.want {
				t.Fatalf("thirdSuitAsk = (%s, %v), want ok=%v", c.Format("fr"), ok, tc.want)
			}
			if ok && c.Format("fr") != "2K" {
				t.Fatalf("the third suit of 1T - 1P - 2T is Carreau: got %s", c.Format("fr"))
			}
		})
	}
}

// TestThirdSuitForcingIsAlwaysTheCollante pins the shape of the auction the
// convention lives in. The third suit is the one just above the opening minor;
// where that suit is the very one responder has already named, the side has no
// third suit left, since spades -- which a responder holding four would simply
// have bid at the one level -- are never the ask.
func TestThirdSuitForcingIsAlwaysTheCollante(t *testing.T) {
	responder := hand("A52", "AQJ76", "943", "T6") // five hearts, 11H
	e := thirdSuitAuction(responder, hand("83", "K43", "AQJ932", "Q6"), Diamonds, Hearts)
	if c, _, ok := e.thirdSuitAsk(e.ps[0]); ok {
		t.Fatalf("1K - 1C - 2K has no third suit below Pique, got %s", c.Format("fr"))
	}
	// The same hand opposite 1C: there the collante is Carreau, and free.
	e = thirdSuitAuction(responder, hand("K43", "83", "Q6", "AQJ932"), Clubs, Hearts)
	c, _, ok := e.thirdSuitAsk(e.ps[0])
	if !ok || c.Format("fr") != "2K" {
		t.Fatalf("thirdSuitAsk over 1T - 1C - 2T = (%s, %v), want 2K", c.Format("fr"), ok)
	}
}

// TestThirdSuitForcingNotOverArtificialAuctions guards the deduction the
// convention rests on: it reads a natural minor opening, repeated at the two
// level for its own length. A call that promises no length in the suit it
// names leaves nothing to deduce, and the ask must not fire.
func TestThirdSuitForcingNotOverArtificialAuctions(t *testing.T) {
	e := thirdSuitAuction(hand("AQJ76", "A52", "943", "T6"), hand("K43", "83", "Q6", "AQJ932"), Clubs, Spades)
	e.calls[4].M = m(12, 16, "", "") // the 2C rebid, stripped of its promised length
	if c, _, ok := e.thirdSuitAsk(e.ps[0]); ok {
		t.Fatalf("the third suit forcing fired (%s) over a rebid promising no length", c.Format("fr"))
	}
}

// TestThirdSuitAnswerShapes drives opener's answers, in the convention's own
// order, over 1C - 1S - 2C - 2D and over the 1D - 1S - 2D - 2H auction where
// the third suit is a major and a 4-4 fit is still to be found.
func TestThirdSuitAnswerShapes(t *testing.T) {
	cases := []struct {
		name         string
		opener       *Hand
		minor, third Suit
		call         string
		hint         string
	}{
		{
			name:   "3 cartes dans sa majeure : le fit 5-3 cherché",
			opener: hand("K43", "83", "Q6", "AQJ932"),
			minor:  Clubs,
			third:  Diamonds,
			call:   "3P",
			hint:   "3 cartes dans votre majeure",
		},
		{
			name:   "l'arrêt dans la troisième couleur",
			opener: hand("K4", "83", "KQ6", "AJ9432"),
			minor:  Clubs,
			third:  Diamonds,
			call:   "3SA",
			hint:   "arrêt à Carreau",
		},
		{
			name:   "4 cartes dans la troisième couleur, sans l'arrêt",
			opener: hand("KQ", "83", "9432", "AQJ92"),
			minor:  Clubs,
			third:  Diamonds,
			call:   "3K",
			hint:   "sans l'arrêt",
		},
		{
			name:   "ni soutien ni arrêt : la couleur d'ouverture",
			opener: hand("KQ", "83", "92", "AQJ9432"),
			minor:  Clubs,
			third:  Diamonds,
			call:   "3T",
			hint:   "répétition de la couleur d'ouverture",
		},
		{
			name:   "4 cartes dans la troisième couleur majeure : le fit 4-4 possible",
			opener: hand("K4", "Q432", "AQJ932", "8"),
			minor:  Diamonds,
			third:  Hearts,
			call:   "3C",
			hint:   "4 cartes à Cœur",
		},
		{
			name:   "le soutien de la majeure passe avant ce fit-là",
			opener: hand("K43", "Q432", "AQJ932", ""),
			minor:  Diamonds,
			third:  Hearts,
			call:   "3P",
			hint:   "3 cartes dans votre majeure",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := thirdSuitAuction(hand("AQJ76", "A52", "943", "T6"), tc.opener, tc.minor, Spades)
			ask := m(11, 40, "", "").asForcing()
			ask.thirdSuit, ask.thirdSuitSuit = true, tc.third
			e.calls = append(e.calls,
				SeatCall{Seat: 0, Call: bidSuit(2, tc.third), M: ask},
				SeatCall{Seat: 1, Call: passCall})
			c, mn := e.thirdSuitAnswer(e.ps[2], tc.third)
			if got := c.Format("fr"); got != tc.call {
				t.Fatalf("answer = %s (%s), want %s", got, mn.fr, tc.call)
			}
			if !strings.Contains(mn.fr, tc.hint) {
				t.Fatalf("comment %q does not mention %q", mn.fr, tc.hint)
			}
		})
	}
}
