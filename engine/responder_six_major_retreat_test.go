package main

import "testing"

// afterNTRebid returns the first call a seat makes once partner's notrump
// rebid is on the table.
func afterNTRebid(calls []SeatCall, seat int) (SeatCall, bool) {
	seen := false
	for _, sc := range calls {
		if sc.Seat == partnerOf(seat) && sc.Call.IsBid() && sc.Call.Strain == SNoTrump {
			seen = true
			continue
		}
		if seen && sc.Seat == seat {
			return sc, true
		}
	}
	return SeatCall{}, false
}

// TestResponderRetreatsToSixCardMajor: the weak responder signs off in his own
// six-card major over opener's balanced notrump rebid (audit du par, donne
// 809). South opens 1D, North answers 1S on SQJT952 with 8 HL, South rebids
// 1SA (12-14 balanced). Game and invitation are both out of reach, and the
// count -- which only ever aims at a game -- had nothing left to bid: North
// passed and the side played 1SA on a nine-card spade fit, three down. The
// balanced rebid promises a doubleton, so the sixth card certifies the fit;
// 2S is the contract, and it promises nothing.
func TestResponderRetreatsToSixCardMajor(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "EW"]
[Deal "N:QJT952.K864.9.42 A64.QT5.J43.KT97 K73.AJ9.A865.J65 8.732.KQT72.AQ83"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	sc, ok := afterNTRebid(calls, north)
	if !ok {
		t.Fatalf("North never spoke after the 1SA rebid\nauction: %s", formatAuction(calls))
	}
	if got := sc.Call.Format("fr"); got != "2P" {
		t.Fatalf("North's rebid = %s (%s), want 2P on the six-card major\nauction: %s",
			got, sc.M.fr, formatAuction(calls))
	}
	// The retreat names a length. Without a ceiling on the points, that length
	// reads as news and opener raises a hand that has none.
	if sc.M.maxPts < 0 {
		t.Fatalf("the retreat leaves the hand uncapped (%s): partner will read the six cards as extra values\nauction: %s",
			sc.M.fr, formatAuction(calls))
	}
}

// TestInvitationKeepsPriorityOverTheRetreat guards the boundary: a hand whose
// count reaches the invitation belongs in the invitation zone, where the same
// six-card major is already promoted. Retreating it would sell a game the side
// can afford. North holds SAKJ763 -- 13 H, so 27 facing the maximum of a 12-14
// rebid -- and must propose rather than sign off.
func TestInvitationKeepsPriorityOverTheRetreat(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "None"]
[Deal "S:Q5.A72.AQ964.T85 42.KJ4.T752.AKQJ AKJ763.Q65.K3.42 T98.T983.J8.9763"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	sc, ok := afterNTRebid(calls, north)
	if !ok {
		t.Fatalf("North never spoke after the 1SA rebid\nauction: %s", formatAuction(calls))
	}
	if sc.M.maxPts >= 0 && sc.M.maxPts <= 10 {
		t.Fatalf("North signs off with %s (%s) on a hand worth an invitation\nauction: %s",
			sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
	}
}
