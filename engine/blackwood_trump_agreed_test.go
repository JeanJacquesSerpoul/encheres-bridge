package main

import (
	"strings"
	"testing"
)

// TestBlackwoodNeedsAnAgreedTrump checks that keycard Blackwood is not asked on
// a suit the partnership never agreed. West holds AKJT82 - KJT643 4 and hears
// 1D - 1S - 1NT: the diamond "fit" exists only in West's own head, from the
// opening's three-card promise plus its own six. Asking 4NT there would have
// East answering "two keycards and the trump queen" about a suit nobody named
// -- the trump king and queen the answers count would be undefined. West names
// the suit first; the ask follows once diamonds are on the table.
func TestBlackwoodNeedsAnAgreedTrump(t *testing.T) {
	const east, west = 1, 3
	const pbn = `[Dealer "E"]
[Vulnerable "NS"]
[Deal "E:74.Q752.AQ75.AJ6 95.T864.98.KT987 AKJT82..KJT643.4 Q63.AKJ93.2.Q532"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{west, 1, "3K", "propose l'atout"}, // names the suit rather than asking
		{east, 2, "4T", "contrôle"},        // diamonds agreed: the controls run
		{east, 3, "4SA", "cartes clefs"},   // only now is the ask about a known trump
		{west, 3, "5C", "clefs"},
	}
	for _, w := range seq {
		n, found := 0, false
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
					t.Fatalf("seat %s call #%d comment %q does not mention %q", seatNames[w.seat], w.nth, sc.M.fr, w.hint)
				}
				found = true
				break
			}
			n++
		}
		if !found {
			t.Fatalf("seat %s made fewer than %d calls\nauction: %s", seatNames[w.seat], w.nth+1, formatAuction(calls))
		}
	}
}
