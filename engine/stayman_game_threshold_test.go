package engine

import (
	"strings"
	"testing"
)

// TestStaymanFitInvitesBelowGameThreshold: after 1NT - 2C - 2S, South holds
// AJ94.92.764.KJ75 -- 9 H, 10 HLD with the doubleton. Facing 15-17 the pair
// counts 25 to 27, and a major game needs 27 HLD [E-9]: it is there only
// opposite a maximum. South must invite with 3S, not bid 4S; North, a flat
// 15-count, declines.
func TestStaymanFitInvitesBelowGameThreshold(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "EW"]
[Deal "E:832.Q863.AJ82.98 AJ94.92.764.KJ75 65.JT75.QT3.AQ43 KQT7.AK4.K95.T62"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	const south = 2
	// South passed in first seat: his calls are Passe, 2C, then the one here.
	got, comment := southsCall(calls, south, 2)
	if got != "3P" || !strings.Contains(comment, "proposition") {
		t.Fatalf("South's call after 2P = %s (%s), want 3P (game invitation)\nauction: %s",
			got, comment, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract != bidSuit(3, Spades) {
		t.Fatalf("final contract = %s, want 3P (North minimum declines)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestStaymanFitInviteAcceptedByMaximum: the same South facing a 17-count
// (KQT7.AK4.KQ5.T62) -- 10 + 17 reaches the 27 HLD of a major game, so North
// accepts the invitation.
func TestStaymanFitInviteAcceptedByMaximum(t *testing.T) {
	d := dealWithQuietOpponents(0, map[int]*Hand{
		0: hand("KQT7", "AK4", "KQ5", "T62"),
		2: hand("AJ94", "92", "764", "KJ75"),
	})
	calls := NewEngine(d).Run()
	got, comment := southsCall(calls, 2, 1)
	if got != "3P" {
		t.Fatalf("South's call after 2P = %s (%s), want 3P (game invitation)\nauction: %s",
			got, comment, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract != bidSuit(4, Spades) {
		t.Fatalf("final contract = %s, want 4P (maximum accepts)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
