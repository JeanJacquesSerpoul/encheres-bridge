package engine

import (
	"strings"
	"testing"
)

// TestOpenerBalancedRebidNeedsStopperAfter2on1 checks that opener's balanced
// minimum rebid (12-14H) after partner's forcing 2/1 new-suit response in
// competition requires a stopper in the intervention suit, exactly as the
// same rebid already does after a 1-level response. Without a stopper the
// bid must not be made at all -- neither at its natural level nor bumped up
// by further competitive bidding -- rather than guessing notrump unguarded.
//
// West = A K T 4 . 8 6 3 . K 7 3 2 . K 8 opens 1D; North overcalls 1H (5+
// hearts); East, forcing with a two-over-one response, bids 2C; West holds
// no heart stopper (863) and must not bid notrump on the next round.
func TestOpenerBalancedRebidNeedsStopperAfter2on1(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3

	northH := hand("J98", "AQJT9", "QJ", "765")
	eastH := hand("Q6", "742", "98", "AQJ432")

	t.Run("sans arrêt à Cœur : pas de Sans-Atout", func(t *testing.T) {
		westH := hand("AKT4", "863", "K732", "K8")
		d := dealWith(east, map[int]*Hand{west: westH, north: northH, east: eastH})
		calls := NewEngine(d).Run()

		for _, sc := range calls {
			if sc.Seat == west && sc.Call.IsBid() && sc.Call.Strain == SNoTrump {
				t.Fatalf("West bid notrump (%s, %q) with no heart stopper\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
		}
	})

	t.Run("avec l'arrêt à Cœur : Sans-Atout normal", func(t *testing.T) {
		westH := hand("AKT4", "K63", "7632", "K8")
		d := dealWith(east, map[int]*Hand{west: westH, north: northH, east: eastH})
		calls := NewEngine(d).Run()

		found := false
		for _, sc := range calls {
			if sc.Seat == west && sc.Call == bid(2, SNoTrump) {
				found = true
				if !strings.Contains(sc.M.fr, "12-14") {
					t.Fatalf("comment %q does not mention 12-14H", sc.M.fr)
				}
			}
		}
		if !found {
			t.Fatalf("West never rebid 2SA with a heart stopper\nauction: %s", formatAuction(calls))
		}
	})
}
