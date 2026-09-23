package engine

import "testing"

// TestQuantitative4NT replays the three worked examples of docs/addon_4.md:
// over a 1NT opening (15-17), a quantitative 4NT asks the opener to sign off
// at 15, bid 5NT at 16, or bid 6NT at 17 (never a plain binary accept/decline).
func TestQuantitative4NT(t *testing.T) {
	const north, south = 0, 2
	cases := []struct {
		name       string
		northHand  *Hand
		southHand  *Hand
		northFinal string
	}{
		{
			// North 17H (maximum): certain of 33+ combined, direct 6SA.
			name:       "exemple 1 : ouvreur maximum, conclusion directe",
			northHand:  hand("AQJ6", "K42", "65", "AK95"),
			southHand:  hand("K2", "QJT", "AK7", "QJ876"),
			northFinal: "6SA",
		},
		{
			// North 16H (middle of 15-17): partial acceptance at 5SA.
			name:       "exemple 2 : ouvreur milieu de zone, 5SA",
			northHand:  hand("AT82", "KQ97", "K43", "A8"),
			southHand:  hand("KQ9", "AJ6", "AQ95", "976"),
			northFinal: "5SA",
		},
		{
			// North 17H again: direct 6SA.
			name:       "exemple 3 : ouvreur maximum, conclusion directe",
			northHand:  hand("KQT", "AJT2", "AT7", "K86"),
			southHand:  hand("A8", "K96", "KQ865", "AT7"),
			northFinal: "6SA",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWith(north, map[int]*Hand{north: tc.northHand, south: tc.southHand})
			calls := NewEngine(d).Run()

			n := 0
			for _, sc := range calls {
				if sc.Seat != north {
					continue
				}
				n++
				switch n {
				case 1:
					if got := sc.Call.Format("fr"); got != "1SA" {
						t.Fatalf("North's opening = %s, want 1SA\nauction: %s", got, formatAuction(calls))
					}
				case 2:
					if got := sc.Call.Format("fr"); got != tc.northFinal {
						t.Fatalf("North's final word = %s (%s), want %s\nauction: %s", got, sc.M.fr, tc.northFinal, formatAuction(calls))
					}
					return
				}
			}
			t.Fatalf("North never got a second call\nauction: %s", formatAuction(calls))
		})
	}
}
