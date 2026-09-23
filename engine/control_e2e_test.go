package engine

import (
	"strings"
	"testing"
)

// TestControlBidEndToEnd replays a full, naturally-generated auction where a
// fit is found on opener's second suit (hearts, after a 1S opening), and the
// combined side explores slam through control bids. Economic order is read on
// the ladder: over 3H the cheapest control bid is 3S, so West starts there
// with the spade ace [S-2b]. The exchange then runs 4C - 4D, East is out of
// room below 4H and asks, and the pair bids the small slam (docs/addon_4.md).
func TestControlBidEndToEnd(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: west,
		Hands: [4]*Hand{
			north: hand("JT", "A85", "J532", "9865"),
			east:  hand("72", "K972", "K8", "KJT72"),
			south: hand("Q64", "QT", "QT9764", "Q4"),
			west:  hand("AK9853", "J643", "A", "A3"),
		},
	}
	calls := NewEngine(d).Run()

	want := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{west, 0, "1P", ""},             // 1S
		{east, 0, "2T", ""},             // 2C
		{west, 1, "2C", ""},             // 2H
		{east, 1, "3C", ""},             // 3H
		{west, 2, "3P", "l'As"},         // 3S: the cheapest control bid, the ace
		{east, 2, "4T", "le Roi"},       // 4C control bid, the king
		{west, 3, "4K", "l'As"},         // 4D control bid, the ace
		{east, 3, "4SA", "Blackwood"},   // no cue room left below 4H: keycard ask
		{west, 4, "5T", "cartes clefs"}, // 5C: 0 or 3 keycards
		{east, 4, "6C", "chelem"},       // 6H, once the controls are located
	}
	for _, w := range want {
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
	if got := contract.Format("fr"); got != "6C" {
		t.Fatalf("final contract = %s, want 6C (6H)\nauction: %s", got, formatAuction(calls))
	}
}

// TestControlBidBothSidesContinue is a regression test for two reported
// bugs on the same deal: (1) a control bid didn't record the agreed trump
// length, so the responder's partner (here East, the opener) could not
// detect the fit on their own next turn and fell back to a generic
// "préférence" instead of continuing the control-bid sequence; (2) the
// trump suit itself was silently excluded from the cue rotation.
//
// West holds three-card heart support that the auction has never mentioned,
// so [S-0] makes him say the trump before any control: 3H, forcing. The
// exchange then runs on an agreed trump -- East's conventional 3NT doubt
// (a singleton club in West's suit), West's club ace, East's diamond ace --
// which is what the two bugs above broke.
func TestControlBidBothSidesContinue(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: north,
		Hands: [4]*Hand{
			north: hand("JT65", "J5", "T862", "KT8"),
			east:  hand("98", "KQ983", "AKJ75", "5"),
			south: hand("AQ3", "642", "943", "J942"),
			west:  hand("K742", "AT7", "Q", "AQ763"),
		},
	}
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if strings.Contains(sc.M.fr, "préférence") {
			t.Fatalf("East fell back to a generic preference instead of continuing the control-bid sequence\nauction: %s", formatAuction(calls))
		}
	}

	want := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{west, 1, "3C", "l'atout avant les contrôles"}, // 3H forcing raise: the trump first
		{east, 2, "3SA", "oui mais"},                   // conventional doubt, singleton club
		{west, 2, "4T", "l'As"},                        // 4C control bid, the ace of clubs
		{east, 3, "4K", "l'As"},                        // 4D control bid, the ace of diamonds
	}
	for _, w := range want {
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
				if !strings.Contains(sc.M.fr, w.hint) {
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
}

// TestControlBidThirdAceViaBlackwood is a regression test for a reported
// case: once West has shown an ace by cue-bidding, cueing spades to show a
// further one would already cross the hearts game level — structurally
// impossible, not a bug. Showing an ace by control bid raises the bidder's
// own promised floor, so the side recognizes the real combined strength and
// asks Blackwood instead of signing off (docs/addon_4.md); here West asks and
// hears every keycard plus the trump queen.
//
// The pair stops in six, not seven, and that is the price of [S-0]. Double
// dummy the grand makes — par is 13 tricks — and the old engine bid it: West
// cued three times without ever naming the trump, and each cue lifted his
// promised floor until the combined minimum cleared the 37 the grand slam
// asks for. One of those rounds now goes to the forcing 3H raise instead, so
// the count reaches 35 and stops at the small slam. The grand was reachable
// only through cue-bids on a fit nobody had expressed, which is exactly the
// route the rule closes: a control on an unnamed trump tells partner a
// keycard he cannot place. If the missed grand is judged the greater cost,
// the lever is the 37-point bar in keycardAnswer, not the fit requirement.
func TestControlBidThirdAceViaBlackwood(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := &Deal{
		Dealer: north,
		Hands: [4]*Hand{
			north: hand("JT65", "J5", "T862", "KT8"),
			east:  hand("98", "KQ983", "AKJ75", "5"),
			south: hand("KQ3", "642", "943", "J942"),
			west:  hand("A742", "AT7", "Q", "AQ763"),
		},
	}
	calls := NewEngine(d).Run()

	want := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{west, 1, "3C", "l'atout avant les contrôles"}, // 3H forcing raise: the trump first
		{west, 2, "4T", "l'As"},                        // 4C control bid, the ace of clubs
		{east, 3, "4K", "l'As"},                        // 4D control bid, the ace of diamonds
		{west, 3, "4SA", "Blackwood"},                  // spades cannot be cued below 4H: ask
	}
	for _, w := range want {
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
				if !strings.Contains(sc.M.fr, w.hint) {
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
	if got := contract.Format("fr"); got != "6C" {
		t.Fatalf("final contract = %s, want 6C (6H, small slam)\nauction: %s", got, formatAuction(calls))
	}
}
