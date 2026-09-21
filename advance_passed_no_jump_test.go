package main

import (
	"strings"
	"testing"
)

// TestPassedAdvancerDoesNotJumpRaise: South passes, West passes, North opens
// 1C, East passes -- denying the values to act -- and only then does West
// overcall 1S over South's 1H. East holds K32 A543 K965 65: three trumps and
// 11 HLD once the fit is counted, which used to produce the invitational jump
// to 3S. The jump asks partner to bid game on values East's pass has already
// denied, so a passed advancer supports at the cheapest level instead [A-5].
func TestPassedAdvancerDoesNotJumpRaise(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:4.K9876.JT42.QJ2 QJ975.JT2.AQ3.43 AT86.Q.87.AKT987 K32.A543.K965.65"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	advanced := false
	for _, sc := range calls {
		if sc.Seat != east || !sc.Call.IsBid() {
			continue
		}
		advanced = true
		if sc.Call != bid(2, SSpades) {
			t.Fatalf("East's advance = %s (%s), want 2P (cheapest raise, the hand has passed)\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		if strings.Contains(sc.M.fr, "à saut") {
			t.Fatalf("East's raise still reads as a jump: %q\nauction: %s", sc.M.fr, formatAuction(calls))
		}
		break
	}
	if !advanced {
		t.Fatalf("East never supported the overcall at all\nauction: %s", formatAuction(calls))
	}
}
