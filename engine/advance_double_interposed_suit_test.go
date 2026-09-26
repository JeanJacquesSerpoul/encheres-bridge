package engine

import "testing"

// TestAdvanceDoubleSkipsInterposedSuit checks that the advancer never answers
// the takeout double in the suit the responder has just bid. East opened 1D,
// South doubled, West bid 1S: North, with four spades (KJ43) and 8 H, used to
// answer 2S "four cards, 0-10", bidding West's own suit.
func TestAdvanceDoubleSkipsInterposedSuit(t *testing.T) {
	pbn := `[Dealer "E"]
[Deal "S:Q98.K743.KQ.QJ96 A762.QT5.T76.T52 KJ43.J98.9852.K7 T5.A62.AJ43.A843"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	if len(calls) < 4 {
		t.Fatalf("auction too short: %s", formatAuction(calls))
	}
	if w := calls[2].Call; !w.IsBid() || w.Level != 1 || w.Strain != SSpades {
		got := w.Format("fr")
		t.Skipf("West no longer bids 1S over the double (%s): scenario gone\nauction: %s", got, formatAuction(calls))
	}
	north := calls[3]
	if north.Call.IsBid() && north.Call.Strain == SSpades {
		t.Fatalf("North advance = %s (%s): bids West's suit\nauction: %s",
			north.Call.Format("fr"), north.M.fr, formatAuction(calls))
	}
}
