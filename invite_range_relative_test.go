package main

import (
	"strings"
	"testing"
)

// The 2SA invitation over a 12-14 1SA rebid asks opener to place his hand
// *inside the range already shown*: decline on 12 (the minimum of the bracket),
// accept on 13-14. The old rule required shownMin+2 = 14 to accept, so a
// 13-point opener wrongly declined and called itself "minimum".

// lastCallOf returns seat's final call and its meaning.
func lastCallOf(calls []SeatCall, seat int) (SeatCall, bool) {
	for i := len(calls) - 1; i >= 0; i-- {
		if calls[i].Seat == seat {
			return calls[i], true
		}
	}
	return SeatCall{}, false
}

// TestDecline2NTInviteOnRangeMinimum: N holds 12H (the bottom of 12-14) and
// must pass South's 2SA invitation. Deal from the reported session; the par
// table confirms only seven notrump tricks for N/S, so 3SA would fail.
func TestDecline2NTInviteOnRangeMinimum(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:KJ3.K2.KQ73.8752 T985.Q754.A84.64 A762.A63.JT65.QJ Q4.JT98.92.AKT93"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	sc, ok := lastCallOf(calls, 0)
	if !ok || sc.Call.Kind != KindPass {
		t.Fatalf("North's last call = %v, want Passe (12H, minimum of 12-14)\nauction: %s",
			sc.Call.Format("fr"), formatAuction(calls))
	}
	if !strings.Contains(sc.M.fr, "minimum") {
		t.Fatalf("decline meaning %q should say minimum", sc.M.fr)
	}
}

// TestAccept2NTInviteOnRangeMaximum: same deal with the spade queen moved to
// North (13H, upper half of 12-14): North must now accept with 3SA.
func TestAccept2NTInviteOnRangeMaximum(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:KQ3.K2.KQ73.8752 T985.Q754.A84.64 A762.A63.JT65.QJ J4.JT98.92.AKT93"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	// North's third call (after 1D and 1SA) is the answer to the invitation.
	n := 0
	for _, sc := range calls {
		if sc.Seat != 0 {
			continue
		}
		if n == 2 {
			if got := sc.Call.Format("fr"); got != "3SA" {
				t.Fatalf("North answer = %s (%s), want 3SA (13H, maximum of 12-14)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "maximum") {
				t.Fatalf("accept meaning %q should say maximum", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("North never answered the invitation\nauction: %s", formatAuction(calls))
}
