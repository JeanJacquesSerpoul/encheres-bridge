package main

import (
	"strings"
	"testing"
)

// TestPartscoreLawOfTotalTricks: North opens 1C on AT86 Q 87 AKT987 and
// repeats the clubs; South (4 K9876 JT42 QJ2) holds three of them, so the
// side owns a nine-card fit. West overcalls 1S, East raises to 2S, and the
// point count -- 26 combined, five short of the minor game [E-9] -- has
// nothing more to say. The law of total tricks does: nine trumps are worth
// the three level whatever the honour count, and leaving the opponents a
// partscore while nine trumps sit unused is exactly what it forbids [L-1].
func TestPartscoreLawOfTotalTricks(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:4.K9876.JT42.QJ2 QJ975.JT2.AQ3.43 AT86.Q.87.AKT987 K32.A543.K965.65"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, declarer, _ := finalContract(calls)
	if contract != bid(3, SClubs) {
		t.Fatalf("final contract = %s, want 3T (the nine-card fit competes)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	if sideOf(declarer) != 0 {
		t.Fatalf("declarer = %s, want the North-South side\nauction: %s", seatNames[declarer], formatAuction(calls))
	}
	found := false
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "loi des levées totales") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the competitive bid does not read as the law of total tricks\nauction: %s", formatAuction(calls))
	}
}

// TestPartscoreLawStopsAtItsOwnLevel: the same law that buys the three level
// forbids the fourth on nine trumps, and the level it affords is also the
// ceiling of the rule -- once our side has bid it, the cheapest call in the
// fit lands above the law and the rule goes quiet. Nothing on this deal, on
// either side, may reach the four level.
func TestPartscoreLawStopsAtItsOwnLevel(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:4.K9876.JT42.QJ2 QJ975.JT2.AQ3.43 AT86.Q.87.AKT987 K32.A543.K965.65"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Call.IsBid() && sc.Call.Level >= 4 {
			t.Fatalf("%s bid %s past the level nine trumps afford\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), formatAuction(calls))
		}
	}
}
