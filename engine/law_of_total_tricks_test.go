package engine

import "testing"

// TestLawOfTotalTricksCompetitiveRaise replays a reported deal where South,
// facing North's 1S opening and East's 2H overcall, held only 9-10 HLD and
// stopped the auction dead at a plain 2S raise. North-South actually hold a
// nine-card spade fit (5 opposite 4): the law of total tricks (Jean-René
// Vernes -- the total tricks available are approximately the total combined
// trumps) says a nine-card fit is worth competing to the three level
// regardless of high-card points, since East's monster 7-card heart suit
// makes it likely EO can make a heart partscore or better if left
// undisturbed at the two level. South must raise to 3S, not 2S.
func TestLawOfTotalTricksCompetitiveRaise(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("A9842", "5", "K72", "AJ72"),
		east:  hand("T", "AKQ9872", "A", "984"),
		south: hand("KQJ5", "JT6", "T654", "Q6"),
		west:  hand("763", "3", "QJ983", "KT53"),
	})
	calls := NewEngine(d).Run()

	const seatSouth = south
	n := 0
	for _, sc := range calls {
		if sc.Seat != seatSouth {
			continue
		}
		n++
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "3P" {
				t.Fatalf("South's raise = %s (%s), want 3P (law of total tricks, nine-card fit)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
	}

	final, _, _ := finalContract(calls)
	if final.Format("fr") != "3P" {
		t.Fatalf("final contract = %s, want 3P\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}
