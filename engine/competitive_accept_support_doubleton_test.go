package engine

import (
	"strings"
	"testing"
)

// TestSupportingHandCountsItsDoubleton is a regression test for a reported
// deal: 1C – (1S) – 2C – (2S) – 3C – (3S), and East declined the invitation
// "minimum" holding KT9 AT854 J83 T2 opposite a five-card-plus overcall.
//
// He is not minimum. The contested acceptance compares partner's floor plus
// his own value against the 27 a major game needs — an HLD figure — while the
// value offered was HL plus real shortness, that is HLD minus the doubleton.
// East's club doubleton sits in the hand that will be dummy, where a spare
// trump turns it into a ruff: 17 + 10 reaches 27 and the vulnerable game is
// there. The old count stopped at 26 and passed [G-1].
//
// TestOpenerDeclinesCompetitiveRaiseGame guards the other side of the rule:
// the same doubleton in the long-trump hand buys nothing and must not count.
func TestSupportingHandCountsItsDoubleton(t *testing.T) {
	const pbn = `[Dealer "S"]
[Vulnerable "EW"]
[Deal "S:Q3.32.AKQT.QJ865 AJ87654.Q976..A3 2.KJ.976542.K974 KT9.AT854.J83.T2"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "4P" {
				t.Fatalf("East's answer to the invitation = %s (%s), want 4P\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "accepte") {
				t.Fatalf("comment %q does not accept\nauction: %s", sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("East never answered the invitation\nauction: %s", formatAuction(calls))
}
