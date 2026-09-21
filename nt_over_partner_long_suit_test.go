package main

import (
	"strings"
	"testing"
)

// TestNotrumpOverPartnerLongSuit checks the nine-tricks-off-the-top decision:
// North repeats his diamonds at the three level over the opponents' club
// auction, South holds AJ53 in clubs and two diamonds, and 3NT is the contract
// -- not the partscore the plain honour count would settle for (9 + 12 = 21,
// four short of the 25 the 3NT threshold usually asks), nor five of a minor,
// which needs two more tricks for the same bonus.
func TestNotrumpOverPartnerLongSuit(t *testing.T) {
	const north, south = 0, 2
	const pbn = `[Dealer "N"]
[Vulnerable "All"]
[Deal "N:A5.73.AKQT8742.7 K63.AQJ9.9.KQ864 T2.KT862.J5.AJ53 QJ9874.54.63.T92"]`

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
		{south, 1, "3SA", "arrêt dans leur couleur"},
		{north, 2, "Passe", ""}, // never pulled to 5D
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
	if got := formatAuction(calls); strings.Contains(got, "N:5K") {
		t.Fatalf("North pulled 3NT to five of a minor\nauction: %s", got)
	}
}
