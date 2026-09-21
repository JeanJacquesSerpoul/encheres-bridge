package main

import (
	"strings"
	"testing"
)

// TestRebidOverOneNTUnbalanced checks opener's rebid over the 1SA "poubelle"
// response. The 2SA/3SA notrump rebids promise a genuinely balanced hand
// (docs/bidings.md, "la redemande à SA": 2SA = 17-18H régulier, 3SA = 18-19H
// régulier), so an unbalanced hand must show its shape naturally. Here West is
// 6-2-1-4 with 17H/19HL: too strong for a 13-16 simple repetition and not
// balanced, so it must make the jump repetition (3S, 17-19HL, bel unicolore),
// never a rebid described as "régulier".
func TestRebidOverOneNTUnbalanced(t *testing.T) {
	// W = AKQJ93.52.3.AK84 : 17H, 19HL, 6-2-1-4 (singleton diamond).
	pbn := `[Dealer "W"]
[Deal "W:AKQJ93.52.3.AK84 654.K874.K94.QT9 T.QT3.AT8762.J75 872.AJ96.QJ5.632"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const west = 3 // seats: 0=N, 1=E, 2=S, 3=W
	n := 0
	for _, sc := range calls {
		if sc.Seat != west {
			continue
		}
		if n == 1 { // West's rebid
			if got := sc.Call.Format("fr"); got != "3P" {
				t.Fatalf("West rebid = %s (%s), want 3P (jump repetition)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if strings.Contains(sc.M.fr, "régulier") {
				t.Fatalf("unbalanced hand described as régulier: %q\nauction: %s",
					sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("West made fewer than 2 calls\nauction: %s", formatAuction(calls))
}
