package main

import (
	"strings"
	"testing"
)

// TestKeycardHonourCeiling checks the arithmetic that can rule out the low
// reading of an ambiguous keycard step [S-6]: with the missing keycards in
// the opponents' hands, only so many points are left for partner.
func TestKeycardHonourCeiling(t *testing.T) {
	// Trump hearts, the hand holds the diamond ace and the heart king: three
	// aces (12 points) are missing. A partner with no keycard at all can hold
	// at most 40 - 16 - 12 = 12 honour points.
	h := hand("7", "KQT986", "AK8", "KJ3")
	if got := keycardHonourCeiling(h, Hearts, 0); got != 12 {
		t.Fatalf("ceiling with no keycard = %d, want 12", got)
	}
	// Given three of them, partner keeps the three aces: nothing is left with
	// the opponents, so the ceiling is simply what this hand does not hold.
	if got := keycardHonourCeiling(h, Hearts, 3); got != 24 {
		t.Fatalf("ceiling with three keycards = %d, want 24", got)
	}
}

// TestAmbiguousStepReadHighOnPartnersFloor: South jump-raises to 3H showing
// 17-19 HLD, North asks keycards and hears the ambiguous "0 or 3" step.
// North holds 16 honour points and two keycards, so a partner with none
// could not hold more than twelve points -- five short of the floor his jump
// raise promised. The step therefore reads three, the pair holds all five
// keycards, and the auction belongs in a slam rather than at the five level.
func TestAmbiguousStepReadHighOnPartnersFloor(t *testing.T) {
	pbn := `[Dealer "E"]
[Deal "E:J9654.72.Q4.8742 AKQ.AJ53.JT7.A65 T832.4.96532.QT9 7.KQT986.AK8.KJ3"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 {
		t.Fatalf("final contract = %s, want a slam with every keycard held\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	stopped := false
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "arrêt à 5") {
			stopped = true
		}
	}
	if stopped {
		t.Fatalf("the ask signed off at the five level holding all five keycards\nauction: %s",
			formatAuction(calls))
	}
}
