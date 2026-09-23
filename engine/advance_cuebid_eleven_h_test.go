package engine

import "testing"

// TestAdvanceAsksOnElevenUsefulH is the reported deal. East opens 1H, South
// overcalls 1S and North holds AQ73 642 75 AJ92: eleven honour points, four
// trumps, nothing wasted in their suit. Counted in HLD the hand reaches only
// 12 -- the doubleton diamond is all it gets -- so it used to jump to 3S and
// cap the side in the game zone on a guess about an overcall that covers
// 9-18 HL. Eleven useful H is the second door into [A-6]: North cue-bids 2H
// and lets South place the contract.
func TestAdvanceAsksOnElevenUsefulH(t *testing.T) {
	const north = 0
	d, err := ParsePBN([]byte(`[Dealer "E"]
[Vulnerable "None"]
[Deal "N:AQ73.642.75.AJ92 K.AKT98.KT82.T87 JT942.Q73.AQJ.KQ 865.J5.9643.6543"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	first, ok := firstBidOf(calls, north)
	if !ok {
		t.Fatalf("North made no bid\nauction: %s", formatAuction(calls))
	}
	if first.Call != bidSuit(2, Hearts) || !first.M.overcallAsk {
		t.Fatalf("North's advance = %s (%s), want 2C asking the overcall's strength\nauction: %s",
			first.Call.Format("fr"), first.M.fr, formatAuction(calls))
	}
}

// TestUsefulHDiscountsBareHonourInTheirSuit pins the discount that keeps the
// new door from swallowing the "points utiles" rule [A-5]. Both hands print
// eleven honour points and hold three trumps for partner's overcall; only the
// second one's honours are worth their pips.
func TestUsefulHDiscountsBareHonourInTheirSuit(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "None"]
[Deal "S:95.Q76.KJT98.AK2 AQJ83.K54.A72.94 T62.T98.6543.QT3 K74.AJ32.Q.J8765"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	e := NewEngine(d)
	east := e.ps[1] // K74 AJ32 Q J8765, behind South's 1D opening
	if got := east.hand.H(); got != 11 {
		t.Fatalf("East prints %d H, the test wants the eleven-point hand", got)
	}
	e.Run() // the discount reads the auction, so the suits must be named first
	if got := e.hAgainstTheirBidding(east, Spades); got >= 11 {
		t.Fatalf("useful H with a bare queen in their diamonds = %d, want under 11", got)
	}
}
