package engine

import (
	"strings"
	"testing"
)

// TestAdvanceDoubleLawSevenCardMajor reproduces the deal where East, holding a
// seven-card spade suit (QJ65432) but only 5H, must advance West's takeout
// double of the hearts with 4S rather than the minimum 3S. West's double
// promises four spades, so the seventh trump assures an eleven-card fit and the
// Law of Total Tricks commands game.
func TestAdvanceDoubleLawSevenCardMajor(t *testing.T) {
	// Dealer North. N 1H, E pass, S 2H, W double, N 3H, E must answer 4S.
	pbn := `[Dealer "N"]
[Deal "N:K.AJ7643.J9.Q932 QJ65432.Q8.52.T4 8.T92.QT7643.A65 AT97.K5.AK8.KJ87"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1 // seats: 0=N, 1=E, 2=S, 3=W
	// East's *second* call (its first is the opening pass) is the answer to the double.
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "4P" {
				t.Fatalf("East advance = %s (%s), want 4P (law of total tricks)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "levées totales") {
				t.Fatalf("comment %q does not mention the law of total tricks", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("East never made a second call\nauction: %s", formatAuction(calls))
}
