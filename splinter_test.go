package main

import (
	"strings"
	"testing"
)

// dealWithQuietOpponents builds a deal where the given seats hold exactly
// the supplied hands; the remaining cards are dealt to the other two seats
// by alternating rank within each suit, so high cards split evenly between
// them instead of landing in one stray strong hand that would intervene.
func dealWithQuietOpponents(dealer int, given map[int]*Hand) *Deal {
	var used [4]map[byte]bool
	for s := Clubs; s <= Spades; s++ {
		used[s] = map[byte]bool{}
	}
	for _, h := range given {
		for s := Clubs; s <= Spades; s++ {
			for i := 0; i < len(h.Suits[s]); i++ {
				used[s][h.Suits[s][i]] = true
			}
		}
	}
	var hands [4]*Hand
	var empty []int
	for seat := 0; seat < 4; seat++ {
		if h, ok := given[seat]; ok {
			hands[seat] = h
			continue
		}
		hands[seat] = &Hand{}
		empty = append(empty, seat)
	}
	idx := 0
	for s := Clubs; s <= Spades; s++ {
		for i := 0; i < len(rankOrder); i++ {
			r := rankOrder[i]
			if used[s][r] {
				continue
			}
			seat := empty[idx%len(empty)]
			hands[seat].Suits[s] += string(r)
			idx++
		}
	}
	for _, seat := range empty {
		for s := Clubs; s <= Spades; s++ {
			hands[seat].Suits[s] = sortRanks(hands[seat].Suits[s])
		}
	}
	return &Deal{Dealer: dealer, Hands: hands}
}

// TestSplinterResponse checks the specified Splinter (docs/addon_5.md): a
// double jump over 1H/1S showing 4+ trumps, a singleton or void elsewhere,
// no good 5+ card side suit, and 13-15 HLD. The exception (1H-3S instead of
// the expected 1H-4S) follows automatically from the formula: spades outrank
// hearts, so the cheapest spade reply is already at the one level.
func TestSplinterResponse(t *testing.T) {
	const north, south = 0, 2
	cases := []struct {
		name       string
		southHand  *Hand
		northHand  *Hand
		wantOpen   string
		wantAnswer string
		hint       string
	}{
		{
			name:       "1C - 4T : chicane/singleton Trefle",
			southHand:  hand("3", "AQT76", "K98", "AKJ4"),
			northHand:  hand("Q987", "KJ98", "AQ32", "5"),
			wantOpen:   "1C",
			wantAnswer: "4T",
			hint:       "Splinter",
		},
		{
			// Exception: over 1H, spades rank above hearts, so the double
			// jump lands on 3S rather than 4S.
			name:       "1C - 3P : exception, singleton Pique",
			southHand:  hand("QJ32", "AK765", "98", "A8"),
			northHand:  hand("5", "QJ98", "AJ73", "QJ42"),
			wantOpen:   "1C",
			wantAnswer: "3P",
			hint:       "Splinter",
		},
		{
			name:       "1P - 4C : singleton Coeur",
			southHand:  hand("AJT96", "K843", "K92", "A"),
			northHand:  hand("KQ87", "5", "AJT6", "Q987"),
			wantOpen:   "1P",
			wantAnswer: "4C",
			hint:       "Splinter",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWithQuietOpponents(south, map[int]*Hand{south: tc.southHand, north: tc.northHand})
			calls := NewEngine(d).Run()

			n := 0
			for _, sc := range calls {
				if sc.Seat != south {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.wantOpen {
					t.Fatalf("South's opening = %s, want %s\nauction: %s", got, tc.wantOpen, formatAuction(calls))
				}
				break
			}
			for _, sc := range calls {
				if sc.Seat != north {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.wantAnswer {
					t.Fatalf("North's answer = %s (%s), want %s\nauction: %s", got, sc.M.fr, tc.wantAnswer, formatAuction(calls))
				}
				if !strings.Contains(sc.M.fr, tc.hint) {
					t.Fatalf("comment %q does not mention %q", sc.M.fr, tc.hint)
				}
				n++
				break
			}
			if n == 0 {
				t.Fatalf("North never called\nauction: %s", formatAuction(calls))
			}
		})
	}
}

// TestSplinterTrapGoodSideSuit checks that a good 5+ card side suit
// disqualifies the Splinter (docs/addon_5.md, "Pas de Splinter" example):
// with a fit and a singleton, but also a good five-card spade suit, the
// hand shows the spade suit naturally instead of splintering in diamonds.
func TestSplinterTrapGoodSideSuit(t *testing.T) {
	const north, south = 0, 2
	southHand := hand("9", "AKQT6", "K98", "A654") // opens 1H
	northHand := hand("KJ643", "J732", "6", "KQ7") // 5-card spade suit, singleton diamond
	d := dealWithQuietOpponents(south, map[int]*Hand{south: southHand, north: northHand})
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if sc.Call.Level >= 3 {
			t.Fatalf("North should not splinter with a good five-card side suit, got %s (%s)\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		if got := sc.Call.Format("fr"); got != "1P" {
			t.Fatalf("North's answer = %s (%s), want 1P (natural spade suit)\nauction: %s", got, sc.M.fr, formatAuction(calls))
		}
		return
	}
	t.Fatalf("North never called\nauction: %s", formatAuction(calls))
}
