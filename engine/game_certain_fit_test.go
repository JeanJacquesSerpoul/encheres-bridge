package engine

import "testing"

// TestGameOnlyOnCertainFit replays a reported deal: 1C - 1H - (2D) - 4C, and
// West with 832 KQJ653 2 Q74 bid 4H "on combined strength". East's jump
// rebid is a club one-suiter that may hold a singleton heart -- it did -- so
// the six-card major has no certain eight-card fit, while the nine clubs are
// sure. Game is named only on a certain fit: 5C, not 4H.
func TestGameOnlyOnCertainFit(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "N"][Vulnerable "NS"][Deal "N:54.AT87.AJT9843. A7.9.KQ76.AKJT82 KQJT96.42.5.9653 832.KQJ653.2.Q74"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	const west = 3
	n := 0
	for _, sc := range calls {
		if sc.Seat != west {
			continue
		}
		if n == 1 {
			if sc.Call.IsBid() && sc.Call.Strain == SHearts {
				t.Fatalf("West's second call = %s (%s): game in hearts without a certain fit\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			if got := sc.Call.Format("fr"); got != "5T" {
				t.Fatalf("West's second call = %s (%s), want 5T\nauction: %s", got, sc.M.fr, formatAuction(calls))
			}
			return
		}
		n++
	}
	t.Fatalf("West made fewer than two calls\nauction: %s", formatAuction(calls))
}

// TestNoControlsOverFiveOfTheMinor continues the same deal: over West's 5C,
// East's 5D cue-bid left no sign-off but 6C, off the heart ace and a spade.
// Above five of the minor a control commits to slam, so without a certain
// one East passes and the side plays 5C.
func TestNoControlsOverFiveOfTheMinor(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "N"][Vulnerable "NS"][Deal "N:54.AT87.AJT9843. A7.9.KQ76.AKJT82 KQJT96.42.5.9653 832.KQJ653.2.Q74"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "5T" {
		t.Fatalf("final contract = %s, want 5T\nauction: %s", got, formatAuction(calls))
	}
}
