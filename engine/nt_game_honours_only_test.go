package main

import (
	"testing"
)

// TestNoNTOnBidAndRaisedSuitWithLoneStopper covers two related defects on the
// same deal. North opens 1D; E overcalls 1H, S raises to 2D (11+ HLD), W raises
// to 2H. North holds KQJ3.A7.65432.Q3.
//
//  1. Point count: North is 12 HCP but 15 HLD once the two doubletons count for
//     the diamond fit. Those distribution points buy no notrump tricks, so the
//     combined honour strength is only 12+11 = 23 -- short of game.
//  2. Stopper: the opponents have bid and raised hearts (a running eight-card
//     suit), and North holds only Ax there -- a lone stopper dislodged at once.
//
// Either way North must not offer notrump; it competes in its diamond fit (3D).
func TestNoNTOnBidAndRaisedSuitWithLoneStopper(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:KQJ3.A7.65432.Q3 875.QJT32.A.J542 A92.865.QJT7.A97 T64.K94.K98.KT86"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0 // seats: 0=N, 1=E, 2=S, 3=W
	// North's second call (after the 1D opening) is the rebid over the raise.
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if sc.Call.Kind == KindBid && sc.Call.Strain == SNoTrump {
				t.Fatalf("North rebid = %s (%s): offered notrump with a lone heart stopper on 23 combined honours\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("North never made a second call\nauction: %s", formatAuction(calls))
}
