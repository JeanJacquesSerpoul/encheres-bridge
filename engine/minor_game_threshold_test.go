package main

import (
	"testing"
)

// TestMinorGameNeedsThirty: North opens 1C on AT86 Q 87 AKT987 and repeats
// the clubs over South's 1H; South (4 K9876 JT42 QJ2) holds three clubs, so
// the side owns a nine-card fit and 20 honour points between them. Five clubs
// is eleven tricks and [E-9] prices it at 30 HLD combined -- the pair has 26,
// and no stopper in the spades the opponents have bid, so 3NT is not on the
// table either. The invitation ladder used to price every minor fit at the
// notrump figure of 25 and drove this deal to 5C, two off.
func TestMinorGameNeedsThirty(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:4.K9876.JT42.QJ2 QJ975.JT2.AQ3.43 AT86.Q.87.AKT987 K32.A543.K965.65"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Call.IsBid() && sc.Call.Level >= 5 {
			t.Fatalf("%s bid %s (%s) on 26 combined HLD\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
}
