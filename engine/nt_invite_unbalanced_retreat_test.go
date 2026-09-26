package engine

import "testing"

// TestNTInviteUnbalancedRetreat: 1D - 1S - 2C - 2NT, opener holds
// — K84 AT7654 AQT7, a minimum with a spade void. Passing leaves the side in a
// notrump partscore that partner's own suit cannot reach; the diamonds are the
// strain. Opener declines by going back to 3D [RO-19d].
func TestNTInviteUnbalancedRetreat(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "S"]
[Deal "N:Q7654.A5.Q2.KJ53 AJ92.QT763.J.986 .K84.AT7654.AQT7 KT83.J92.K983.42"]`)
	calls := NewEngine(d).Run()
	const north, south = 0, 2
	wantCall(t, calls, north, 2, "2SA")
	wantCall(t, calls, south, 3, "3K")
}
