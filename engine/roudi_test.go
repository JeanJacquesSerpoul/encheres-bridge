package main

import (
	"strings"
	"testing"
)

// TestRoudiThreeSteps drives the three-step Roudi (docs/addon_9.md) through
// full deals: after 1m - 1M - 1SA (opener 12-14 balanced), responder's 2C
// asks with 11+H and exactly five cards in his major. Opener answers 2D with
// two cards in the major, minimum or maximum alike; 2H with three cards and a
// minimum; 2S with three cards and a maximum. Responder then places the
// contract -- a stop at the two level when both hands are minimum, game in
// the 5-3 fit or 3NT otherwise, and an invitation at 2NT when the 2D answer
// left the opener's zone open.
func TestRoudiThreeSteps(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	type step struct {
		seat int
		nth  int
		call string
		hint string
	}
	cases := []struct {
		name     string
		hands    [4]*Hand
		seq      []step
		contract string
	}{
		{
			// 2H answer (three hearts, minimum) facing a minimum responder
			// (11H): the auction stops right there, in the 5-3 fit.
			name: "2C : 3 cartes et jeu faible, le répondant minimum passe",
			hands: [4]*Hand{
				north: hand("A52", "KJT63", "72", "K94"), // 11H, 5 hearts
				east:  hand("T73", "54", "KQ63", "AT65"), // 9H
				south: hand("K84", "Q92", "AJ85", "Q73"), // 12H, 3 hearts
				west:  hand("QJ96", "A87", "T94", "J82"), // 8H
			},
			seq: []step{
				{south, 0, "1K", ""},
				{north, 0, "1C", ""},
				{south, 1, "1SA", "Roudi disponible"},
				{north, 1, "2T", "Roudi"},
				{south, 2, "2C", "3 cartes"},
				{north, 2, "Passe", "arrêt"},
			},
			contract: "2C",
		},
		{
			// 2D answer (two hearts, minimum) facing 13H: game anyway, and
			// without the 5-3 fit the contract is 3NT.
			name: "2K : 2 cartes et jeu faible, le répondant fort conclut 3SA",
			hands: [4]*Hand{
				north: hand("A52", "KJT63", "Q2", "K94"), // 13H, 5 hearts
				east:  hand("T73", "A542", "AK6", "652"), // 11H
				south: hand("K84", "Q9", "J853", "AQ73"), // 12H, 2 hearts
				west:  hand("QJ96", "87", "T94", "JT86"), // 4H
			},
			seq: []step{
				{south, 1, "1SA", "Roudi disponible"},
				{north, 1, "2T", "Roudi"},
				{south, 2, "2K", "2 cartes"},
				{north, 2, "3SA", "pas de fit"},
			},
			contract: "3SA",
		},
		{
			// 2S answer (three hearts, maximum 13-14): even the minimum
			// responder (11H) bids the game in the 5-3 fit.
			name: "2P : 3 cartes et jeu fort, manche dans le fit 5-3",
			hands: [4]*Hand{
				north: hand("A52", "KQT63", "72", "Q94"), // 11H, 5 hearts
				east:  hand("763", "54", "AKQ43", "T52"), // 9H
				south: hand("KQ4", "A92", "J85", "KJ73"), // 14H, 3 hearts
				west:  hand("JT98", "J87", "T96", "A86"), // 6H
			},
			seq: []step{
				{south, 1, "1SA", "Roudi disponible"},
				{north, 1, "2T", "Roudi"},
				{south, 2, "2P", "jeu fort"},
				{north, 2, "4C", "fit majeur 5-3"},
			},
			contract: "4C",
		},
		{
			// Two hearts and a maximum take the same 2D step as the minimum,
			// so the zone is still open: the minimum responder (11H) invites
			// at 2NT and opener, maximum, bids the game.
			name: "2K sur un maximum : le répondant propose, l'ouvreur conclut",
			hands: [4]*Hand{
				north: hand("A52", "KQT63", "72", "Q94"), // 11H, 5 hearts
				east:  hand("763", "542", "AKQ4", "T52"), // 9H
				south: hand("KQ4", "A9", "J853", "KJ73"), // 14H, 2 hearts
				west:  hand("JT98", "J87", "T96", "A86"), // 6H
			},
			seq: []step{
				{south, 1, "1SA", "Roudi disponible"},
				{north, 1, "2T", "Roudi"},
				{south, 2, "2K", "minimum ou maximum"},
				{north, 2, "2SA", "proposition"},
				{south, 3, "3SA", ""},
			},
			contract: "3SA",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Deal{Dealer: south, Hands: tc.hands}
			calls := NewEngine(d).Run()
			for _, w := range tc.seq {
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
							t.Fatalf("comment %q does not mention %q", sc.M.fr, w.hint)
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
			contract, _, _ := finalContract(calls)
			if got := contract.Format("fr"); got != tc.contract {
				t.Fatalf("final contract = %s, want %s\nauction: %s", got, tc.contract, formatAuction(calls))
			}
		})
	}
}

// TestRoudiAfterMajorOpening reproduces the reported Bridge Teacher deal:
// North opens 1H, South answers 1S (five spades, 13H) and North rebids 1SA
// (12-14). The engine used to conclude a flat 3SA (nine tricks); the Roudi
// also applies after 1H - 1S - 1SA, finds North's three-card spade support
// (2H answer: three cards, minimum) and lands the par contract 4S.
func TestRoudiAfterMajorOpening(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: north,
		Hands: [4]*Hand{
			north: hand("AJ3", "AJ852", "QT6", "T6"), // 12H, 5 hearts, 3 spades
			east:  hand("Q", "KT93", "95432", "K92"), // 8H
			south: hand("KT742", "76", "AK8", "QJ7"), // 13H, 5 spades
			west:  hand("9865", "Q4", "J7", "A8543"), // 7H
		},
	}
	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "1C", ""},
		{south, 0, "1P", ""},
		{north, 1, "1SA", "Roudi disponible"},
		{south, 1, "2T", "Roudi"},
		{north, 2, "2C", "3 cartes"},
		{south, 2, "4P", "fit majeur 5-3"},
	}
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
					t.Fatalf("comment %q does not mention %q", sc.M.fr, w.hint)
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
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P (the deal's par)\nauction: %s", got, formatAuction(calls))
	}
}

// TestRoudiNotWithSixCards checks the convention's perimeter: with a six-card
// major (or in the slam zone) responder keeps the natural developments, so
// 2C over the 1NT rebid stays unavailable and the existing six-card logic
// (direct game in the major) applies -- roudiAsk demands exactly five cards.
func TestRoudiNotWithSixCards(t *testing.T) {
	e := &Engine{ps: [4]*playerState{}}
	e.ps[2] = &playerState{seat: 2}
	p := &playerState{seat: 0, hand: hand("A5", "KJT632", "72", "K94")} // six hearts
	e.ps[0] = p
	e.calls = []SeatCall{
		{Seat: 2, Call: bidSuit(1, Diamonds)},
		{Seat: 3, Call: passCall},
		{Seat: 0, Call: bidSuit(1, Hearts)},
		{Seat: 1, Call: passCall},
		{Seat: 2, Call: bid(1, SNoTrump)},
		{Seat: 3, Call: passCall},
	}
	if _, _, ok := e.roudiAsk(p); ok {
		t.Fatalf("roudiAsk accepted a six-card major; the convention requires exactly five")
	}
}
