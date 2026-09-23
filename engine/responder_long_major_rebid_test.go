package main

import (
	"strings"
	"testing"
)

// TestResponderRebidsLongMajorNotNotrump reproduces a reported deal where
// South, after 1C - 1S - 2C, proposed game with a natural 2NT despite holding
// six spades and a club void. Bidding notrump opposite a void in the suit
// partner has just rebid is absurd: it would be played off the top. With an
// unbalanced hand and a six-card major already shown, responder must rebid the
// major (2S) rather than notrump, which here lands the far better 4S game.
func TestResponderRebidsLongMajorNotNotrump(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("Q2", "AK94", "K4", "AT853"), // 16 HCP, opens 1C, rebids 2C
		south: hand("AJ9873", "J6", "QJ932", ""), // 6 spades, void in clubs
		east:  hand("T4", "752", "T876", "KQ42"),
		west:  hand("K65", "QT83", "A5", "J976"),
	})

	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 0, "1T", ""},            // 1C opening
		{south, 0, "1P", "1 sur 1"},     // 1S response, 4+ spades
		{north, 1, "2T", ""},            // 2C rebid, six clubs
		{south, 1, "2P", "proposition"}, // rebids the six-card major, NOT 2NT
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

	// South must never propose notrump with a void in partner's rebid suit.
	for _, sc := range calls {
		if sc.Seat == south && sc.Call.Format("fr") == "2SA" {
			t.Fatalf("South bid 2SA with a club void\nauction: %s", formatAuction(calls))
		}
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P (spade game)\nauction: %s", got, formatAuction(calls))
	}
}
