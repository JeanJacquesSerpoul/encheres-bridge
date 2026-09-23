package engine

import (
	"testing"
)

// TestNoKeycardAskBelowTheSlamZoneInAMajor: with a major trump, 4NT and every
// answer to it sit above 4M. A keycard check that comes back short therefore
// strands the pair in five of the major -- eleven tricks for a game bonus --
// and nothing can bring the auction back down. Below 33 combined the answer
// cannot even change the decision: four keycards need 33 to bid the slam
// [S-7], so only "all five" would do.
//
// East opens 2C on AKQ985 A A52 K64 and rebids 2S; West (64 KT32 K863 Q72)
// names 3NT, East starts cue-bidding and the exchange used to end 4C-4D-4H-
// 4NT-5D-5S. It now signs off in 4S, where the controls were heading.
func TestNoKeycardAskBelowTheSlamZoneInAMajor(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:J2.Q9754.94.A983 64.KT32.K863.Q72 T73.J86.QJT7.JT5 AKQ985.A.A52.K64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.M.blackwood {
			t.Fatalf("%s asked for keycards below the slam zone, buying the five level\nauction: %s",
				seatNames[sc.Seat], formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if contract != bid(4, SSpades) {
		t.Fatalf("final contract = %s, want 4P\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestKeycardAskStaysInTheSlamZone is the other half: the same machinery must
// still ask once the count is there. West holds 742 KQ32 K863 J7 -- the fit
// and nine honour points -- which carries the pair into the slam zone, so the
// five level is paid for and the ask is affordable.
func TestKeycardAskStaysInTheSlamZone(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:J3.JT98.QJT.T983 742.KQ32.K863.J7 T6.7654.974.AQ52 AKQ985.A.A52.K64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	asked := false
	for _, sc := range calls {
		if sc.M.blackwood {
			asked = true
		}
	}
	if !asked {
		t.Fatalf("nobody asked for keycards inside the slam zone\nauction: %s", formatAuction(calls))
	}
}
