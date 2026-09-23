package engine

import "testing"

// TestCompetitivePassCeiling checks what the responder's pass over an
// intervention announces. South holds T2 KT862 J5 AJ53 -- 9 H, 10 HL -- and
// East's 2C overcall leaves it nothing to bid: a two-over-one needs 11 HL and
// the 1NT escape is gone, notrump now costing the two level. The pass must not
// cap the hand at 7, a ceiling the system never promised; it denies 11 HL and
// nothing less.
//
// Over a one-level intervention the ceiling stays 7: there, an 8-10 hand with a
// stopper takes 1NT instead of passing.
func TestCompetitivePassCeiling(t *testing.T) {
	cases := []struct {
		name string
		pbn  string
		seat int
		want int
	}{
		{
			// N 1D - E 2C - S ? : notrump would cost the two level.
			name: "sur une intervention au palier de 2, le passe ne plafonne qu'à 10",
			pbn:  `[Dealer "N"][Vulnerable "All"][Deal "N:A5.73.AKQT8742.7 K63.AQJ9.9.KQ864 T2.KT862.J5.AJ53 QJ9874.54.63.T92"]`,
			seat: 2, want: 10,
		},
		{
			// N 1D - E 1S - S ? : 1NT is still there for 8-10 with a stopper.
			name: "sur une intervention au palier de 1, le passe reste plafonné à 7",
			pbn:  `[Dealer "N"][Vulnerable "All"][Deal "N:A5.73.AKQT8742.7 KQJ864.AQJ9.9.K2 T2.KT862.J5.AJ53 973.54.63.QT9864"]`,
			seat: 2, want: 7,
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
				if sc.Seat != tc.seat {
					continue
				}
				if sc.Call.Kind != KindPass {
					t.Fatalf("seat %s opened with %s (%s), expected a pass\nauction: %s",
						seatNames[tc.seat], sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
				}
				if sc.M.maxPts != tc.want {
					t.Fatalf("pass announces a ceiling of %d, want %d\nauction: %s", sc.M.maxPts, tc.want, formatAuction(calls))
				}
				return
			}
			t.Fatalf("seat %s never called\nauction: %s", seatNames[tc.seat], formatAuction(calls))
		})
	}
}
