package main

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func hand(spades, hearts, diamonds, clubs string) *Hand {
	h := &Hand{}
	h.Suits[Spades] = sortRanks(strings.ToUpper(spades))
	h.Suits[Hearts] = sortRanks(strings.ToUpper(hearts))
	h.Suits[Diamonds] = sortRanks(strings.ToUpper(diamonds))
	h.Suits[Clubs] = sortRanks(strings.ToUpper(clubs))
	return h
}

// dealWith builds a full deal where the given seats hold exactly the
// supplied hands; the remaining cards are dealt deterministically (but
// arbitrarily, they never influence the calls under test) to the other seats.
func dealWith(dealer int, given map[int]*Hand) *Deal {
	var used [4]map[byte]bool
	for s := Clubs; s <= Spades; s++ {
		used[s] = map[byte]bool{}
	}
	for seat, h := range given {
		for s := Clubs; s <= Spades; s++ {
			for i := 0; i < len(h.Suits[s]); i++ {
				if used[s][h.Suits[s][i]] {
					panic(fmt.Sprintf("dealWith: duplicate card %c in suit %d (seat %d)", h.Suits[s][i], s, seat))
				}
				used[s][h.Suits[s][i]] = true
			}
		}
	}
	type card struct {
		s Suit
		r byte
	}
	var deck []card
	for s := Clubs; s <= Spades; s++ {
		for i := 0; i < len(rankOrder); i++ {
			r := rankOrder[i]
			if !used[s][r] {
				deck = append(deck, card{s, r})
			}
		}
	}
	rng := rand.New(rand.NewSource(99))
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })

	var hands [4]*Hand
	idx := 0
	for seat := 0; seat < 4; seat++ {
		if h, ok := given[seat]; ok {
			hands[seat] = h
			continue
		}
		h := &Hand{}
		var suits [4][]byte
		for n := 0; n < 13; n++ {
			c := deck[idx]
			idx++
			suits[c.s] = append(suits[c.s], c.r)
		}
		for s := Clubs; s <= Spades; s++ {
			h.Suits[s] = sortRanks(string(suits[s]))
		}
		hands[seat] = h
	}
	return &Deal{Dealer: dealer, Hands: hands}
}

// TestMichaelsShape checks the specified Michaels two-suiter (docs/addon_3.md):
// a 5-5+ two-suiter over a one-level opening, where the pair of suits shown
// depends on the opened suit.
func TestMichaelsShape(t *testing.T) {
	const south, west = 2, 3
	cases := []struct {
		name      string
		southHand *Hand
		westHand  *Hand
		want      string
		hint      string
	}{
		{
			name:      "cue-bid sur 1C : autre majeure et Trefle",
			southHand: hand("64", "AKQ85", "K92", "742"),
			westHand:  hand("AKJ72", "3", "63", "AQJ85"),
			want:      "2C", hint: "cue-bid Michaël",
		},
		{
			// Exact hand from the reference document.
			name:      "saut a 3T sur 1C : autre majeure et Carreau",
			southHand: hand("64", "AKQ85", "92", "AJ73"),
			westHand:  hand("KQT93", "7", "KJT84", "92"),
			want:      "3T", hint: "Michaël",
		},
		{
			name:      "2SA sur 1P : les deux mineures",
			southHand: hand("AKQ85", "42", "A96", "742"),
			westHand:  hand("2", "63", "KQJ85", "AQT85"),
			want:      "2SA", hint: "deux mineures",
		},
		{
			name:      "2K sur 1K : les deux majeures",
			southHand: hand("32", "94", "AKQJ8", "A962"),
			westHand:  hand("AQJ85", "KQT85", "43", "7"),
			want:      "2K", hint: "deux majeures",
		},
		{
			name:      "2SA sur 1T : Coeur et l'autre mineure",
			southHand: hand("432", "54", "K92", "AKQJ8"),
			westHand:  hand("7", "AKQJ9", "QJT84", "96"),
			want:      "2SA", hint: "Michaël",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWith(south, map[int]*Hand{south: tc.southHand, west: tc.westHand})
			calls := NewEngine(d).Run()
			for _, sc := range calls {
				if sc.Seat != west {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.want {
					t.Fatalf("West's call = %s (%s), want %s\nauction: %s", got, sc.M.fr, tc.want, formatAuction(calls))
				}
				if tc.hint != "" && !strings.Contains(sc.M.fr, tc.hint) {
					t.Fatalf("comment %q does not mention %q", sc.M.fr, tc.hint)
				}
				return
			}
			t.Fatalf("West never called\nauction: %s", formatAuction(calls))
		})
	}
}

// TestMichaelsTrapSingleSuited checks that a genuine single-suited hand
// (seven clubs, nothing else at five cards) never triggers the Michaels
// shape check: over a major opening, 3C is reserved for the Michaels jump,
// never a natural preempt (docs/addon_3.md, the mnemonic pitfall example).
func TestMichaelsTrapSingleSuited(t *testing.T) {
	h := hand("5", "T84", "K9", "KQJT973")
	eng := &Engine{}
	p := &playerState{seat: 3, hand: h}
	if _, _, ok := eng.michaelsShape(p, Spades); ok {
		t.Fatalf("single-suited seven-card club hand must not trigger the Michaels shape check")
	}
}

// TestMichaelsAdvance checks the advancer's response once the opener's
// partner has passed over the Michaels cue-bid: pick the better of partner's
// two suits and bid up to the level supported by length and values
// (docs/addon_3.md, "les développements").
func TestMichaelsAdvance(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: south,
		Hands: [4]*Hand{
			north: hand("985", "T94", "875", "T963"), // 0 HP: guaranteed pass
			east:  hand("QT3", "J762", "AQJT4", "K"),
			south: hand("64", "AKQ85", "K92", "742"), // opens 1H
			west:  hand("AKJ72", "3", "63", "AQJ85"), // cue-bids 2H (spades+clubs)
		},
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if sc.Call.Kind != KindPass {
			t.Fatalf("North (0 HP, competitive auction) should pass, got %s\nauction: %s", sc.Call.Format("fr"), formatAuction(calls))
		}
		break
	}
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if got := sc.Call.Format("fr"); got != "4P" {
			t.Fatalf("East's advance = %s (%s), want 4P\nauction: %s", got, sc.M.fr, formatAuction(calls))
		}
		if !strings.Contains(sc.M.fr, "loi des atouts") {
			t.Fatalf("comment %q does not mention the law of total tricks", sc.M.fr)
		}
		return
	}
	t.Fatalf("East never called\nauction: %s", formatAuction(calls))
}
