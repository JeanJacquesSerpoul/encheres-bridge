package engine

import "testing"

// TestNinthTrumpReachesGame: 1C - 1S - 2S, West holds Q87632 A65 A94 7. The
// raise promises four spades, the side holds ten, and the ninth trump is worth
// the point [E-9d] that turns the old 3S invitation -- which East, minimum,
// declined -- into 4S, cold double dummy. Deal 20261001-1002 of the par bench,
// a seed no threshold was tuned on.
func TestNinthTrumpReachesGame(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "E"]
[Vulnerable "All"]
[Deal "N:KJ.K983.K7.98652 AT54.72.JT5.AKJT 9.QJT4.Q8632.Q43 Q87632.A65.A94.7"]`)
	calls := NewEngine(d).Run()
	const east, west = 1, 3
	wantCall(t, calls, east, 2, "2P")
	wantCall(t, calls, west, 2, "4P")
}
