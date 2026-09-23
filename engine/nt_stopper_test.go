package engine

import (
	"strings"
	"testing"
)

// TestNoNotrumpWithoutStopperInOverdcalledSuit reproduces a deal where South,
// facing East's 1S overcall over North's 1D opening, has a complete void in
// spades. The old logic picked a blind "proposition de manche" 2NT in the
// invitation zone regardless of opponents' suits; bidding notrump with no
// control at all in the overcalled suit is a system error, since the
// opponents can simply run spades against 2NT/3NT. South names his five-card
// heart suit instead -- 2H, natural and forcing [RC-1b] -- which is also what
// finds the 5-3 heart fit the diamond raise used to bury.
func TestNoNotrumpWithoutStopperInOverdcalledSuit(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("Q3", "T94", "KJ92", "AQ63"),
		east:  hand("AT864", "K", "Q643", "752"),
		south: hand("", "A8632", "AT75", "KT98"),
		west:  hand("KJ9752", "QJ75", "8", "J4"),
	})

	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if sc.Call == bid(2, SNoTrump) {
			t.Fatalf("South bid 2SA with a spade void (no stopper in East's overcalled suit)\nauction: %s", formatAuction(calls))
		}
	}

	found := false
	for _, sc := range calls {
		if sc.Seat == south && sc.Call.IsBid() {
			if sc.Call != bid(2, SHearts) || !strings.Contains(sc.M.fr, "forcing") {
				t.Fatalf("South's first bid = %s (%q), want 2C (2H, natural and forcing)\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("South never bid his heart suit\nauction: %s", formatAuction(calls))
	}
}
