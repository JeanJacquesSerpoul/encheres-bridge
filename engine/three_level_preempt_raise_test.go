package engine

import "testing"

// Facing partner's three-level major preempt (seven cards), three trumps make
// ten: the law of total tricks puts the side at the four level, whatever the
// points [F-6b]. The raise is both the extended preempt and the game.

// TestThreeLevelPreemptRaisedOnTenTrumps: South opens 3S on AKJ9862.T96.6.J9;
// North, T54.K.A93.AQ8752 (13 H), holds three spades and must bid 4S, not
// pass.
func TestThreeLevelPreemptRaisedOnTenTrumps(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "EW"]
[Deal "S:AKJ9862.T96.6.J9 Q7.J8742.KJ7.643 T54.K.A93.AQ8752 3.AQ53.QT8542.KT"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	got, comment := southsCall(calls, 0, 0)
	if got != "4P" {
		t.Fatalf("North's answer to 3P = %s (%s), want 4P (law of total tricks)\nauction: %s",
			got, comment, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract != bidSuit(4, Spades) {
		t.Fatalf("final contract = %s, want 4P\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestThreeLevelPreemptExtendedWhenWeak: the law holds on a weak hand too --
// the raise then extends the preempt.
func TestThreeLevelPreemptExtendedWhenWeak(t *testing.T) {
	d := dealWithQuietOpponents(2, map[int]*Hand{
		2: hand("AKJ9862", "T96", "6", "J9"),
		0: hand("Q54", "8765", "J93", "872"),
	})
	calls := NewEngine(d).Run()
	if got, comment := southsCall(calls, 0, 0); got != "4P" {
		t.Fatalf("North's answer to 3P = %s (%s), want 4P (extending the preempt)\nauction: %s",
			got, comment, formatAuction(calls))
	}
}

// TestThreeLevelPreemptPassedWithTwoTrumps: two trumps and less than 16 H
// leave nine trumps and no game: North still passes.
func TestThreeLevelPreemptPassedWithTwoTrumps(t *testing.T) {
	d := dealWithQuietOpponents(2, map[int]*Hand{
		2: hand("AKJ9862", "T96", "6", "J9"),
		0: hand("54", "K876", "A93", "Q872"),
	})
	calls := NewEngine(d).Run()
	if got, comment := southsCall(calls, 0, 0); got != "Passe" {
		t.Fatalf("North's answer to 3P = %s (%s), want Passe\nauction: %s",
			got, comment, formatAuction(calls))
	}
}
