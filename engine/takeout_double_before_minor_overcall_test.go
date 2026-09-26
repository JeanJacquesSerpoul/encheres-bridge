package engine

import "testing"

// TestTakeoutDoubleBeforeMinorOvercall checks that over a major opening the
// four cards in the other major take priority: a hand that could overcall a
// minor only at the two level doubles instead, as long as it has the takeout
// double's floor and shape. Without the fourth card in the other major the
// natural overcall stays.
func TestTakeoutDoubleBeforeMinorOvercall(t *testing.T) {
	const west = 3
	cases := []struct {
		name string
		pbn  string
		want string
	}{
		{
			// W = T8 KQ92 Q7 AK843 (14 H): 1S - X, not 2C.
			name: "sur 1P, quatre cœurs : contre plutôt que 2T",
			pbn:  `[Dealer "S"][Vulnerable "EW"][Deal "E:K64.AT843.JT.T65 AQ972.75.K98.QJ2 T8.KQ92.Q7.AK843 J53.J6.A65432.97"]`,
			want: "Contre",
		},
		{
			// W = T86 KQ9 Q7 AK843: three hearts only, the takeout shape is
			// missing and the natural 2C stays.
			name: "sur 1P, trois cœurs : 2T reste",
			pbn:  `[Dealer "S"][Vulnerable "EW"][Deal "E:K4.AT8432.JT.T65 AQ972.75.K98.QJ2 T86.KQ9.Q7.AK843 J53.J6.A65432.97"]`,
			want: "2T",
		},
		{
			// Over 1H, W = JT73 A8 3 AK9642 (12 H): four spades, but six
			// clubs keep the natural 2C.
			name: "sur 1C, six trèfles : 2T reste malgré quatre piques",
			pbn:  `[Dealer "S"][Deal "S:KQ.KQ763.KQT5.J3 JT73.A8.3.AK9642 98642.J542.J2.T5 A5.T9.A98764.Q87"]`,
			want: "2T",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParsePBN([]byte(tc.pbn))
			if err != nil {
				t.Fatalf("bad deal: %v", err)
			}
			calls := NewEngine(d).Run()
			for _, sc := range calls {
				if sc.Seat != west {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.want {
					t.Fatalf("first call of West = %s (%s), want %s\nauction: %s",
						got, sc.M.fr, tc.want, formatAuction(calls))
				}
				return
			}
			t.Fatalf("West made no call\nauction: %s", formatAuction(calls))
		})
	}
}
