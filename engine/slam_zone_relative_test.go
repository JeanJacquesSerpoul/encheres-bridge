package engine

import (
	"strings"
	"testing"
)

// The slam zone is a combined count [E-9], so the conventional ladders have
// to measure a hand against the floor the opening promised, not against a
// number tuned for the 15-17 notrump. Facing 20-21 the zone opens at 13 HLD,
// facing the 22-23 rebid of a strong 2C at 11 -- and the fixed thresholds
// were closing those auctions in game with 33 points on the table.

// TestStaymanSlamZoneFacingTwoNT: North opens 2NT (20-21), South holds 13 H
// with four-card heart support (16 HLD once the fit is found). Raising to 4H
// buries a pair holding 33: the auction must explore instead.
func TestStaymanSlamZoneFacingTwoNT(t *testing.T) {
	pbn := `[Dealer "W"]
[Deal "W:8543.763.5432.T8 Q2.AKQ2.AKQ7.972 T7.84.J86.KQJ543 AKJ96.JT95.T9.A6"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 {
		t.Fatalf("final contract = %s, want a slam on 33 combined honours\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestNotrumpLadderSlamZoneFacingStrongRebid: after 2C - 2D - 2NT (22-23),
// responder's 11 honour points already put the pair at 33. The ladder used
// to close at 3NT with everything above eleven going to a quantitative 4NT;
// the slam zone reached, the ask comes first [S-8].
func TestNotrumpLadderSlamZoneFacingStrongRebid(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AK5.AK.AJ65.QJT3 9864.6532.T32.98 T2.Q98.KQ84.A762 QJ73.JT74.97.K54"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 {
		t.Fatalf("final contract = %s, want a slam on 33 combined honours\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	asked := false
	for _, sc := range calls {
		if sc.Call == bid(4, SNoTrump) && strings.Contains(sc.M.fr, "demande des As") {
			asked = true
		}
	}
	if !asked {
		t.Fatalf("no keycard ask before the slam\nauction: %s", formatAuction(calls))
	}
}
