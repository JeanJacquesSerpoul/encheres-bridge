package engine

import (
	"strings"
	"testing"
)

// TestGameTryNotBlastOverSimpleRaise reproduces the deal where West
// (T8.AKQ65.AK964.4, 16H / 21 HLD) blasted 4H over East's simple raise
// (6-10 HLD). East may hold a bare 6, so 21 + 6 = 27 is only the bare game
// threshold: West must make a game try and let partner decide — direct game
// facing the raise needs 22+ HLD (22 + 6 = 28). On this layout East (10H,
// spade ace) accepts and the making 4H is still reached, via the try.
func TestGameTryNotBlastOverSimpleRaise(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:K62.8.QJ832.9852 AJ5.J942.T75.KJ3 Q9743.T73.-.AQT76 T8.AKQ65.AK964.4"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	// West's second call (after the 1H opening) must be a game try, not 4H.
	const west = 3
	n := 0
	for _, sc := range calls {
		if sc.Seat != west {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got == "4C" || !strings.Contains(sc.M.fr, "essai") {
				t.Fatalf("West rebid = %s (%s), want a game try (partner may hold just 6)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			break
		}
		n++
	}

	// East holds 10H with the spade ace: the try is accepted and 4H reached.
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4C" {
		t.Fatalf("final contract = %s, want 4C (try accepted on a maximum)\nauction: %s",
			got, formatAuction(calls))
	}
}
