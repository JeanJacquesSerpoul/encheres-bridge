package engine

import "testing"

// TestPreferenceBackToOpenerMajor checks that responder takes the opening
// major back rather than passing out the cheap second suit when both fits are
// the same length.
//
// North opens 1S on AT653.KT.AJ97.96 (5-2-4-2, 12 HCP), South answers the
// "poubelle" 1NT on 98.J5432.K65.KT4 (7 HCP) and North names his second suit,
// 2D (docs/bidings.md, "LE BICOLORE ECONOMIQUE" -- non-forcing, responder
// chooses the strain). Five plus two spades and four plus three diamonds are
// both seven-card holdings: the major plays no worse on North's longer trump
// suit and its partscore is worth half as much again, so South must bid 2S.
func TestPreferenceBackToOpenerMajor(t *testing.T) {
	pbn := `[Dealer "W"]
[Vulnerable "All"]
[Deal "W:K7.876.Q842.AJ73 AT653.KT.AJ97.96 QJ42.AQ9.T3.Q852 98.J5432.K65.KT4"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2 // seats: 0=N, 1=E, 2=S, 3=W
	n := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "2P" {
				t.Fatalf("South's second call = %s (%s), want 2P (préférence)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "2P" {
		t.Fatalf("final contract = %s, want 2P\nauction: %s", got, formatAuction(calls))
	}
}

// TestNoPreferenceWithLongerSecondSuit guards the other side of the rule: the
// preference is a length comparison, not a reflex. Same 1S - 1NT - 2D auction,
// but South holds 9.Q7642.K65.J842 -- one spade and three diamonds, so the
// diamond fit (4+3) is a card longer than the spade one (5+1) and 2D must be
// left alone.
func TestNoPreferenceWithLongerSecondSuit(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AT653.KT.AJ97.96 KQ2.A83.T842.K75 9.Q7642.K65.J842 J874.J95.Q3.AQT3"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "2K" {
		t.Fatalf("final contract = %s, want 2K (pas de préférence)\nauction: %s",
			got, formatAuction(calls))
	}
}
