package engine

import (
	"testing"
)

// TestNoBlackwoodWithUncontrolledSideSuit reproduces a reported deal where the
// point count reached the slam zone (North 20 HLD support, South ~13) but the
// partnership held two fast losers in an unbid, uncontrolled side suit
// (hearts: North T4, South QJ965 -- neither the ace nor the king). Blackwood
// counts keycards, not second-round control, so it saw only the missing heart
// ACE (one keycard) and blasted 6S off the top two heart tricks. With no side
// control shown and no cue-bidding room under the 4S jump, the auction must
// stay in game rather than drive to a doomed small slam.
func TestNoBlackwoodWithUncontrolledSideSuit(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("K843", "T4", "AKQJ2", "A4"),
		east:  hand("52", "K2", "T8", "QT97652"),
		south: hand("AJT97", "QJ965", "543", ""),
		west:  hand("Q6", "A873", "976", "KJ83"),
	})

	calls := NewEngine(d).Run()

	// No 4NT Blackwood must appear anywhere in the auction.
	for _, sc := range calls {
		if sc.Call == bid(4, SNoTrump) {
			t.Fatalf("engine launched Blackwood despite an uncontrolled heart suit\nauction: %s",
				formatAuction(calls))
		}
	}

	contract, decl, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P (game, not slam)\nauction: %s", got, formatAuction(calls))
	}
	if seatNames[decl] != "S" {
		t.Fatalf("declarer = %s, want S\nauction: %s", seatNames[decl], formatAuction(calls))
	}
}
