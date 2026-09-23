package engine

import (
	"strings"
	"testing"
)

// TestTakeoutDoubleZones checks the takeout double as the reference synthesis
// states it: a 12-point floor (12 HL over a minor, 12 H over a major) with
// shortness in the opened suit and no derogation on the four cards promised in
// the other major, then the three zones of the answers -- the ladder of jumps
// anchored on game, and the notrump answers with their precise stopper price.
func TestTakeoutDoubleZones(t *testing.T) {
	const west, east = 3, 1
	cases := []struct {
		name string
		pbn  string
		seat int
		nth  int
		want string
		hint string
	}{
		{
			// W = AJ54 K932 96 KT9: the ideal shape, but 11 H only.
			name: "passe avec 11H, le contre demande 12",
			pbn:  `[Dealer "S"][Deal "S:KQ2.A4.KJT54.543 AJ54.K932.96.KT9 963.T875.732.QJ2 T87.QJ6.AQ8.A876"]`,
			seat: west, want: "Passe",
		},
		{
			// W = AQ54 KJ3 Q962 T9 (12H): the majors are there and the 4-3-3-3
			// would double, but four cards in the opened minor are real length
			// in their suit.
			name: "passe avec quatre cartes dans la mineure d'ouverture",
			pbn:  `[Dealer "S"][Deal "S:KJ2.A4.KJT54.543 AQ54.KJ3.Q962.T9 96.T9872.A3.J762 T873.Q65.87.AKQ8"]`,
			seat: west, want: "Passe",
		},
		{
			// E = 754 KJ54 Q87 K65 (9H): over 1S the heart answer has no jump
			// available, so 2H carries the weak and the middle zone at once.
			name: "sur 1P, 2C couvre 0-10H avec quatre cartes",
			pbn:  `[Dealer "S"][Deal "S:AQJT98.T9.52.A43 32.AQ76.AKJ4.Q92 K6.832.T963.JT87 754.KJ54.Q87.K65"]`,
			seat: east, want: "2C", hint: "0-10",
		},
		{
			// Same shortage of room: with five hearts and the middle zone the
			// answer is 3H, and game 4H would take six.
			name: "sur 1P, 3C montre cinq cartes en zone moyenne",
			pbn:  `[Dealer "S"][Deal "S:AQJ982.T9.K52.A4 K3.AQ76.AJ43.T92 T54.82.T96.QJ873 76.KJ543.Q87.K65"]`,
			seat: east, want: "3C", hint: "5 cartes",
		},
		{
			// W = 32 AQ76 AKJ4 Q92 (16 HLD, nothing wasted in spades) may try
			// for game at 3H; East, in the middle zone of his wide answer,
			// accepts.
			name: "l'essai à 3C est accepté en zone moyenne",
			pbn:  `[Dealer "S"][Deal "S:AQJT98.T9.52.A43 32.AQ76.AKJ4.Q92 K6.832.T963.JT87 754.KJ54.Q87.K65"]`,
			seat: east, nth: 1, want: "4C", hint: "accepte",
		},
		{
			// E = J87 Q65 KJ9 A876 (11H): no four-card major, a stopper and a
			// half in their suit.
			name: "2SA, 11H et arrêt et demi",
			pbn:  `[Dealer "S"][Deal "S:K92.A4.AT5432.54 AQ54.KJ32.76.QT9 T63.T987.Q8.KJ32 J87.Q65.KJ9.A876"]`,
			seat: east, want: "2SA", hint: "arrêt et demi",
		},
		{
			// E = T87 Q65 AQ8 AJ76 (13H): two stoppers in their suit, 3NT.
			name: "3SA, 12-14H et deux arrêts",
			pbn:  `[Dealer "S"][Deal "S:KQ2.A4.KJT54.543 AJ54.KJ32.96.KT9 963.T987.732.Q82 T87.Q65.AQ8.AJ76"]`,
			seat: east, want: "3SA", hint: "deux arrêts",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParsePBN([]byte(tc.pbn))
			if err != nil {
				t.Fatalf("bad deal: %v", err)
			}
			calls := NewEngine(d).Run()
			n := 0
			for _, sc := range calls {
				if sc.Seat != tc.seat {
					continue
				}
				if n == tc.nth {
					if got := sc.Call.Format("fr"); got != tc.want {
						t.Fatalf("call #%d of seat %s = %s (%s), want %s\nauction: %s",
							tc.nth, seatNames[tc.seat], got, sc.M.fr, tc.want, formatAuction(calls))
					}
					if tc.hint != "" && !strings.Contains(sc.M.fr, tc.hint) {
						t.Fatalf("comment %q does not mention %q", sc.M.fr, tc.hint)
					}
					return
				}
				n++
			}
			t.Fatalf("seat %s made fewer than %d calls\nauction: %s", seatNames[tc.seat], tc.nth+1, formatAuction(calls))
		})
	}
}
