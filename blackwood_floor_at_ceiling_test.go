package main

import (
	"strings"
	"testing"
)

// TestKeycardFloorAtTheCeilingReadsHigh: East opens 1H on K9.AKJT9862.4.KQ
// (16 H, two keycards), jump-rebids 3H, and West's 3NT promises 12. A
// zero-keycard West could hold at most 40-16-12 = 12, exactly that floor, so
// the "zero or three" answer is three and the five keycards are there. The
// engine used to keep the low reading at the boundary and sign off in 5H --
// asking for keycards and then ignoring the answer.
func TestKeycardFloorAtTheCeilingReadsHigh(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "NS"]
[Deal "E:K9.AKJT9862.4.KQ JT87.43.Q753.J94 A5432.7.AJ96.A63 Q6.Q5.KT82.T8752"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1
	var asked, answer bool
	for _, sc := range calls {
		if sc.Seat == east && sc.M.blackwood {
			asked = true
		}
		if sc.Seat == east && asked && sc.Call.Level >= 6 {
			answer = true
		}
	}
	if !asked {
		t.Fatalf("East never asked for keycards\nauction: %s", formatAuction(calls))
	}
	if !answer {
		t.Fatalf("East asked for keycards and then stopped short of slam\nauction: %s", formatAuction(calls))
	}

	contract, _, _ := finalContract(calls)
	if contract != bid(6, SHearts) {
		t.Fatalf("final contract = %s, want 6C (6H)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	for _, sc := range calls {
		if sc.Seat == east && sc.Call == bid(6, SHearts) && !strings.Contains(sc.M.fr, "chelem") {
			t.Fatalf("6H comment %q should name the slam", sc.M.fr)
		}
	}
}
