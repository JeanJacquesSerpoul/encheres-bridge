package main

import (
	"strings"
	"testing"
)

// TestTrumpAceAboveGameInvitesSlam replays a reported auction. After
// 1H-1S-4S-5C-5D, South (AJ98.74.QJ42.AK8) holds no heart control but does
// hold the trump ace. 5S used to read "plus de contrôle à montrer, je nomme la
// manche" -- wrong on both counts: 5S is not the game, and the trump ace is a
// control. It must show that ace as a slam try denying hearts, so that North
// (AKQ of hearts) can cue 6H and South bid the small slam.
func TestTrumpAceAboveGameInvitesSlam(t *testing.T) {
	pbn := `[Dealer "N"]
[Vulnerable "All"]
[Deal "N:KQ76.AKQT932.T.T 542.8.8765.Q7632 AJ98.74.QJ42.AK8 T3.J65.AK93.J954"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var bids []SeatCall
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "nomme la manche") && sc.Call.Level > 4 {
			t.Fatalf("%s %s explained as the game: %q\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		if sc.Call.IsBid() {
			bids = append(bids, sc)
		}
	}
	want := []struct{ call, hint string }{
		{"1C", ""},
		{"1P", ""},
		{"4P", ""},
		{"5T", "l'As"},
		{"5K", "singleton"},
		{"5P", "l'As d'atout, pas de contrôle à Cœur : invitation au chelem"},
		{"6C", "l'As"},
		{"6P", "petit chelem"},
	}
	if len(bids) != len(want) {
		t.Fatalf("got %d bids, want %d\nauction: %s", len(bids), len(want), formatAuction(calls))
	}
	for i, w := range want {
		if got := bids[i].Call.Format("fr"); got != w.call || !strings.Contains(bids[i].M.fr, w.hint) {
			t.Fatalf("bid #%d = %s (%q), want %s mentioning %q\nauction: %s",
				i, got, bids[i].M.fr, w.call, w.hint, formatAuction(calls))
		}
	}
}

// scaffoldFiveLevelControls sets up 1S-4S-5C-5D with South on the club
// control and North on the diamond control, the exchange past the game.
func scaffoldFiveLevelControls(south, north *Hand) *Engine {
	e := scaffoldControlEngine(south, north, 20)
	e.ps[2].ctrlShown[Clubs] = true
	e.ps[0].ctrlShown[Diamonds] = true
	e.calls = append(e.calls,
		SeatCall{Seat: 2, Call: bid(5, SClubs)},
		SeatCall{Seat: 3, Call: passCall},
		SeatCall{Seat: 0, Call: bid(5, SDiamonds)},
		SeatCall{Seat: 1, Call: passCall},
	)
	return e
}

// TestTrumpAboveGameWithoutHonourIsNotTheGame checks that a hand with neither
// a side control nor a trump honour returns to the fit at the five level
// without calling it the game or pretending to show a control.
func TestTrumpAboveGameWithoutHonourIsNotTheGame(t *testing.T) {
	e := scaffoldFiveLevelControls(hand("5432", "74", "QJ42", "AK8"), hand("KQJ7", "AQT93", "T", "T9"))
	c, mn := e.continueControlBid(e.ps[2], Spades)
	if got := c.Format("fr"); got != "5P" {
		t.Fatalf("call = %s, want 5P", got)
	}
	if mn.controlBid || strings.Contains(mn.fr, "manche") {
		t.Fatalf("5S without a trump honour must be a plain return to the fit, got %q", mn.fr)
	}
}

// TestTrumpControlAboveGamePassedWithoutMissingControl checks the other side
// of the slam try: North, who cannot control the heart suit South denied,
// passes 5S instead of bidding a slam with two quick losers.
func TestTrumpControlAboveGamePassedWithoutMissingControl(t *testing.T) {
	e := scaffoldFiveLevelControls(hand("AJ98", "74", "QJ42", "AK8"), hand("KQ76", "JT9832", "T", "T"))
	c, mn := e.continueControlBid(e.ps[2], Spades)
	if got := c.Format("fr"); got != "5P" || !mn.controlBid || mn.controlSuit != Spades {
		t.Fatalf("South: call = %s (%q), want 5P as the trump control", got, mn.fr)
	}
	e.record(2, c, mn)
	e.calls = append(e.calls, SeatCall{Seat: 3, Call: passCall})
	c, mn = e.continueControlBid(e.ps[0], Spades)
	if c.Kind != KindPass || !strings.Contains(mn.fr, "pas de contrôle à Cœur") {
		t.Fatalf("North: call = %s (%q), want a pass without the heart control", c.Format("fr"), mn.fr)
	}
}
