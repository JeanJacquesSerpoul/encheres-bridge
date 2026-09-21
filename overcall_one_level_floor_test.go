package main

import "testing"

// TestNoOneLevelOvercallOnLengthPoints is the reported deal. West opens 1C and
// North, with KJT76 T3 Q87654 -, used to overcall 1S: nine HL, the floor of
// the one-level overcall, but six of them in honours. Length points alone do
// not make an overcall [I-5]: at the one level it takes 8 H on top of the good
// five-card suit, and North passes.
func TestNoOneLevelOvercallOnLengthPoints(t *testing.T) {
	const north = 0
	d, err := ParsePBN([]byte(`[Dealer "W"]
[Vulnerable "None"]
[Deal "W:AQ32.J6.A3.AKT53 KJT76.T3.Q87654. 94.KQ985.T2.Q986 85.A742.KJ9.J742"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if sc.Call.Kind != KindPass {
			t.Fatalf("North's first call = %s (%s), want Passe\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		return
	}
	t.Fatalf("North made no call\nauction: %s", formatAuction(calls))
}
