package main

import (
	"strings"
	"testing"
)

// TestResponderSixMajorGameOverNTRebid checks that responder, holding an
// unbalanced hand with a six-card major and game-going values, concludes in
// four of that major rather than 3NT after opener's balanced notrump rebid.
// Opener's 1SA rebid is balanced (docs/bidings.md: 12-14H régulier), so it
// guarantees at least a doubleton support — a 6-2 fit — and with a singleton
// in responder's hand notrump would run off the top.
func TestResponderSixMajorGameOverNTRebid(t *testing.T) {
	// N = K87653.A5.Q.AQ96 : 15H, 6-2-1-4 (singleton diamond), 6 spades.
	// S opens 1D, N responds 1S (forcing), S rebids 1SA (12-14 balanced).
	pbn := `[Dealer "S"]
[Deal "S:AJ.KJ84.T9843.K3 Q2.Q9732.K62.T54 K87653.A5.Q.AQ96 T94.T6.AJ75.J872"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0 // seats: 0=N, 1=E, 2=S, 3=W
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 { // North's rebid over 1SA
			if got := sc.Call.Format("fr"); got != "4P" {
				t.Fatalf("North rebid = %s (%s), want 4P (spade game)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("North made fewer than 2 calls\nauction: %s", formatAuction(calls))
}

// TestResponderBalancedStaysInNT guards the sibling case: with a *balanced*
// hand and no six-card major, responder should still choose 3NT, not a suit
// game. This keeps the six-card-major promotion from over-firing.
func TestResponderBalancedStaysInNT(t *testing.T) {
	// N = KQ54.A5.AJ7.Q963 : 16H, balanced 4-2-3-4 (no six-card major).
	pbn := `[Dealer "S"]
[Deal "S:AJ6.T83.KQ98.K72 T98.KJ62.654.J54 KQ54.A5.AJ7.Q963 732.Q974.T32.AT8"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); strings.HasPrefix(got, "4") && strings.HasSuffix(got, "P") {
				t.Fatalf("balanced North jumped to a spade game %s (%s); expected notrump\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("North made fewer than 2 calls\nauction: %s", formatAuction(calls))
}
