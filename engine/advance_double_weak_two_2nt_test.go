package engine

import "testing"

// TestAdvanceDoubleOfWeakTwoWith2NT checks the natural 2NT answer to a double
// of a weak two. South opened 2S, East doubled in the balancing seat: West,
// balanced with AQ95 in spades and 10 H, used to answer 3D "0-7", the minimum
// answer, for want of a notrump answer above the one level.
func TestAdvanceDoubleOfWeakTwoWith2NT(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:KJ6432.J65.J75.7 AQ95.87.K864.J32 87.Q93.AQ3.K9654 T.AKT42.T92.AQT8"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	if len(calls) < 6 || calls[3].Call.Kind != KindDouble {
		t.Skipf("East no longer doubles 2S: scenario gone\nauction: %s", formatAuction(calls))
	}
	if w := calls[5].Call; !w.IsBid() || w.Level != 2 || w.Strain != SNoTrump {
		t.Fatalf("West advance = %s (%s), want 2NT\nauction: %s",
			w.Format("fr"), calls[5].M.fr, formatAuction(calls))
	}
}
