package engine

import "testing"

// TestOneLevelOvercallOnOpeningValues is the reported deal. East opens 1D and
// South, with J9754 AQ65 K5 Q2, used to pass: the five spades hold a single
// honour, so [I-5] refused the natural overcall, and five cards in a major
// bar the takeout double [I-6]. Twelve honour points are an opening, and an
// opening speaks: South bids 1S [I-5c].
func TestOneLevelOvercallOnOpeningValues(t *testing.T) {
	const south = 2
	d, err := ParsePBN([]byte(`[Dealer "E"]
[Vulnerable "None"]
[Deal "N:AKQ.82.QT643.K83 8.KJT9.A9872.AT6 J9754.AQ65.K5.Q2 T632.743.J.J9754"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if want := bid(1, SSpades); sc.Call != want {
			t.Fatalf("South's first call = %s (%s), want 1P\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		if sc.M.lens[Spades] < 5 {
			t.Fatalf("1P records %d spades, want 5\nauction: %s",
				sc.M.lens[Spades], formatAuction(calls))
		}
		return
	}
	t.Fatalf("South made no call\nauction: %s", formatAuction(calls))
}
