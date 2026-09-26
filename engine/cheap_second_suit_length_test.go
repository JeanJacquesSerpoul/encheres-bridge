package engine

import "testing"

// TestCheapSecondSuitLengthensTheOpening: 1D - 1S - 2C shows four clubs and,
// the clubs ranking below the diamonds, four diamonds at least [RO-19c]. West
// holds KJT7 Q54 9842 A4: four diamonds facing four is the eight-card fit,
// two clubs facing four is not. Without the inference West read the opening's
// "three or more", found no fit and passed 2C. West must not leave the side
// in clubs.
func TestCheapSecondSuitLengthensTheOpening(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "S"]
[Deal "N:6542.KT82.J6.JT8 Q3.A6.AKT73.K653 A98.J973.Q5.Q972 KJT7.Q54.9842.A4"]`)
	calls := NewEngine(d).Run()
	const east, west = 1, 3
	wantCall(t, calls, east, 2, "2T")
	sc, _ := nthCallOf(calls, east, 2)
	if sc.M.lens[Diamonds] < 4 {
		t.Fatalf("2C records %d diamonds, want 4+", sc.M.lens[Diamonds])
	}
	w, _ := nthCallOf(calls, west, 3)
	if w.Call.Kind == KindPass {
		t.Fatalf("West passed 2C with four diamonds and two clubs\nauction: %s", formatAuction(calls))
	}
}
