package main

import "testing"

// TestNoControlsOnNarrowRaiseCeiling: "ne démarrer les enchères de contrôle
// que si on envisage un chelem" [S-1]. West opens 1C, East responds 1S
// (forcing), West raises to 2S -- "soutien simple, 12-16HLD", a four-point
// spread that already tells the whole story. East's own count for the fit
// (12 HCP, a singleton club, a doubleton diamond, five trumps) is 17, so the
// raw combined maximum grazes exactly 33 -- but a partner pinned to a narrow,
// codified range is not the wide-range hand (a takeout double, an overcall)
// that leniency exists for: on average the raise is well short of 33, and the
// pair has no slam to explore. The auction belongs in a plain 4S, not a
// four-round cue exchange that only relearns what the raise already said.
func TestNoControlsOnNarrowRaiseCeiling(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "E"]
[Vulnerable "NS"]
[Deal "E:J4.JT42.J653.A52 KT73.Q6.AT4.QJ87 95.A5.KQ72.T9643 AQ862.K9873.98.K"]
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
