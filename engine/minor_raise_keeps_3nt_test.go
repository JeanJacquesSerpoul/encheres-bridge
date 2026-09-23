package engine

import "testing"

// TestMinorRaiseKeeps3NT replays a deal where opener holds four-card support
// for responder's two-over-one minor with 17-19 HLD. The old jump raise to
// four of the minor bypassed 3NT and drove the pair into 5D (eleven tricks)
// when 3NT (nine tricks) was the making game. Since the two-over-one already
// forces to game, opener must raise below 3NT — the jump past 3NT is reserved
// for near-certain slam hands (22+ HLD).
//
// South: QT763 / T2 / AQT9 / AK (15H, 17-19 HLD with the diamond fit)
// North: 98 / KQ4 / K8732 / Q86 (10H, 11 HL — a minimum two-over-one)
// Par: 3NT by N/S; 5D is one level too high in value.
func TestMinorRaiseKeeps3NT(t *testing.T) {
	const north, south = 0, 2
	d := dealWith(south, map[int]*Hand{
		south: hand("QT763", "T2", "AQT9", "AK"),
		north: hand("98", "KQ4", "K8732", "Q86"),
	})
	calls := NewEngine(d).Run()

	n := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		n++
		switch n {
		case 1:
			if got := sc.Call.Format("fr"); got != "1P" {
				t.Fatalf("South's opening = %s, want 1P\nauction: %s", got, formatAuction(calls))
			}
		case 2:
			if got := sc.Call.Format("fr"); got != "3K" {
				t.Fatalf("South's rebid = %s (%s), want 3K (raise below 3SA)\nauction: %s", got, sc.M.fr, formatAuction(calls))
			}
		}
	}

	final, _, _ := finalContract(calls)
	if final.Format("fr") != "3SA" {
		t.Fatalf("final contract = %v, want 3SA\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}
