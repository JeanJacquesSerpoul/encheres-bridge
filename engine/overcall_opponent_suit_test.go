package main

import "testing"

// TestNoOvercallInOpponentsSuit checks that a suit named by the opponents stays
// off limits for a natural overcall, however late in the auction it was bid.
// Sequence: W 1C, N pass, E 1S, S pass, W 1NT, N pass, E pass, S ? — South holds
// AT762.AK7.3.QJ42 (14H, 15HL, five spades), but East showed four spades with
// its 1S response, so East-West own seven spades against North-South's six.
// Bidding 2S there is a cue-bid, not an overcall; South must find another call.
//
// The engine used to derive the opponents' suit from the last bid alone, which
// is 1NT here, so the spades bid two calls earlier looked free.
func TestNoOvercallInOpponentsSuit(t *testing.T) {
	pbn := `[Dealer "W"]
[Deal "S:AT762.AK7.3.QJ42 J95.QJ9.AJ8.AT97 Q.8642.T9765.K86 K843.T53.KQ42.53"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2 // seats: 0=N, 1=E, 2=S, 3=W
	for _, sc := range calls {
		if sc.Seat != south || !sc.Call.IsBid() {
			continue
		}
		if sc.Call.Strain == SSpades {
			t.Fatalf("South bid %s (%s) in East's suit\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
}
