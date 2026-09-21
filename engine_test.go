package main

import (
	"math/rand"
	"strings"
	"testing"
)

func dealFrom(rng *rand.Rand) *Deal {
	cards := make([]int, 52)
	for i := range cards {
		cards[i] = i
	}
	rng.Shuffle(52, func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	var hands [4]*Hand
	for seat := 0; seat < 4; seat++ {
		h := &Hand{}
		var suits [4][]byte
		for _, c := range cards[seat*13 : seat*13+13] {
			suits[c/13] = append(suits[c/13], rankOrder[c%13])
		}
		for s := Clubs; s <= Spades; s++ {
			h.Suits[s] = sortRanks(string(suits[s]))
		}
		hands[seat] = h
	}
	return &Deal{Dealer: rng.Intn(4), Hands: hands}
}

// TestAuctionsTerminateAndAreLegal fuzzes the engine with random deals.
func TestAuctionsTerminateAndAreLegal(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	maxLen := 0
	for i := 0; i < 2000; i++ {
		deal := dealFrom(rng)
		calls := NewEngine(deal).Run()
		if len(calls) > maxLen {
			maxLen = len(calls)
		}
		if len(calls) >= 40 {
			t.Fatalf("deal %d: auction hit the safety cap (%d calls)", i, len(calls))
		}
		// Check seat rotation and call legality.
		var last Call
		hasBid := false
		lastBidSide := -1
		seat := deal.Dealer
		for k, sc := range calls {
			if sc.Seat != seat {
				t.Fatalf("deal %d call %d: expected seat %d, got %d", i, k, seat, sc.Seat)
			}
			switch sc.Call.Kind {
			case KindBid:
				if hasBid && !sc.Call.higherThan(last) {
					t.Fatalf("deal %d call %d: insufficient bid %v over %v", i, k, sc.Call, last)
				}
				last, hasBid, lastBidSide = sc.Call, true, sideOf(sc.Seat)
			case KindDouble:
				if !hasBid || lastBidSide == sideOf(sc.Seat) {
					t.Fatalf("deal %d call %d: illegal double", i, k)
				}
			}
			seat = (seat + 1) % 4
		}
		// Auction must end with three passes after a bid, or four passes.
		n := len(calls)
		if hasBid {
			if n < 4 || calls[n-1].Call.Kind != KindPass || calls[n-2].Call.Kind != KindPass || calls[n-3].Call.Kind != KindPass {
				t.Fatalf("deal %d: auction did not end with three passes", i)
			}
		} else if n != 4 {
			t.Fatalf("deal %d: passed-out auction has %d calls", i, n)
		}
	}
	t.Logf("longest auction over 2000 random deals: %d calls", maxLen)
}

func TestParsePBN(t *testing.T) {
	good := `[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`
	d, err := ParsePBN([]byte(good))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Dealer != 0 {
		t.Fatalf("dealer = %d, want 0 (N)", d.Dealer)
	}
	if d.Hands[0].Suits[Spades] != "AKQ" || d.Hands[0].H() != 20 {
		t.Fatalf("north hand parsed wrong: %+v", d.Hands[0])
	}
	// Rotation starting from another seat.
	rotated := `[Dealer "E"]
[Deal "W:T543.52.JT8.KT54 AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76"]`
	d2, err := ParsePBN([]byte(rotated))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d2.Hands[0].Suits[Spades] != "AKQ" {
		t.Fatalf("rotation broken: north spades = %q", d2.Hands[0].Suits[Spades])
	}
	// A void written as "-" instead of an empty suit.
	void := `[Board "1"]
[Dealer "N"]
[Declarer "E"]
[Vulnerable "None"]
[Deal "N:K94.9.Q72.J98742 QJ73.QJ.AJT4.Q65 A82.T654.85.AKT3 T65.AK8732.K963.-"]`
	d3, err := ParsePBN([]byte(void))
	if err != nil {
		t.Fatalf("unexpected error for '-' void: %v", err)
	}
	if d3.Hands[3].Len(Clubs) != 0 || d3.Hands[3].Len(Hearts) != 6 {
		t.Fatalf("west hand parsed wrong: %+v", d3.Hands[3])
	}
	for _, bad := range []string{
		`[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`, // no dealer
		`[Dealer "N"]`, // no deal
		`[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT5"]`, // 12 cards
		`[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 A9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`, // duplicate ace
	} {
		if _, err := ParsePBN([]byte(bad)); err == nil {
			t.Fatalf("expected error for %s", strings.Split(bad, "\n")[0])
		}
	}
}
