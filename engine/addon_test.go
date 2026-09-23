package main

import (
	"strings"
	"testing"
)

// TestTakeoutDoubleAddon checks the takeout-double rules of docs/addon_1.md:
// the 11-17H zone with its shape conditions, the 18+ any-shape double, the
// rule of the three zones for the answers and the cue-bid development.
// The deals are built around example hands from that document.
func TestTakeoutDoubleAddon(t *testing.T) {
	const west, east = 3, 1
	cases := []struct {
		name string
		pbn  string
		seat int    // seat whose call is checked
		nth  int    // 0 = first call of that seat, 1 = second...
		want string // expected call, French notation
		hint string // substring expected in the French comment
	}{
		{
			// W = AQ72 KT3 974 AJ3 (14H, 4-3 majors) doubles 1D: derrière une
			// mineure le 4-3-3-3 reste un contre d'appel.
			name: "contre d'appel sur 1K, majeures 4-3 en 4-3-3-3",
			pbn: `[Dealer "S"]
[Deal "S:K54.A92.KQJ85.42 AQ72.KT3.974.AJ3 JT9.QJ8.AT3.K765 863.7654.62.QT98"]`,
			seat: west, want: "Contre", hint: "contre d'appel",
		},
		{
			// W = K3 AT87 T53 AK75 (14H) passes over 1D: only two spades.
			name: "passe avec deux cartes à Pique",
			pbn: `[Dealer "S"]
[Deal "S:A54.K32.AQJ86.42 K3.AT87.T53.AK75 QJT.QJ9.K97.QJT9 98762.654.42.863"]`,
			seat: west, want: "Passe",
		},
		{
			// W = AT5 2 A9875 AQ63 (14H) passes over 1H: the fourth spade is
			// missing and no amount of values allows the derogation -- East
			// would bid a spade contract that goes down.
			name: "passe sur 1C sans quatrième Pique, malgré l'ouverture",
			pbn: `[Dealer "S"]
[Deal "S:432.AQJ87.K2.K54 AT5.2.A9875.AQ63 KQJ9.KT96.QJT.J2 876.543.643.T987"]`,
			seat: west, want: "Passe",
		},
		{
			// W = AQ72 K3 KJ54 T87 (13H) doubles 1H: four spades and a
			// doubleton in the opened major.
			name: "contre sur 1C, 4 piques et court à Coeur",
			pbn: `[Dealer "S"]
[Deal "S:K5.AQJ96.A83.932 AQ72.K3.KJ54.T87 JT9.T87.Q96.AKQJ 8643.542.T72.654"]`,
			seat: west, want: "Contre", hint: "contre d'appel",
		},
		{
			// W = AK95 A3 T87 AKJ3 (19H): any-shape double from 18H.
			name: "contre toutes distributions, 18H et plus",
			pbn: `[Dealer "S"]
[Deal "S:876.K52.AQJ43.Q4 AK95.A3.T87.AKJ3 QJT.QJT9.K96.T98 432.8764.52.7652"]`,
			seat: west, want: "Contre", hint: "toutes distributions",
		},
		{
			// E = JT63 QJ4 K954 72 (7H): zone 0-7, minimum answer in the
			// four-card major.
			name: "réponse 0-7H sans saut, priorité à la majeure",
			pbn: `[Dealer "S"]
[Deal "S:A9.K73.Q8.AKJ954 KQ54.A92.AJ32.86 872.T865.T76.QT3 JT63.QJ4.K954.72"]`,
			seat: east, want: "1P", hint: "0-7",
		},
		{
			// E = KJ93 Q7 T63 K954 (9H): zone 8-10, jump answer.
			name: "réponse 8-10H avec saut",
			pbn: `[Dealer "S"]
[Deal "S:8.A93.AKQJ92.Q73 AQ54.KJ82.74.A82 T762.T654.85.JT6 KJ93.Q7.T63.K954"]`,
			seat: east, want: "2P", hint: "saut",
		},
		{
			// E = AJT7 74 AQ92 JT3 (12H): game-forcing zone, cue-bid.
			name: "réponse 11+H, cue-bid",
			pbn: `[Dealer "S"]
[Deal "S:984.QJ9.KJT75.AK KQ65.AK32.86.952 32.T865.43.Q8764 AJT7.74.AQ92.JT3"]`,
			seat: east, want: "2K", hint: "cue-bid",
		},
		{
			// Same deal: the doubler answers the cue-bid with his cheapest
			// four-card major (2H).
			name: "le contreur nomme sa majeure sur le cue-bid",
			pbn: `[Dealer "S"]
[Deal "S:984.QJ9.KJT75.AK KQ65.AK32.86.952 32.T865.43.Q8764 AJT7.74.AQ92.JT3"]`,
			seat: west, nth: 1, want: "2C", hint: "majeure quatrième",
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

func formatAuction(calls []SeatCall) string {
	parts := make([]string, len(calls))
	for i, sc := range calls {
		parts[i] = seatNames[sc.Seat] + ":" + sc.Call.Format("fr")
	}
	return strings.Join(parts, " ")
}
