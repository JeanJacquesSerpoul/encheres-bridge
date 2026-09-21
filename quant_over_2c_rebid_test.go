package main

import (
	"testing"
)

// TestQuantitativeNeedsTheCountNotThePosition: East opens 2C on AKQ AK4 A52
// K643 (23H) and rebids 2NT; West (64 T32 KJ863 Q72, six honour points) can
// only say 3NT, a bid the ladder prices at 4-11 [N-6]. Reading the answer as
// a position inside that band -- "middle of 4-11, so partially accept" --
// walked the pair from 3NT to 4NT to 5NT to 6NT on twenty-nine honours.
//
// Two things now stop it. The try itself needs the pair within five points of
// the slam zone, and 23 opposite a floor of 4 is not; and the answer to a try
// is arithmetic, not a position: what decides is the count added to what the
// asker has promised.
func TestQuantitativeNeedsTheCountNotThePosition(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:J9752.Q976.94.98 64.T32.KJ863.Q72 T83.J85.QT7.AJT5 AKQ.AK4.A52.K643"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Call.IsBid() && sc.Call.Level >= 5 {
			t.Fatalf("%s bid %s (%s) on 29 combined honours\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestQuantitativeStillFiresWhenTheCountIsThere is the other half, and it is
// the deal TestAceStepTwoHeartsIsNotCappedAtSeven already guards: 25 opposite
// the same 4-11 band, but a floor of 4 puts the pair at 29 -- within reach --
// and the responder's eight honour points bring the count to 33. The try must
// still be made, and accepted.
func TestQuantitativeStillFiresWhenTheCountIsThere(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "N"]
[Deal "N:A4.AQJ.AKQ2.AJ87 QT65.652.T87.KQ9 KJ732.K43.J.T532 98.T987.96543.64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(6, SNoTrump) {
		t.Fatalf("final contract = %s, want 6SA on 33 combined honours\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
