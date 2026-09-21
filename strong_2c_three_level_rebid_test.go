package main

import (
	"strings"
	"testing"
)

// TestStrongTwoClubsThreeLevelRebidFindsGame: East opens 2C on A K64 A52
// AKQ985 -- 20H and a six-card club suit worth nine tricks on its own [O-2]
// -- West relays 2D and East rebids 3C, forcing. West (KT32 Q72 K863 64) has
// no fit and no suit of his own, and the waiting 2NT of [F-1] is no longer
// legal at that height. The engine used to fall through to the safety net and
// raise the clubs to 4C on a doubleton, stopping the pair below every game on
// 28 combined honours. There is nothing left to wait for: 3NT is the game
// that needs the fewest tricks, and it is the one to name.
func TestStrongTwoClubsThreeLevelRebidFindsGame(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:Q9754.A983.94.J2 KT32.Q72.K863.64 J86.JT5.QJT7.T73 A.K64.A52.AKQ985"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	// West's calls: the dealer-round pass, the 2D relay, then the answer.
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never answered the 3C rebid\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(3, SNoTrump) {
		t.Fatalf("West's answer = %s (%s), want 3SA\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	if !strings.Contains(answer.M.fr, "manche") {
		t.Fatalf("comment %q does not read as placing the game", answer.M.fr)
	}
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestStrongTwoClubsThreeLevelDiamondRebid: the same sequence one suit over.
// East's six-card suit is diamonds, so the rebid is 3D -- the very step the
// relay itself used -- and West, again without a fit or a suit of his own,
// names 3NT. The rule keys off the height of the rebid, not the suit.
func TestStrongTwoClubsThreeLevelDiamondRebid(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:Q9754.A983.J2.94 KT32.Q72.64.K863 J86.JT5.T73.QJT7 A.K64.AKQ985.A52"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never answered the 3K rebid\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(3, SNoTrump) {
		t.Fatalf("West's answer = %s (%s), want 3SA\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
}

// TestStrongTwoClubsThreeLevelFittedRaise: with the fit and 8H, [F-1] calls
// for "three of the suit, slam interest" -- but the opener has just used the
// three level himself, so the raise is four of it. The bid must come from the
// rule, not from the illegal-bid net: the net only finds the level when it
// judges the hand worth a level more, and a rule that exists only when a
// safety net happens to agree is not a rule.
func TestStrongTwoClubsThreeLevelFittedRaise(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:T9754.A983.T4.J9 KQ32.Q72.J763.86 J86.JT5.2.KQT743 A.K64.AKQ985.A52"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never raised\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(4, SDiamonds) {
		t.Fatalf("West's raise = %s (%s), want 4K\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	if !strings.Contains(answer.M.fr, "espoir de chelem") {
		t.Fatalf("comment %q does not read as the slam-interest raise", answer.M.fr)
	}
	if contract, _, _ := finalContract(calls); contract != bid(5, SDiamonds) {
		t.Fatalf("final contract = %s, want 5K\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestStrongTwoClubsMinorRaiseZoneIsSplit: in a major, the whole 0-7 zone can
// land on the same raise because that raise is the game. In a minor the game
// is a level higher, so one bid for seven points left the opener guessing over
// exactly the six he needs for the eleventh trick. The zone is split on honour
// points -- what he is missing is two tricks, and length in a side suit facing
// a one-suited hand rarely supplies one.
//
// West holds KT32 Q72 J763 86: the fit and six honour points, so he names the
// game the opener's nine tricks and his own two are worth.
func TestStrongTwoClubsMinorRaiseZoneIsSplit(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:Q9754.A983.T4.J9 KT32.Q72.J763.86 J86.JT5.2.KQT743 A.K64.AKQ985.A52"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never raised\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(5, SDiamonds) {
		t.Fatalf("West's raise = %s (%s), want 5K (the fit and five honour points)\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract != bid(5, SDiamonds) {
		t.Fatalf("final contract = %s, want 5K\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestStrongTwoClubsMinorRaiseStaysLowOnABust is the other half of that split:
// the same fit with three honour points is worth the cheapest raise and no
// more. The opener, told 0-4 rather than 0-7, has a zone he can actually
// judge -- and passes.
func TestStrongTwoClubsMinorRaiseStaysLowOnABust(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:QJ9754.A983.T.J9 T32.Q72.J763.864 K86.JT5.42.KQT73 A.K64.AKQ985.A52"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never raised\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(4, SDiamonds) {
		t.Fatalf("West's raise = %s (%s), want 4K\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	if answer.M.maxPts != 4 {
		t.Fatalf("the raise announces 0-%d, want the split zone 0-4\nauction: %s",
			answer.M.maxPts, formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract != bid(4, SDiamonds) {
		t.Fatalf("final contract = %s, want 4K\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// seatNthCall returns the nth (0-indexed) call made by a seat.
func seatNthCall(calls []SeatCall, seat, nth int) (SeatCall, bool) {
	n := 0
	for _, sc := range calls {
		if sc.Seat != seat {
			continue
		}
		if n == nth {
			return sc, true
		}
		n++
	}
	return SeatCall{}, false
}

// TestStrongTwoClubsWaitingBidDoesNotCapAGoodHand: the waiting 2NT of [F-1]
// announces 0-7 -- it says "nothing to describe", and the opener re-evaluates
// against that ceiling. West holds KT32 64 K863 Q72 opposite East's 2C then
// 2H: no three-card fit, no five-card suit, and eight honour points. Bidding
// the waiting 2NT there announced a ceiling West does not have, East rebid 3H
// as a minimum and the auction died in a partscore with 28 combined honours.
// Eight opposite the eighteen the opening promises already reaches the
// notrump game, so West names it.
func TestStrongTwoClubsWaitingBidDoesNotCapAGoodHand(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:Q9754.J2.94.A983 KT32.64.K863.Q72 J86.T73.QJT7.JT5 A.AKQ985.A52.K64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never answered the 2C rebid\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(3, SNoTrump) {
		t.Fatalf("West's answer = %s (%s), want 3SA\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	if answer.M.minPts != 8 || answer.M.maxPts != 11 {
		t.Fatalf("the answer announces %d-%d, want 8-11 (an open ceiling sends the opener hunting a slam)\nauction: %s",
			answer.M.minPts, answer.M.maxPts, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if !isGame(contract) && contract.Level < 6 {
		t.Fatalf("final contract = %s: the auction still dies below game on 28 honours\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestStrongTwoClubsWaitingBidSurvivesOnABust is the other half: three honour
// points really is nothing to describe, so the waiting 2NT stays, the opener
// repeats his suit as a minimum and the pair stops in the partscore its 23
// honours are worth.
func TestStrongTwoClubsWaitingBidSurvivesOnABust(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:KQ97.JT7.QJT.983 T432.62.9863.QJ7 J865.43.K74.AT52 A.AKQ985.A52.K64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never answered the 2C rebid\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(2, SNoTrump) {
		t.Fatalf("West's answer = %s (%s), want 2SA (the waiting bid)\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract != bid(3, SHearts) {
		t.Fatalf("final contract = %s, want 3C\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestStrongTwoClubsWaitingBidOverASpadeRebid runs the same check over the
// highest suit, where the waiting 2NT is still legal but every other descriptive
// call has moved up a level: East opens 2C on AKQ985 A A52 K64 and rebids 2S,
// West holds 64 KT32 K863 Q72 -- no fit, no suit, eight honour points. The rule
// keys off the hand, not the suit: 3NT, and the pair reaches the eleven tricks
// its 28 honours are worth instead of dying in a partscore.
func TestStrongTwoClubsWaitingBidOverASpadeRebid(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:J2.Q9754.94.A983 64.KT32.K863.Q72 T73.J86.QJT7.JT5 AKQ985.A.A52.K64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	answer, ok := seatNthCall(calls, west, 2)
	if !ok {
		t.Fatalf("West never answered the 2P rebid\nauction: %s", formatAuction(calls))
	}
	if answer.Call != bid(3, SNoTrump) {
		t.Fatalf("West's answer = %s (%s), want 3SA\nauction: %s",
			answer.Call.Format("fr"), answer.M.fr, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if !isGame(contract) && contract.Level < 5 {
		t.Fatalf("final contract = %s: the auction dies below game on 28 honours\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
