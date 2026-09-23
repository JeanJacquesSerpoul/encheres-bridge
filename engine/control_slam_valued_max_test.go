package main

import "testing"

// TestNoControlsOnSlamValuedMaximum: "la demande de contrôle doit s'appliquer
// uniquement s'il y a un espoir de chelem" [S-1]. West holds SAK97 H3
// DAKQT962 C9 -- 23 HLD once East's 2S raise agrees spades -- and the raise's
// own ceiling is 10, so the raw combined maximum lands on exactly 33 and used
// to open a cue exchange. The maximum has to be counted in slam value: West's
// singleton heart sits under the very suit East bid first, buys no trick, and
// the real ceiling is 31. The auction belongs in a plain 4S.
func TestNoControlsOnSlamValuedMaximum(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "N"]
[Vulnerable "EW"]
[Deal "N:Q63.Q87.7543.AK6 J854.K9642..JT53 T2.AJT5.J8.Q8742 AK97.3.AKQT962.9"]
`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.M.controlBid {
			t.Fatalf("started a cue exchange with no slam in view: %s %s (%s)\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4P" {
		t.Fatalf("final contract = %s, want 4P\nauction: %s", got, formatAuction(calls))
	}
}
