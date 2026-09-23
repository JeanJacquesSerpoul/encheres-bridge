package main

import "testing"

// TestNoCheapTwoSuiterOnRegularMinimum checks that opener passes the 1NT
// response with a regular minimum instead of naming a four-card second suit.
//
// East opens 1D on K82.JT.AJ87.KQ43 -- 4-4-3-2, 14 HCP -- and West answers 1NT
// (6-10, no major). The engine rebid 2C, but the cheap second suit "promet une
// main irrégulière, sauf exception" (docs/bidings.md, "LE BICOLORE
// ÉCONOMIQUE"), the exception being the 15-17H false two-suiter. Here game is
// out of reach (14 + 10 at most) and the 4-4 minor tells partner nothing he can
// use: 1NT is the better contract, so East must pass.
func TestNoCheapTwoSuiterOnRegularMinimum(t *testing.T) {
	pbn := `[Dealer "E"]
[Deal "S:J653.KQ8.3.A7652 74.A96.K9542.T98 AQT9.75432.QT6.J K82.JT.AJ87.KQ43"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1 // seats: 0=N, 1=E, 2=S, 3=W
	// East's second call (after 1D) must be a pass, not the 2C two-suiter.
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 1 {
			if sc.Call.Kind != KindPass {
				t.Fatalf("East's rebid = %s (%s), want Passe\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}
}

// TestCheapTwoSuiterKeptOnIrregular guards the other side of the same rule: an
// irregular hand still names its cheap second suit over the 1NT response.
// South opens 1H on A32.K9842.4.AQ72 (3-5-1-4, 13 HCP) and must rebid 2C --
// there the second suit improves the contract and the singleton diamond makes
// notrump the wrong strain.
func TestCheapTwoSuiterKeptOnIrregular(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:A32.K9842.4.AQ72 KQT987.7.JT82.84 64.AQ.Q965.J9653 J5.JT653.AK73.KT"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	n := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "2T" {
				t.Fatalf("South's rebid = %s (%s), want 2T (bicolore économique)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}
}
