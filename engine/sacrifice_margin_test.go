package engine

import "testing"

// TestSacrificePricedOneTrickWorse guards the margin on the law of total
// tricks (audit du par, donne 222). East opens 1S, South doubles, West raises
// to 3D then 4S, and North -- five hearts facing the double's promised four,
// so nine trumps by the law -- used to sacrifice at 5H: the law read one
// down, 100 doubled, against a non-vulnerable 4S worth 420. The law counts
// both camps' tricks together and the sacrificing side is the one without the
// honours, so the same auction really was three down, 500. Priced one trick
// worse, 300 against 420 still pays on paper -- but the sacrifice is now
// refused because the count is nine trumps for a contract needing eleven
// tricks, two down at par, three once the margin is applied.
func TestSacrificePricedOneTrickWorse(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "None"]
[Deal "E:AKT543.Q4.74.K94 J8.A873.J63.AQ82 Q972.K9.KQT82.JT 6.JT652.A95.7653"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	for _, sc := range calls {
		if sc.Seat != north || !sc.Call.IsBid() {
			continue
		}
		if sc.Call.Level >= 5 {
			t.Fatalf("North sacrifices at %s (%s): nine trumps do not pay for eleven tricks\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
}

// TestSacrificeSpendsTheFitOnce guards the second half of the fix (audit du
// par, donne 327). North, a bust facing South's six spades, sacrificed at 4S
// over their heart game; West bid on to 5H, and North sacrificed *again* at
// 5S with the very same nine trumps -- doubled, four down, 1100 against a par
// of 660. The law gives one total per deal: a fit buys one level, not one per
// round of the auction.
func TestSacrificeSpendsTheFitOnce(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "All"]
[Deal "S:JT8632..AQ82.A42 Q5.AKQ752.5.KQT9 974.643.964.8753 AK.JT98.KJT73.J6"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	spent := 0
	for _, sc := range calls {
		if sideOf(sc.Seat) == 0 && sc.M.lawBid && sc.M.lawSuit == Spades {
			spent++
		}
	}
	if spent > 1 {
		t.Fatalf("North-South buy %d levels in spades on the law, want at most 1\nauction: %s",
			spent, formatAuction(calls))
	}
	if spent == 0 {
		t.Fatalf("North-South never took the paying sacrifice\nauction: %s", formatAuction(calls))
	}
}
