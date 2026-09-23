package engine

import (
	"strings"
	"testing"
)

// TestReopenerAcceptsTheNotrumpInvitation: over 1D passed out to South, his
// 1H réveil (8-13 HL) drew partner's 2SA -- 13-15 with a stopper, an
// invitation [R-1]. South holds J82.AQJ65.52.K83: 11 H, the top of his own
// bracket, and 3NT makes on the heart suit plus the diamonds. He used to pass,
// the absolute count (11 + 13 = 24) falling one short of the 25 a notrump game
// asks for.
func TestReopenerAcceptsTheNotrumpInvitation(t *testing.T) {
	pbn := `[Dealer "W"]
[Vulnerable "NS"]
[Deal "W:A6.K742.J876.A65 QT7.T8.AKQ9.QJ92 K9543.93.T43.T74 J82.AQJ65.52.K83"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	var answer *SeatCall
	for i, sc := range calls {
		if sc.Seat == south && sc.Call == bid(1, SHearts) {
			answer = nil
			continue
		}
		if sc.Seat == south && i > 3 {
			answer = &calls[i]
			break
		}
	}
	if answer == nil || answer.Call != bid(3, SNoTrump) || !strings.Contains(answer.M.fr, "haut du réveil") {
		got := "rien"
		if answer != nil {
			got = answer.Call.Format("fr") + " (" + answer.M.fr + ")"
		}
		t.Fatalf("South's answer to 2SA = %s, want 3SA\nauction: %s", got, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestReopenerDeclinesTheNotrumpInvitation is the other side of the same
// rule: a réveil at the bottom of its bracket (9 HL) passes the 2SA.
func TestReopenerDeclinesTheNotrumpInvitation(t *testing.T) {
	e := &Engine{opener: -1}
	e.ps[0] = &playerState{seat: 0, hand: hand("QT7", "T8", "AKQ9", "QJ92"), shownMax: 40}
	e.ps[1] = &playerState{seat: 1, hand: hand("K9543", "93", "T43", "T74"), shownMax: 40}
	e.ps[2] = &playerState{seat: 2, hand: hand("J82", "AQJ65", "52", "863"), shownMax: 40}
	e.ps[3] = &playerState{seat: 3, hand: hand("A6", "K742", "J876", "AK5"), shownMax: 40}

	reveil := m(8, 13, "réveil par une couleur", "suit réveil").withLen(Hearts, 5).asReopen()
	invite := m(13, 15, "2SA sur le réveil", "2NT over the réveil")
	invite.reopenNTInvite = true
	e.record(3, bid(1, SDiamonds), m(12, 23, "ouverture mineure", "minor opening").withLen(Diamonds, 3))
	e.record(0, passCall, noInfo())
	e.record(1, passCall, noInfo())
	e.record(2, bid(1, SHearts), reveil)
	e.record(3, passCall, noInfo())
	e.record(0, bid(2, SNoTrump), invite)
	e.record(1, passCall, noInfo())

	if got := e.ps[2].hand.HL(); got != 9 {
		t.Fatalf("test hand is %d HL, want 9 (the bottom of the réveil bracket)", got)
	}
	c, mn := e.conclude(e.ps[2])
	if c.Kind != KindPass {
		t.Fatalf("call = %s (%q), want a pass: the réveil is minimum", c.Format("fr"), mn.fr)
	}
	if !strings.Contains(mn.fr, "minimum du réveil") {
		t.Fatalf("pass comment = %q, want it to say the réveil is minimum", mn.fr)
	}
}
