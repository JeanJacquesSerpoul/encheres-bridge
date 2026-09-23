package main

import "testing"

// TestWeakTwoNeedsHonoursInSuit replays a reported deal where South opened a
// weak 2S on JT9872 with 7H of which a single point sat in the suit (the
// rest: QJ of hearts, K of diamonds). A weak two wants its honour strength
// concentrated in the long major -- with the points mostly outside, the hand
// passes (too much defence, too weak a suit).
//
// The corrected auction then lets West (18H, six diamonds) open 2C and rebid
// 3D -- not a 2NT mislabelled "22-23" -- and East-West stop at 4D, the par
// contract (+130).
func TestWeakTwoNeedsHonoursInSuit(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:Q64.AT762.82.KQ5 5.K98.65.JT98643 JT9872.QJ4.KT7.7 AK3.53.AQJ943.A2"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south, west = 2, 3
	for _, sc := range calls {
		if sc.Seat == south {
			if sc.Call.Kind != KindPass {
				t.Fatalf("South's first call = %s (%s), want Passe (only 1H in the long suit)\nauction: %s",
					sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
			}
			break
		}
	}

	n := 0
	for _, sc := range calls {
		if sc.Seat != west || sc.Call.Kind != KindBid {
			continue
		}
		n++
		switch n {
		case 1:
			if got := sc.Call.Format("fr"); got != "2T" {
				t.Fatalf("West's opening = %s, want 2T\nauction: %s", got, formatAuction(calls))
			}
		case 2:
			if got := sc.Call.Format("fr"); got != "3K" {
				t.Fatalf("West's rebid = %s (%s), want 3K (six-card suit, 18H -- not a 22-23 notrump)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
		}
	}

	final, _, _ := finalContract(calls)
	if final.Format("fr") != "4K" {
		t.Fatalf("final contract = %s, want 4K (par +130)\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}

// TestPreemptNeedsHonoursInSuit checks the same concentration rule on every
// preempt level: at least half the hand's honour points must sit in the long
// suit, whether the opening is a weak two (6 cards), a three-level (7) or a
// four-level (8) barrage.
func TestPreemptNeedsHonoursInSuit(t *testing.T) {
	cases := []struct {
		name string
		h    *Hand
		want string // "" = no preempt
	}{
		{"7 cartes, honneurs dehors", hand("JT98742", "KQ", "Q32", "4"), ""},
		{"7 cartes, honneurs dedans", hand("KQJ9742", "32", "Q32", "4"), "3P"},
		{"8 cartes, honneurs dehors", hand("JT987642", "A", "K32", ""), ""},
		{"8 cartes, honneurs dedans", hand("KQJT8742", "3", "432", "2"), "4P"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := NewEngine(dealWith(0, map[int]*Hand{0: tc.h}))
			c, _, ok := e.preemptOpening(tc.h)
			if tc.want == "" {
				if ok {
					t.Fatalf("preempt = %s, want none (honours outside the suit)", c.Format("fr"))
				}
				return
			}
			if !ok || c.Format("fr") != tc.want {
				t.Fatalf("preempt = %v (ok=%v), want %s", c.Format("fr"), ok, tc.want)
			}
		})
	}
}
