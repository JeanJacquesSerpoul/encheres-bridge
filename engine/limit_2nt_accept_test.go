package engine

import "testing"

// wantCall checks a seat's n-th call (1-based) against its French spelling.
func wantCall(t *testing.T, calls []SeatCall, seat, n int, want string) {
	t.Helper()
	sc, ok := nthCallOf(calls, seat, n)
	if !ok {
		t.Fatalf("%s never made call #%d\nauction: %s", seatNames[seat], n, formatAuction(calls))
	}
	if got := sc.Call.Format("fr"); got != want {
		t.Fatalf("%s call #%d = %s (%s), want %s\nauction: %s",
			seatNames[seat], n, got, sc.M.fr, want, formatAuction(calls))
	}
}

// TestAcceptPartnersTwoNTLimit: 1C - 1S - 2NT (18-19) is itself the game
// invitation, so the responder cannot invite by bidding 2NT again -- the old
// decision found that proposal "unavailable" and passed. With six points the
// middle of the 18-19 range reaches 25 [E-9b]: North bids 3NT.
func TestAcceptPartnersTwoNTLimit(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "W"]
[Deal "N:8753.J8.KT7.Q986 Q94.7542.QJ6.K52 J2.AKQT.A83.AJ73 AKT6.963.9542.T4"]`)
	calls := NewEngine(d).Run()
	const north, south = 0, 2
	wantCall(t, calls, south, 2, "2SA")
	wantCall(t, calls, north, 3, "3SA")
}

// TestSixCardMajorOverTwoNTLimit: 1H - 1S - 2NT (18-19) facing a six-card
// spade suit and five points. Opener's balanced 2NT holds two spades at
// least, the fit is eight cards, and the sixth trump is a trick: South bids
// 4S [RO-16b] rather than passing 2NT (or choosing 3NT).
func TestSixCardMajorOverTwoNTLimit(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "W"]
[Deal "N:AK6.JT753.K7.AKJ J5.A2.Q95432.Q63 Q98742.K8.JT86.5 T3.Q964.A.T98742"]`)
	calls := NewEngine(d).Run()
	const north, south = 0, 2
	wantCall(t, calls, north, 2, "2SA")
	wantCall(t, calls, south, 2, "4P")
}
