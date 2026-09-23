package engine

import "testing"

// TestGameNotInviteOn25 replays a deal where East opens 1D (12-23 HL) and
// West holds 13 honour points with the overcalled suit stopped: the combined
// honour count reaches 25 even opposite a dead-minimum opening, so West must
// conclude 3NT, not propose 2NT. The old "marginal notrump game" downgrade
// turned this into an invitation that East, misreading West's cap, then
// declined -- stranding the pair in 2NT with game values. The distributional
// case that downgrade was written for (a count reaching 25 only through
// trump-fit HLD points) is covered by the honour-only cMinNT threshold in
// gameCall.
func TestGameNotInviteOn25(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:AT7.QT72.765.832 43.A96.AQJ42.Q76 KJ962.K43.T8.JT4 Q85.J85.K93.AK95"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const west = 3 // seats: 0=N, 1=E, 2=S, 3=W
	for _, sc := range calls {
		if sc.Seat != west || sc.Call.Kind != KindBid {
			continue
		}
		if got := sc.Call.Format("fr"); got != "3SA" {
			t.Fatalf("West's bid = %s (%s), want 3SA on 13H opposite an opening\nauction: %s",
				got, sc.M.fr, formatAuction(calls))
		}
		break
	}
	final, _, _ := finalContract(calls)
	if final.Format("fr") != "3SA" {
		t.Fatalf("final contract = %s, want 3SA\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}
