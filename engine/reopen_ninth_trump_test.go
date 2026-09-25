package engine

import "testing"

// TestReopenAdvanceNinthTrump checks that the advancer of a suit réveil counts
// a point for each trump beyond the eighth [R-1]. After 1S - Pass - Pass - 2H,
// North holds A75 A864 J953 Q8: 11 H + 1 for the doubleton + 1 for the ninth
// trump = 13 HLD, the cue-bid's zone. North cue-bids 2S and South, at the top
// of the réveil, bids game in hearts.
func TestReopenAdvanceNinthTrump(t *testing.T) {
	const pbn = `[Dealer "W"]
[Vulnerable "EW"]
[Deal "E:Q2.2.8764.KT9753 J93.KJT53.AKT.64 KT864.Q97.Q2.AJ2 A75.A864.J953.Q8"]`

	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var got []string
	for _, sc := range calls {
		got = append(got, seatNames[sc.Seat]+":"+sc.Call.Format("fr"))
	}
	want := []string{"W:1P", "N:Passe", "E:Passe", "S:2C", "W:Passe", "N:2P", "E:Passe", "S:4C"}
	if len(got) < len(want) {
		t.Fatalf("auction too short: %s", formatAuction(calls))
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("call #%d = %s, want %s\nauction: %s", i, got[i], w, formatAuction(calls))
		}
	}
}
