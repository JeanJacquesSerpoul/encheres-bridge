package main

import (
	"strings"
	"testing"
)

// TestAdvanceLongSuitOverNotrumpOvercall reproduces the reported deal. South
// opens 1D, North answers 1H, East overcalls 1NT (15-17 with a stopper), South
// rebids 2D. West holds J84 64 72 AKT974: 8 H -- but a six-card club suit
// headed by AK, worth 10 HL, which puts the side on 25 opposite East's floor.
//
// West used to pass, and the pair played 2NT for North-South. The generic
// endgame [A-4] valued West on honours alone, the rule for notrump ("a long
// side suit may not run"), and a suit that does run has no way to be heard.
// West must name it; East, who holds the stoppers West lacks, then places 3NT
// -- which the double-dummy solver makes exactly.
func TestAdvanceLongSuitOverNotrumpOvercall(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "EW"]
[Deal "S:Q7.98.AQJ98653.5 J84.64.72.AKT974 T963.KJT75..QJ62 AK52.AQ32.KT4.83"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	var bid3C *SeatCall
	for i := range calls {
		if calls[i].Seat == west && calls[i].Call.IsBid() {
			bid3C = &calls[i]
			break
		}
	}
	if bid3C == nil {
		t.Fatalf("West never bid, burying the game\nauction: %s", formatAuction(calls))
	}
	if bid3C.Call != bidSuit(3, Clubs) {
		t.Fatalf("West's bid = %s (%s), want 3T (the six-card club suit)\nauction: %s",
			bid3C.Call.Format("fr"), bid3C.M.fr, formatAuction(calls))
	}
	contract, decl, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) || sideOf(decl) != sideOf(west) {
		t.Fatalf("final contract = %s by %s, want 3SA for East-West\nauction: %s",
			contract.Format("fr"), seatNames[decl], formatAuction(calls))
	}
}

// TestNotrumpOvercallRecordsItsStopper: the 1NT overcall says "et arrêt" [I-4]
// and its own guard requires one, so the guarantee must reach partner --
// otherwise his notrump arithmetic (ntSafe) keeps refusing the contract for
// want of a stopper that was already promised.
func TestNotrumpOvercallRecordsItsStopper(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:Q7.98.AQJ98653.5 J84.64.72.AKT974 T963.KJT75..QJ62 AK52.AQ32.KT4.83"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	e := NewEngine(d)
	e.Run()
	if !e.ps[east].shownStop[Diamonds] {
		t.Fatalf("East's 1SA overcall did not record the diamond stopper it promises")
	}
}

// TestNoLongSuitAdvanceWithoutTheValues guards the other side: the six-card
// suit is only worth naming when its length points carry the side to the game
// zone opposite the overcall's floor. A ragged suit on a weak hand passes.
func TestNoLongSuitAdvanceWithoutTheValues(t *testing.T) {
	// South opens 1D, North passes, East overcalls 1NT, South rebids 2D, and
	// West decides.
	const north, east, south, west = 0, 1, 2, 3
	adv := func(h *Hand) (Call, meaning) {
		e := &Engine{opener: south}
		for i := 0; i < 4; i++ {
			e.ps[i] = &playerState{seat: i, hand: &Hand{}, shownMax: 40}
		}
		e.ps[west].hand = h
		e.ps[east].shownMin, e.ps[east].shownMax = 15, 17
		e.openCall = bidSuit(1, Diamonds)
		e.calls = []SeatCall{
			{Seat: south, Call: bidSuit(1, Diamonds)},
			{Seat: west, Call: passCall},
			{Seat: north, Call: passCall},
			{Seat: east, Call: bid(1, SNoTrump)},
			{Seat: south, Call: bidSuit(2, Diamonds)},
		}
		return e.advance(e.ps[west])
	}

	// 8 H and a running six-card suit: 10 HL opposite 15 reaches 25, so it is
	// named.
	if c, mn := adv(hand("J84", "64", "72", "AKT974")); c != bidSuit(3, Clubs) {
		t.Fatalf("strong six-card suit: bid = %s (%s), want 3T", c.Format("fr"), mn.fr)
	} else if !strings.Contains(mn.fr, "couleur longue") {
		t.Fatalf("comment %q should name the long suit", mn.fr)
	}
	// The same shape without the top honours: 6 HL opposite 15 is 21, well
	// short of the game zone, and the suit stays unsaid.
	if c, _ := adv(hand("J84", "64", "72", "T97643")); c.Kind != KindPass {
		t.Fatalf("ragged six-card suit on a weak hand: bid = %s, want Passe", c.Format("fr"))
	}
}
