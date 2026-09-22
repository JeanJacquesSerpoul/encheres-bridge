package main

import (
	"strings"
	"testing"
)

// TestSuperAcceptStartsControls replays a reported auction: 1NT-2D-3H, the
// super-accept (four trumps, maximum), and West KT5.KJ943.9.AQJ6 (17 HLD)
// signed off in 4H -- 34 combined with every keycard in the pair. With a slam
// in view the super-accept is where controls start [N-8]: West cues his
// cheapest control, the spade king at 3S [S-2b], and the exchange reaches the
// heart slam.
func TestSuperAcceptStartsControls(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "None"]
[Deal "N:32.75.QJT762.T42 A98.AQ82.A5.K953 QJ764.T6.K843.87 KT5.KJ943.9.AQJ6"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var bids []SeatCall
	for _, sc := range calls {
		if sc.Call.IsBid() {
			bids = append(bids, sc)
		}
	}
	want := []struct{ call, hint string }{
		{"1SA", "15-17"},
		{"2K", "Texas"},
		{"3C", "rectification à saut"},
		{"3P", "enchère de contrôle"},
	}
	if len(bids) < len(want) {
		t.Fatalf("auction too short\nauction: %s", formatAuction(calls))
	}
	for i, w := range want {
		if got := bids[i].Call.Format("fr"); got != w.call || !strings.Contains(bids[i].M.fr, w.hint) {
			t.Fatalf("bid #%d = %s (%q), want %s mentioning %q\nauction: %s",
				i, got, bids[i].M.fr, w.call, w.hint, formatAuction(calls))
		}
	}

	contract, _, _ := finalContract(calls)
	if contract.Level < 6 || contract.Strain != SHearts {
		t.Fatalf("final contract = %s, want a heart slam\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
