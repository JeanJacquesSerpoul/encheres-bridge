package engine

import "testing"

// TestLongSuitRebidPushedToThreeLevel is a regression test for a reported
// deal: North opened 1D on KQJT732 and the club ace, South answered a forcing
// 1S, West overcalled 2H — and North passed. Every description [RO-18] to
// [RO-22] had been taken away (no heart stopper for the notrump rebids, no
// second suit), and the seven-card suit went unsaid. South, hearing nothing,
// invited with 2NT and North passed that too: the pair played 2NT on a hand
// where the diamonds run.
//
// The suit has to be shown. 3D is the only call left [RO-21b], and South's
// heart stopper then turns it into 3NT.
func TestLongSuitRebidPushedToThreeLevel(t *testing.T) {
	const pbn = `[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:T.76.KQJT732.A73 8752.983.6.KT642 J964.AK2.985.QJ9 AKQ3.QJT54.A4.85"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "3K" {
				t.Fatalf("North's rebid over 2H = %s (%s), want 3K\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "3SA" {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", got, formatAuction(calls))
	}
}
