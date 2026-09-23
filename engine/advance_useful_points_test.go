package engine

import "testing"

// TestAdvanceRaiseCountsUsefulPoints pins the "points « utiles »" clause of
// [A-5]: the three advancing bands are counted after the shortness credit the
// opponents' bidding has made worthless.
//
// South opens 1D, West overcalls 1S, East holds K74 AJ32 Q J8765. Counted in
// plain HLD that hand reaches 14 -- eleven honour points, a fifth club, and
// two more for the singleton diamond -- and cue-bid 2D to ask how strong the
// overcall was [A-6]; West answered "l'ouverture", East proposed, West
// accepted, and the pair played 4S. But the diamond is a bare queen under
// South's opening: the honour falls under their ace, and the ruffing value
// was already paid for by the honour itself. The hand is worth 12, which is
// the jump raise and nothing more.
func TestAdvanceRaiseCountsUsefulPoints(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "None"]
[Deal "S:95.Q76.KJT98.AK2 AQJ83.K54.A72.94 T62.T98.6543.QT3 K74.AJ32.Q.J8765"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat == east && sc.M.overcallAsk {
			t.Fatalf("East asked the overcall's strength (%s) on a singleton queen in their suit\nauction: %s",
				sc.Call.Format("fr"), formatAuction(calls))
		}
	}
	first, ok := firstBidOf(calls, east)
	if !ok || first.Call != bidSuit(3, Spades) {
		t.Fatalf("East's advance = %s, want 3P (jump raise)\nauction: %s",
			first.Call.Format("fr"), formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract.Level >= 4 {
		t.Fatalf("final contract = %s, want a partscore\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestAdvanceKeepsShortnessInAnUnbidSuit guards the other side of the same
// rule: only the opponents' suits are devalued. The same East shape with the
// singleton queen in clubs -- a suit nobody has named, where the ace may still
// be with partner -- keeps its 14 HLD and asks.
func TestAdvanceKeepsShortnessInAnUnbidSuit(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "None"]
[Deal "S:95.Q76.K942.AKT3 AQJ83.K54.AT.942 T62.T98.Q7.J8765 K74.AJ32.J8653.Q"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	asked := false
	for _, sc := range calls {
		if sc.Seat == east && sc.M.overcallAsk {
			asked = true
		}
	}
	if !asked {
		t.Fatalf("East did not ask the overcall's strength with 14 useful HLD\nauction: %s",
			formatAuction(calls))
	}
}

// firstBidOf returns the first actual bid made by a seat.
func firstBidOf(calls []SeatCall, seat int) (SeatCall, bool) {
	for _, sc := range calls {
		if sc.Seat == seat && sc.Call.IsBid() {
			return sc, true
		}
	}
	return SeatCall{}, false
}
