package main

import (
	"strings"
	"testing"
)

// TestRaiseAgreesTrumpBeforeTheAsk: South opens 1H and rebids 2NT, North
// holds 23 honour points with three-card heart support. Keycard Blackwood
// needs an agreed trump [S-4b], and North -- the short hand -- has no
// five-card suit of his own to name: the raise is what puts the trump on the
// table. Without it the slam hand had nothing left but the game to bid.
func TestRaiseAgreesTrumpBeforeTheAsk(t *testing.T) {
	pbn := `[Dealer "E"]
[Deal "E:J87.95.QT654.983 T2.AKQ86.982.QJ5 Q9654.T73.7.T762 AK3.J42.AKJ3.AK4"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 {
		t.Fatalf("final contract = %s, want a slam on 35 combined honours\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	raised := false
	for _, sc := range calls {
		if sc.Call == bidSuit(3, Hearts) && sc.M.forcing && strings.Contains(sc.M.fr, "atout") {
			raised = true
		}
	}
	if !raised {
		t.Fatalf("North never agreed the trump below game before asking\nauction: %s",
			formatAuction(calls))
	}
}
