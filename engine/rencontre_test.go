package main

import (
	"strings"
	"testing"
)

// scaffoldRencontre builds a minimal engine state: an opening call by
// `opener`, followed by whatever extra calls describe the interference,
// ready for the seat under test to act next.
func scaffoldRencontre(opener int, openCall Call, extra ...SeatCall) *Engine {
	e := &Engine{opener: opener, openCall: openCall}
	for i := 0; i < 4; i++ {
		e.ps[i] = &playerState{seat: i, hand: &Hand{}, shownMax: 40}
	}
	e.calls = append(e.calls, SeatCall{Seat: opener, Call: openCall, M: noInfo()})
	e.calls = append(e.calls, extra...)
	return e
}

// ---------- Section I: without interference ----------

func TestRencontreMinorOpening(t *testing.T) {
	const north, south = 0, 2
	cases := []struct {
		name      string
		southHand *Hand
		northHand *Hand
		wantOpen  string
		wantResp  string
	}{
		{
			name:      "1T - 2K : rencontre, fit Trefle et 5+ Carreau",
			southHand: hand("Q8", "K7", "J987", "AKJ96"),
			northHand: hand("432", "5", "AKQ32", "Q432"),
			wantOpen:  "1T",
			wantResp:  "2K",
		},
		{
			name:      "1K - 3T : rencontre, fit Carreau et 5+ Trefle",
			southHand: hand("K", "AT32", "AKQ8", "J932"),
			northHand: hand("98", "65", "J432", "AKQ76"),
			wantOpen:  "1K",
			wantResp:  "3T",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWithQuietOpponents(south, map[int]*Hand{south: tc.southHand, north: tc.northHand})
			calls := NewEngine(d).Run()
			for _, sc := range calls {
				if sc.Seat != south {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.wantOpen {
					t.Fatalf("South's opening = %s, want %s\nauction: %s", got, tc.wantOpen, formatAuction(calls))
				}
				break
			}
			for _, sc := range calls {
				if sc.Seat != north {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.wantResp {
					t.Fatalf("North's answer = %s (%s), want %s\nauction: %s", got, sc.M.fr, tc.wantResp, formatAuction(calls))
				}
				if !strings.Contains(sc.M.fr, "rencontre") {
					t.Fatalf("comment %q does not mention rencontre", sc.M.fr)
				}
				return
			}
			t.Fatalf("North never called\nauction: %s", formatAuction(calls))
		})
	}
}

// TestRencontreWeak2 replays the document's own example: South opens a weak
// 2H, West is silent, North answers 4D showing a good six-card diamond
// suit and 3-card heart support, 8-11 HCP (docs/addon_6.md).
func TestRencontreWeak2(t *testing.T) {
	const south, north = 2, 0
	southHand := hand("42", "AKT762", "T76", "83")
	northHand := hand("96", "QJ8", "AK9854", "94")
	d := dealWithQuietOpponents(south, map[int]*Hand{south: southHand, north: northHand})
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if got := sc.Call.Format("fr"); got != "2C" {
			t.Fatalf("South's opening = %s, want 2C (weak two hearts)\nauction: %s", got, formatAuction(calls))
		}
		break
	}
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if got := sc.Call.Format("fr"); got != "4K" {
			t.Fatalf("North's answer = %s (%s), want 4K\nauction: %s", got, sc.M.fr, formatAuction(calls))
		}
		if !strings.Contains(sc.M.fr, "rencontre") {
			t.Fatalf("comment %q does not mention rencontre", sc.M.fr)
		}
		return
	}
	t.Fatalf("North never called\nauction: %s", formatAuction(calls))
}

// ---------- Section II: after an adverse double ----------

func TestRencontreAfterDouble(t *testing.T) {
	const responder = 0 // North, opener's partner
	cases := []struct {
		name     string
		opener   int
		openCall Call
		hand     *Hand
		want     string
	}{
		{
			// 1D-X: the other minor, single jump.
			name:     "1K contre : 3T rencontre",
			opener:   2,
			openCall: bid(1, SDiamonds),
			hand:     hand("32", "54", "9876", "AKQ92"),
			want:     "3T",
		},
		{
			// 1H-X: any new suit, single jump (spades ranks above hearts, so
			// the natural reply is already 1S and the jump lands on 2S).
			name:     "1C contre : 2P rencontre",
			opener:   2,
			openCall: bid(1, SHearts),
			hand:     hand("AK892", "8765", "J32", "4"),
			want:     "2P",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := scaffoldRencontre(tc.opener, tc.openCall,
				SeatCall{Seat: sideOpponent(tc.opener), Call: doubleCall, M: noInfo()})
			e.ps[responder].hand = tc.hand
			c, mn := e.respondCompetitive(e.ps[responder])
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("answer = %s (%s), want %s", got, mn.fr, tc.want)
			}
			if !strings.Contains(mn.fr, "rencontre") {
				t.Fatalf("comment %q does not mention rencontre", mn.fr)
			}
		})
	}
}

// sideOpponent returns a seat on the opposing side to seat.
func sideOpponent(seat int) int { return (seat + 1) % 4 }

// ---------- Section III: after an adverse overcall ----------

func TestRencontreAfterOvercall(t *testing.T) {
	const responder = 0
	cases := []struct {
		name string
		hand *Hand
		want string
	}{
		{
			// Simple 1-level overcall: the rencontre needs a double jump.
			name: "1C 1P intervention : 4T rencontre (double saut)",
			hand: hand("32", "8765", "4", "AKQ92"),
			want: "4T",
		},
		{
			// Weak two-level major intervention over a minor opening: the
			// rencontre needs only a single jump.
			name: "1K 2C intervention : 3P rencontre (simple saut)",
			hand: hand("AK892", "4", "8765", "J32"),
			want: "3P",
		},
	}
	openers := map[string]struct {
		opener   int
		openCall Call
		rho      Call
	}{
		"1C 1P intervention : 4T rencontre (double saut)": {2, bid(1, SHearts), bid(1, SSpades)},
		"1K 2C intervention : 3P rencontre (simple saut)": {2, bid(1, SDiamonds), bid(2, SHearts)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := openers[tc.name]
			e := scaffoldRencontre(o.opener, o.openCall,
				SeatCall{Seat: sideOpponent(o.opener), Call: o.rho, M: noInfo()})
			e.ps[responder].hand = tc.hand
			c, mn := e.respondCompetitive(e.ps[responder])
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("answer = %s (%s), want %s", got, mn.fr, tc.want)
			}
			if !strings.Contains(mn.fr, "rencontre") {
				t.Fatalf("comment %q should mention rencontre", mn.fr)
			}
		})
	}
}

// TestRencontreExceptionNaturalGameLevel checks that a jump landing exactly
// at a major's own game level is excluded from the rencontre reading
// (docs/addon_6.md, section III exception): "1H 2x 4S" and "1H 2S 4H"-style
// sequences deny a fit rather than showing one.
func TestRencontreExceptionNaturalGameLevel(t *testing.T) {
	const responder = 0
	cases := []struct {
		name     string
		opener   int
		openCall Call
		rho      Call
		hand     *Hand
	}{
		{
			// A double jump to 4S over a 1H opening lands exactly at spades'
			// own game level: excluded, even though the shape would
			// otherwise qualify.
			name:     "1C 2T intervention : 4P exclu de la rencontre",
			opener:   2,
			openCall: bid(1, SHearts),
			rho:      bid(2, SClubs),
			hand:     hand("AKQ92", "876", "K54", "32"),
		},
		{
			// A single jump to 4H over a 1D opening + a weak 2S intervention
			// lands exactly at hearts' own game level: excluded.
			name:     "1K 2P intervention : 4C exclu de la rencontre",
			opener:   2,
			openCall: bid(1, SDiamonds),
			rho:      bid(2, SSpades),
			hand:     hand("4", "AK892", "8765", "J32"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := scaffoldRencontre(tc.opener, tc.openCall,
				SeatCall{Seat: sideOpponent(tc.opener), Call: tc.rho, M: noInfo()})
			e.ps[responder].hand = tc.hand
			_, mn := e.respondCompetitive(e.ps[responder])
			if strings.Contains(mn.fr, "rencontre") {
				t.Fatalf("comment %q should not mention rencontre: a jump to a major's own game level stays natural, to play", mn.fr)
			}
		})
	}
}

// ---------- Section IV: answering partner's intervention ----------

func TestRencontreAdvance(t *testing.T) {
	const overcaller, advancer = 2, 0 // South overcalls, North advances
	e := &Engine{opener: 1, openCall: bid(1, SDiamonds)}
	for i := 0; i < 4; i++ {
		e.ps[i] = &playerState{seat: i, hand: &Hand{}, shownMax: 40}
	}
	e.calls = []SeatCall{
		{Seat: 1, Call: bid(1, SDiamonds), M: noInfo()},
		{Seat: overcaller, Call: bidSuit(1, Hearts), M: m(9, 18, "intervention au palier de 1, 5 cartes et plus", "one-level overcall, 5+ cards").withLen(Hearts, 5)},
		{Seat: 3, Call: passCall, M: noInfo()},
	}
	e.ps[overcaller].lastM = &e.calls[1].M
	advancerHand := hand("AK892", "8765", "J32", "4")
	e.ps[advancer].hand = advancerHand
	c, mn := e.advance(e.ps[advancer])
	if got := c.Format("fr"); got != "2P" {
		t.Fatalf("answer = %s (%s), want 2P", got, mn.fr)
	}
	if !strings.Contains(mn.fr, "rencontre") {
		t.Fatalf("comment %q does not mention rencontre", mn.fr)
	}
}

// ---------- Section V: after a prior pass ----------

func TestRencontreAfterPriorPass(t *testing.T) {
	const north, south, east, west = 0, 2, 1, 3
	cases := []struct {
		name      string
		southHand *Hand
		northHand *Hand
		wantOpen  string
		wantResp  string
	}{
		{
			name:      "1C - 2P apres passe prealable de Nord",
			southHand: hand("Q65", "AKQ32", "K987", "K"),
			northHand: hand("AK874", "9876", "J32", "3"),
			wantOpen:  "1C",
			wantResp:  "2P",
		},
		{
			name:      "1P - 3C apres passe prealable de Nord",
			southHand: hand("AKQ32", "Q65", "K987", "K"),
			northHand: hand("9876", "AK874", "J32", "3"),
			wantOpen:  "1P",
			wantResp:  "3C",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWithQuietOpponents(north, map[int]*Hand{south: tc.southHand, north: tc.northHand})
			calls := NewEngine(d).Run()
			n := 0
			for _, sc := range calls {
				if sc.Seat != north {
					continue
				}
				n++
				if n == 1 && sc.Call.Kind != KindPass {
					t.Fatalf("North should pass as dealer, got %s\nauction: %s", sc.Call.Format("fr"), formatAuction(calls))
				}
			}
			for _, sc := range calls {
				if sc.Seat != south {
					continue
				}
				if got := sc.Call.Format("fr"); got != tc.wantOpen {
					t.Fatalf("South's opening = %s, want %s\nauction: %s", got, tc.wantOpen, formatAuction(calls))
				}
				break
			}
			n = 0
			for _, sc := range calls {
				if sc.Seat != north {
					continue
				}
				n++
				if n == 2 {
					if got := sc.Call.Format("fr"); got != tc.wantResp {
						t.Fatalf("North's answer = %s (%s), want %s\nauction: %s", got, sc.M.fr, tc.wantResp, formatAuction(calls))
					}
					if !strings.Contains(sc.M.fr, "rencontre") {
						t.Fatalf("comment %q does not mention rencontre", sc.M.fr)
					}
					return
				}
			}
			t.Fatalf("North never made a second call\nauction: %s", formatAuction(calls))
		})
	}
}

// TestRencontreMinorCappedAtFour replays the reference deal 1S-2D: North
// holds four spades and a good six-card club suit, so the double jump the
// overcall calls for would land on 5C. A rencontre in a minor stops at the
// four level (docs/regles_moteur.md [RC-2]): 4C says the same thing and
// keeps the level the camp may need.
func TestRencontreMinorCappedAtFour(t *testing.T) {
	const north, south = 0, 2
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "All"]
[Deal "S:AKJ98.QJ9.Q6.863 74.T83.AK9543.A2 QT62..872.KQJT97 53.AK76542.JT.54"]
`))
	if err != nil {
		t.Fatalf("ParsePBN: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.Seat != south {
			continue
		}
		if got := sc.Call.Format("fr"); got != "1P" {
			t.Fatalf("South's opening = %s, want 1P\nauction: %s", got, formatAuction(calls))
		}
		break
	}
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if got := sc.Call.Format("fr"); got != "4T" {
			t.Fatalf("North's answer = %s (%s), want 4T\nauction: %s", got, sc.M.fr, formatAuction(calls))
		}
		if !strings.Contains(sc.M.fr, "rencontre") {
			t.Fatalf("comment %q does not mention rencontre", sc.M.fr)
		}
		return
	}
	t.Fatalf("North never called\nauction: %s", formatAuction(calls))
}
