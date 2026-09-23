package engine

import (
	"strings"
	"testing"
)

// TestStrong2DExploresSlamOverAceResponse checks that opener does not sign off
// in the game-level "default" notrump rebid when the hand is far above the
// 24HL floor the 2D opening promised.
//
// North holds AKQ.AKJ.AQ54.KJ2 -- 27 HCP, three aces -- and South's 3C ace
// response locates the fourth. The cheapest notrump rebid is already 3SA, a
// game South passes, burying the extra three points. With every ace accounted
// for the opening's own tool applies (docs/bidings.md, "Le 4SA de l'ouvreur
// est un appel aux Rois"): 27 plus the ace's floor of 4 is one king short of
// the 33-honour slam zone, so North asks. South answers 5C (no king), which
// leaves the pair at 31 known honours -- short of slam -- so North stops in
// 5SA instead of being forced into a six the count never reached.
func TestStrong2DExploresSlamOverAceResponse(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AKQ.AKJ.AQ54.KJ2 J9.T643.K762.Q98 8762.Q987.93.A76 T543.52.JT8.T543"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north, south = 0, 2
	seq := []struct {
		seat int
		nth  int
		call string
		hint string
	}{
		{north, 1, "4SA", "appel aux Rois"},
		{south, 1, "5T", "0 Roi"},
		{north, 2, "5SA", "chelem"},
	}
	for _, w := range seq {
		n, found := 0, false
		for _, sc := range calls {
			if sc.Seat != w.seat {
				continue
			}
			if n == w.nth {
				if got := sc.Call.Format("fr"); got != w.call {
					t.Fatalf("seat %s call #%d = %s (%s), want %s\nauction: %s",
						seatNames[w.seat], w.nth, got, sc.M.fr, w.call, formatAuction(calls))
				}
				if !strings.Contains(sc.M.fr, w.hint) {
					t.Fatalf("seat %s call #%d comment %q does not mention %q\nauction: %s",
						seatNames[w.seat], w.nth, sc.M.fr, w.hint, formatAuction(calls))
				}
				found = true
				break
			}
			n++
		}
		if !found {
			t.Fatalf("seat %s never made call #%d\nauction: %s", seatNames[w.seat], w.nth, formatAuction(calls))
		}
	}
}

// TestStrong2DKeepsDefaultRebidWhenSlamIsOut guards the gate: the same 2D - 3C
// start, but opener holds a bare 24-count. 24 plus the ace's floor of 4 leaves
// the pair five short of the slam zone, so a single king cannot bridge it and
// the king ask would only buy the five level. 3SA stands.
func TestStrong2DKeepsDefaultRebidWhenSlamIsOut(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AK4.AKJ.AQ54.K32 J9.T653.K762.Q98 8762.Q987.93.A76 QT53.42.JT8.JT54"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "3SA" {
		t.Fatalf("final contract = %s, want 3SA (redemande par défaut)\nauction: %s",
			got, formatAuction(calls))
	}
}
