package engine

import "testing"

// Strong hands the PAR audit (seed 20260925, 10 000 boards) found
// misdescribed, against the engine's own rules.

// TestJumpShiftInsteadOfOneLevelNewSuit (board 2465): West opens 1C on
// 3.AKQT.AQ8.AJ643 (20 H, 21 HL) and hears 1D. The one-level new suit "denies
// a jump" [RO-19] -- partner may pass it, and did, with game cold. With 20 HL
// the second suit is shown by the jump shift, game forcing: 2H.
func TestJumpShiftInsteadOfOneLevelNewSuit(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "None"]
[Deal "N:QJ42.764.765.K92 A98.82.KT43.T875 KT765.J953.J92.Q 3.AKQT.AQ8.AJ643"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	const west = 3
	if got, comment := southsCall(calls, west, 1); got != "2C" {
		t.Fatalf("West's rebid = %s (%s), want 2C (jump shift, game forcing)\nauction: %s",
			got, comment, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract.Level < 4 && !(contract.Level == 3 && contract.Strain == SNoTrump) {
		t.Fatalf("final contract %s stays below game\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestReverseOverOneNotrump (board 299): East opens 1H on AKJ8.AKJ97.A75.7
// (20 H) and hears the 1NT response. The engine rebid 2H, "répétition par
// défaut, 12-14" [RO-7]. A higher-ranking four-card suit with 18+ HL is the
// reverse [RO-19], over 1NT as over a suit response: 2S, forcing.
func TestReverseOverOneNotrump(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "None"]
[Deal "N:QT9.853.KT82.AJT AKJ8.AKJ97.A75.7 743.T62.Q63.9852 652.Q4.J94.KQ643"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	const east = 1
	got, comment := southsCall(calls, east, 1)
	if got != "2P" {
		t.Fatalf("East's rebid over 1NT = %s (%s), want 2P (reverse, forcing)\nauction: %s",
			got, comment, formatAuction(calls))
	}
}
