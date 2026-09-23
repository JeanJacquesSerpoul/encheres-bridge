package main

import (
	"strings"
	"testing"
)

// TestReservedConventionalCall guards the two rebids that announce a call and
// then hand it to a convention: the jump 2NT rebid, alerted "3C checkback
// available" [§8.2], and the plain 1NT rebid, alerted "2C Roudi available"
// [§8.3]. A responder with a long major and slam values used to start the
// control exchange on that very call -- partner would have answered the
// question, not heard the control. He must repeat his major instead, which
// agrees the trump suit the cue-bids are about and leaves the reserved call
// where the convention put it.
func TestReservedConventionalCall(t *testing.T) {
	cases := []struct {
		name     string
		pbn      string
		seat     int // the seat whose second call is examined
		nth      int
		reserved string // the call the convention owns, which must not appear
		want     string
		contract string
	}{
		{
			// N = AQ6 AQT8652 T4 8 (13H, seven hearts) facing the 18-19
			// notrump: slam is in view, but 3C is the checkback.
			name: "3T reste le Checkback sur le saut à 2SA",
			pbn: `[Dealer "E"]
[Deal "S:KJT3.K3.AK.KQ765 9852.94.J963.932 AQ6.AQT8652.T4.8 74.J7.Q8752.AJT4"]`,
			seat:     0,
			nth:      1,
			reserved: "3T",
			want:     "3C",
			contract: "6C",
		},
		{
			// S = AQT543 KT5 AK5 9 (16H, six spades) facing the 12-14
			// notrump: same story one level lower, where 2C is the Roudi.
			name: "2T reste le Roudi sur la redemande à 1SA",
			pbn: `[Dealer "N"]
[Deal "N:K9.9764.Q6.AKQ54 8.J83.JT943.J872 AQT543.KT5.AK5.9 J762.AQ2.872.T63"]`,
			seat:     2,
			nth:      1,
			reserved: "2T",
			want:     "2P",
			contract: "6P",
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
				if sc.Call.Format("fr") == tc.reserved && !strings.Contains(sc.M.fr, "Checkback") && !strings.Contains(sc.M.fr, "Roudi") {
					t.Fatalf("%s bid as %q, but the rebid reserved it for the convention\nauction: %s",
						tc.reserved, sc.M.fr, formatAuction(calls))
				}
			}
			n := 0
			for _, sc := range calls {
				if sc.Seat != tc.seat || !sc.Call.IsBid() {
					continue
				}
				if n == tc.nth {
					if got := sc.Call.Format("fr"); got != tc.want {
						t.Fatalf("call #%d of %s = %s (%s), want %s\nauction: %s",
							tc.nth, seatNames[tc.seat], got, sc.M.fr, tc.want, formatAuction(calls))
					}
					if !sc.M.forcing {
						t.Fatalf("the repeat of the major must stay forcing: %q\nauction: %s", sc.M.fr, formatAuction(calls))
					}
					break
				}
				n++
			}
			contract, _, _ := finalContract(calls)
			if got := contract.Format("fr"); got != tc.contract {
				t.Fatalf("final contract = %s, want %s\nauction: %s", got, tc.contract, formatAuction(calls))
			}
		})
	}
}
