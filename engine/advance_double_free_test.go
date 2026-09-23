package main

import "testing"

// TestAdvanceFreeAfterInterventionOverDouble checks that the answer to a takeout
// double is no longer obligatory once the opponents bid over it. Sequence:
// N 1H, E pass, S 2H, W double, N 4H, E ? — West's double is a takeout double of
// hearts, but North's 4H frees East: holding J9653.7.KT953.KT (5H, only worth a
// minimum answer) East must pass rather than describe a forced 0-7 answer with 4S.
func TestAdvanceFreeAfterInterventionOverDouble(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:8.KQJ9643.AJ6.QJ J9653.7.KT953.KT QT7.AT82.87.9865 AK42.5.Q42.A7432"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1 // seats: 0=N, 1=E, 2=S, 3=W
	// East's *second* call (its first is the opening pass) must be a pass: the
	// opponents' 4H over the double removes the obligation to answer.
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "Passe" {
				t.Fatalf("East advance = %s (%s), want Passe (free after intervention over the double)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("East never made a second call\nauction: %s", formatAuction(calls))
}
