package engine

import "testing"

// TestNoFiveCardOvercallOverTheirNotrump is a regression test for a reported
// deal: North opened 1NT and East overcalled 2H on AKJ95 — five cards. The
// "or 14 HL" clause of [I-5], written for overcalls in front of an unlimited
// one-of-a-suit opening, was letting honour strength stand in for length in
// front of a hand that has just announced 15-17 to within two points.
//
// Over their 1NT the natural overcall promises six cards [I-5b]. East, with
// nothing else to say, passes.
func TestNoFiveCardOvercallOverTheirNotrump(t *testing.T) {
	const pbn = `[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:AK7.872.KJ52.A63 Q86.AKJ95.A84.Q8 2.QT63.9763.J952 JT9543.4.QT.KT74"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1
	for _, sc := range calls {
		if sc.Seat == east && sc.Call.IsBid() {
			t.Fatalf("East bid %s (%s) on a five-card suit over their 1NT\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
}

// TestSixCardOvercallOverTheirNotrumpStillBids checks the rule bites on the
// length and not on the strength: the same auction with a six-card suit
// intervenes as before.
func TestSixCardOvercallOverTheirNotrumpStillBids(t *testing.T) {
	const pbn = `[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:AK7.872.KJ52.A63 Q8.AKJ953.A84.Q8 2.QT6.9763.J9542 JT96543.4.QT.KT7"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1
	for _, sc := range calls {
		if sc.Seat == east {
			if got := sc.Call.Format("fr"); got != "2C" {
				t.Fatalf("East's call over their 1NT = %s (%s), want 2C\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			return
		}
	}
	t.Fatalf("East never called\nauction: %s", formatAuction(calls))
}
