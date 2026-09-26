package engine

import "testing"

// TestOpenerOneLevelNewSuitForcing checks that responder does not pass opener's
// new suit at the one level [RO-19]. North opened 1C, South answered 1H, North
// rebid 1S: South, a 7 H minimum, used to pass it out in a 4-3 fit.
func TestOpenerOneLevelNewSuitForcing(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "S:962.J764.KQ64.J2 QJ8.KQ952.AT7.43 AKT5.AT.32.KQT75 743.83.J985.A986"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	if len(calls) < 7 {
		t.Fatalf("auction too short: %s", formatAuction(calls))
	}
	if r := calls[4].Call; !r.IsBid() || r.Level != 1 || r.Strain != SSpades {
		t.Skipf("North no longer rebids 1S (%s): scenario gone\nauction: %s", r.Format("fr"), formatAuction(calls))
	}
	if calls[6].Call.Kind == KindPass {
		t.Fatalf("South passed opener's forcing 1S\nauction: %s", formatAuction(calls))
	}
}
