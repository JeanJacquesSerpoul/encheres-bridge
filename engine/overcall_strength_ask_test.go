package engine

import (
	"strings"
	"testing"
)

// TestOvercallStrengthAsk checks the cue-bid that asks how strong a one-level
// suit overcall was. The overcall spans 9-18 HL: with a fit -- three cards at
// least -- and a hand too strong for the capped raises, the advancer bids the
// opener's suit, and the answer is coded: without opening values the overcaller
// returns to his own suit, with them he describes his hand [A-6].
// TestOvercallStrengthAskNeedsAFit is the deal that produced the rule: South
// passes, North opens 1C, South answers 1H, West overcalls 1S. East holds
// 3 AQ43 K96532 65 -- 11 HL, but a singleton in West's spades. The engine used
// to cue-bid 3C there to ask West's zone, and West, hearing a strong hand,
// jumped to 4S on a seven-card fit. Asking now takes three cards in the
// overcall suit [A-6], so East stays quiet.
func TestOvercallStrengthAskNeedsAFit(t *testing.T) {
	const east = 1
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:K4.K9876.JT4.QJ2 QJ9752.JT2.AQ.43 AT86.5.87.AKT987 3.AQ43.K96532.65"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat == east && sc.M.overcallAsk {
			t.Fatalf("East asked the overcall's strength (%s, %q) with a singleton spade\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
	}
	if contract, _, _ := finalContract(calls); contract == bid(4, SSpades) {
		t.Fatalf("the auction still lands in 4P on a seven-card fit\nauction: %s", formatAuction(calls))
	}
}

// TestOvercallStrengthAskShapelyMinimum pins the HL -- not HLD -- reading of
// the "opening values" cut [A-6]. E opens 1D, S overcalls 1S on
// KJ765 9 AT8532 2: 8 H -- the one-level floor [I-5] -- 11 HL, a genuine
// minimum. N cue-bids 2D to ask.
// Counted in HLD the spade fit plus two singletons reach 14, and the engine
// used to answer 2NT -- "l'ouverture, arrêt dans leur couleur" -- on a hand
// with a stiff club and a stiff heart. The reply must be the plain return to
// spades, announcing a minimum.
func TestOvercallStrengthAskShapelyMinimum(t *testing.T) {
	const south = 2
	d, err := ParsePBN([]byte(`[Dealer "E"]
[Vulnerable "NS"]
[Deal "E:QT9.KQJ6.J974.A7 KJ765.9.AT8532.2 2.A87532.6.KT865 A843.T4.KQ.QJ943"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	seen := 0
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		seen++
		if seen != 2 { // 1S overcall, then the reply to the cue-bid
			continue
		}
		if got := sc.Call.Format("fr"); got != "2P" {
			t.Fatalf("South's reply to the cue-bid = %s (%q), want 2P\nauction: %s",
				got, sc.M.fr, formatAuction(calls))
		}
		if !strings.Contains(sc.M.fr, "sans l'ouverture") {
			t.Fatalf("comment %q does not announce a minimum overcall", sc.M.fr)
		}
		return
	}
	t.Fatalf("South made fewer than two calls\nauction: %s", formatAuction(calls))
}

func TestOvercallStrengthAsk(t *testing.T) {
	const west, east = 3, 1
	// S opens 1H, W overcalls 1S on KQJ54 73 T842 Q6 (8H, no opening), East
	// holds T98 K64 AQJ6 KJ7: three spades and 14 HLD, too much for the
	// simple raise the auction would otherwise allow.
	const weakOvercall = `[Dealer "S"]
[Deal "S:A6.AQT82.K73.952 KQJ54.73.T842.Q6 732.J95.95.AT843 T98.K64.AQJ6.KJ7"]`
	// Same auction, but West holds KQJ54 73 KQ92 86: opening values, so he
	// describes instead of signing off. East (T98 964 AJ5 AKJ3) has the same
	// three-card fit and no heart stopper at all.
	const strongOvercall = `[Dealer "S"]
[Deal "S:A6.AQT82.764.Q52 KQJ54.73.KQ92.86 732.KJ5.T83.T974 T98.964.AJ5.AKJ3"]`

	cases := []struct {
		name string
		pbn  string
		seat int
		nth  int
		want string
		hint string
	}{
		{
			name: "l'avancée demande le niveau de l'intervention",
			pbn:  weakOvercall,
			seat: east, want: "3C", hint: "demande le niveau",
		},
		{
			name: "sans l'ouverture, retour à la couleur d'intervention",
			pbn:  weakOvercall,
			seat: west, nth: 1, want: "3P", hint: "sans l'ouverture",
		},
		{
			name: "le cue-bid ne demande pas d'arrêt",
			pbn:  strongOvercall,
			seat: east, want: "2C", hint: "demande le niveau",
		},
		{
			name: "avec l'ouverture, l'intervenant décrit son jeu",
			pbn:  strongOvercall,
			seat: west, nth: 1, want: "3K", hint: "l'ouverture",
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
