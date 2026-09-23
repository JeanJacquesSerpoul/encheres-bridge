package engine

import (
	"strings"
	"testing"
)

// TestNTRebidInCompetition checks opener's balanced-minimum notrump rebid when a
// two-level overcall has taken the 1NT rebid away. Sequence: N 1D, E pass,
// S 1S (forcing), W 2C, N ? — North (AT5.K73.QT742.K9, balanced 12H with a club
// stopper) must rebid 2NT, and the meaning must say 2SA (not 1SA) and promise a
// stopper in the enemy suit. South (14H, no club stopper) then trusts that
// promised stopper and bids 3NT, the double-dummy par contract.
func TestNTRebidInCompetition(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AT5.K73.QT742.K9 96.JT654.653.Q75 J874.AQ8.AKJ8.64 KQ32.92.9.AJT832"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var northRebid, southRebid *SeatCall
	const north, south = 0, 2
	nSeen := map[int]int{}
	for i := range calls {
		sc := &calls[i]
		switch sc.Seat {
		case north:
			if nSeen[north] == 1 { // opening is the first call
				northRebid = sc
			}
			nSeen[north]++
		case south:
			if nSeen[south] == 1 { // 1S response is the first call
				southRebid = sc
			}
			nSeen[south]++
		}
	}

	if northRebid == nil || northRebid.Call.Format("fr") != "2SA" {
		t.Fatalf("North rebid = %v, want 2SA\nauction: %s", northRebid, formatAuction(calls))
	}
	if strings.Contains(northRebid.M.fr, "1SA") {
		t.Fatalf("North 2SA meaning still says 1SA: %q", northRebid.M.fr)
	}
	if !strings.Contains(northRebid.M.fr, "arrêt") {
		t.Fatalf("North 2SA meaning %q does not promise a stopper", northRebid.M.fr)
	}
	if southRebid == nil || southRebid.Call.Format("fr") != "3SA" {
		t.Fatalf("South rebid = %v, want 3SA (game trusting the promised stopper)\nauction: %s",
			southRebid, formatAuction(calls))
	}
}
