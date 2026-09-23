package engine

import "testing"

// TestLandyAdvanceStopsAtWhatTheFitPays: choosing the longer major over
// partner's Landy is an obligation of the convention, not a licence to buy
// whatever level the auction has reached (audit du par, donne 798). East
// opens 1NT, South overcalls 2C Landy -- at least 5-4 in the majors -- West
// jumps to 3NT, and North holds S8 H87 DJT7654 CK965: a doubleton in his
// longer major and 6 HL. The advance answered 4H, "law of total tricks", on a
// fit the law counts as six cards. Doubled, it was eight down.
func TestLandyAdvanceStopsAtWhatTheFitPays(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "None"]
[Deal "E:AT72.KQ64.K32.A3 KQ96543.J952..Q4 J.AT3.AQ98.JT872 8.87.JT7654.K965"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	for _, sc := range calls {
		if sc.Seat != north || !sc.Call.IsBid() {
			continue
		}
		t.Fatalf("North advances the Landy with %s (%s) on a doubleton in the major\nauction: %s",
			sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
	}
}

// TestLandyAdvanceKeepsTheAffordableLevel guards the other side of the
// ceiling: the advance still buys the level its own fit and zone pay for, so
// the correction does not turn into blanket prudence. East opens 1NT, South
// overcalls 2C Landy, West competes with 2S -- and North holds four hearts
// facing the four the Landy promises, with the values to carry the level.
// The convention's answer is 3H, not a pass.
func TestLandyAdvanceKeepsTheAffordableLevel(t *testing.T) {
	e := &Engine{}
	p := &playerState{seat: 0, hand: hand("54", "KQ96", "A8732", "62"), shownMax: 40}
	e.ps = [4]*playerState{p,
		{seat: 1, hand: hand("", "", "", ""), shownMax: 40},
		{seat: 2, hand: hand("", "", "", ""), shownMax: 40},
		{seat: 3, hand: hand("", "", "", ""), shownMax: 40}}
	e.calls = []SeatCall{
		{Seat: 1, Call: bid(1, SNoTrump)},
		{Seat: 2, Call: bid(2, SClubs), M: m(10, 18, "", "").asLandy()},
		{Seat: 3, Call: bid(2, SSpades)},
	}
	c, mn := e.advanceLandy(p)
	if !c.IsBid() {
		t.Fatalf("advance passed (%s), want the major at the level the fit pays for", mn.fr)
	}
	if got := c.Format("fr"); got != "3C" {
		t.Fatalf("advance = %s (%s), want 3C", got, mn.fr)
	}
}
