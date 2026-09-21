package main

import "testing"

// The overcaller's continuation after his partner's simple raise [A-7]. The
// raise caps the advance at 7-10 HLD [A-5] while the overcall itself has said
// nothing beyond 9-18 HL, so the whole decision is the overcaller's, and it is
// pure arithmetic against the 27 HLD of a major game [E-9]: pass while the
// combined *maximum* stays short of it, bid the game once the combined
// *minimum* reaches it, and propose at the three level in between.
//
// Each deal below is the same skeleton -- South opens a minor, West overcalls
// 1S, North passes, East raises to 2S -- with only West's strength moved.
func TestOvercallRebidAfterSimpleRaise(t *testing.T) {
	const west, east = 3, 1
	cases := []struct {
		name      string
		pbn       string
		wantWest  Call // West's rebid over the raise
		wantFinal Call
	}{
		{
			// West: AQJ83 K4 A72 T94 -- 16 HLD. Even facing the raise's
			// maximum the side owns 26: there is no game to look for.
			name:      "16 HLD passes",
			pbn:       `[Deal "S:95.A76.KQ986.K32 AQJ83.K4.A72.T94 762.QT985.T3.A87 KT4.J32.J54.QJ65"]`,
			wantWest:  passCall,
			wantFinal: bidSuit(2, Spades),
		},
		{
			// West: AQJ83 KQ72 A42 5 -- 19 HLD. The maximum clears 27, the
			// minimum does not: propose. East holds 9 and declines.
			name:      "19 HLD proposes, 9 declines",
			pbn:       `[Deal "S:95.A54.KJ987.AQ2 AQJ83.KQ72.A42.5 T62.J98.T6.T9843 K74.T63.Q53.KJ76"]`,
			wantWest:  bidSuit(3, Spades),
			wantFinal: bidSuit(3, Spades),
		},
		{
			// The same 19 HLD facing 10: the proposal announced a 17 floor,
			// and 10 + 17 is the game.
			name:      "19 HLD proposes, 10 accepts",
			pbn:       `[Deal "S:95.A54.KJ987.AJ2 AQJ83.KQ72.A42.5 T62.J98.T6.T9843 K74.T63.Q53.KQ76"]`,
			wantWest:  bidSuit(3, Spades),
			wantFinal: bidSuit(4, Spades),
		},
		{
			// West: AQJ832 KQ72 A4 5 -- 21 HLD. The game is there opposite
			// the raise's *floor*, so it is bid, not proposed.
			name:      "21 HLD bids the game",
			pbn:       `[Deal "S:95.A54.KJ987.AQ2 AQJ832.KQ72.A4.5 T6.J98.T62.T9843 K74.T63.Q53.KJ76"]`,
			wantWest:  bidSuit(4, Spades),
			wantFinal: bidSuit(4, Spades),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParsePBN([]byte("[Dealer \"S\"]\n[Vulnerable \"None\"]\n" + tc.pbn))
			if err != nil {
				t.Fatalf("bad deal: %v", err)
			}
			calls := NewEngine(d).Run()
			// The skeleton itself must hold, or the rebid under test is not
			// the one the rule is about.
			if c, ok := firstBidOf(calls, west); !ok || c.Call != bidSuit(1, Spades) {
				t.Fatalf("West did not overcall 1P\nauction: %s", formatAuction(calls))
			}
			if c, ok := firstBidOf(calls, east); !ok || c.Call != bidSuit(2, Spades) {
				t.Fatalf("East did not make the simple raise 2P\nauction: %s", formatAuction(calls))
			}
			rebid, ok := nthCallOf(calls, west, 2)
			if !ok {
				t.Fatalf("West never got a second turn\nauction: %s", formatAuction(calls))
			}
			if rebid.Call != tc.wantWest {
				t.Fatalf("West's rebid = %s, want %s\nauction: %s",
					rebid.Call.Format("fr"), tc.wantWest.Format("fr"), formatAuction(calls))
			}
			if contract, _, _ := finalContract(calls); contract != tc.wantFinal {
				t.Fatalf("final contract = %s, want %s\nauction: %s",
					contract.Format("fr"), tc.wantFinal.Format("fr"), formatAuction(calls))
			}
		})
	}
}

// nthCallOf returns a seat's n-th call (1-based), bids and passes alike.
func nthCallOf(calls []SeatCall, seat, n int) (SeatCall, bool) {
	seen := 0
	for _, sc := range calls {
		if sc.Seat != seat {
			continue
		}
		if seen++; seen == n {
			return sc, true
		}
	}
	return SeatCall{}, false
}
