package main

import (
	"testing"
)

// TestLandyReservesTwoClubs: 2C over their 1NT is the Landy bid [I-3b], so
// the seat that owns the convention cannot also use it naturally. East holds
// JT6 6 A63 AQJ874 -- a genuine club one-suiter worth a two-level overcall
// anywhere else -- and must stay silent here: bidding 2C would announce both
// majors, and partner would answer in one of them.
func TestLandyReservesTwoClubs(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "N"]
[Deal "S:A42.K74.T75.T632 Q853.QJT953.J9.9 K97.A82.KQ842.K5 JT6.6.A63.AQJ874"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat == east && sc.Call == bid(2, SClubs) {
			t.Fatalf("East overcalled 2T naturally (%s) in the Landy seat\nauction: %s",
				sc.M.fr, formatAuction(calls))
		}
	}
	if contract, _, _ := finalContract(calls); contract != bid(1, SNoTrump) {
		t.Fatalf("final contract = %s, want 1SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestNaturalTwoClubsSurvivesElsewhere: the reservation is worth only the one
// seat the convention lives in. Two passes later the 1NT has been answered,
// Landy is off, and the same club suit is bid naturally.
func TestNaturalTwoClubsSurvivesElsewhere(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("K97", "A82", "KQ842", "K5"),
		east:  hand("Q8532", "QT953", "J9", "T"),
		south: hand("A4", "K74", "T753", "9632"),
		west:  hand("JT6", "J6", "A6", "AQJ874"),
	})
	calls := NewEngine(d).Run()
	found := false
	for _, sc := range calls {
		if sc.Seat == west && sc.Call == bid(2, SClubs) {
			found = true
		}
	}
	if !found {
		t.Fatalf("West never bid its clubs from the balancing seat\nauction: %s", formatAuction(calls))
	}
}
