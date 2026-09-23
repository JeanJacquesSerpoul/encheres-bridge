package main

import (
	"strings"
	"testing"
)

// TestCompetitiveMinorMajorBeforeRaise replays a reported auction: 1D-X, West
// holds 9.Q9763.9742.AT9. The engine raised to 2D ("soutien simple en
// compétition") and the pair drifted to 3NT, the five-card heart suit never
// mentioned. Over a minor opening the major at the one level comes before any
// support [Rm-2, RC-1b]: West must bid 1H.
func TestCompetitiveMinorMajorBeforeRaise(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "NS"]
[Deal "E:AK82.J842.AQJ.K5 QJT5.AKT5.T5.Q86 9.Q9763.9742.AT9 7643..K863.J7432"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const west = 3
	for _, sc := range calls {
		if sc.Seat != west {
			continue
		}
		if got := sc.Call.Format("fr"); got != "1C" || !strings.Contains(sc.M.fr, "forcing") {
			t.Fatalf("West's first call = %s (%q), want 1C (1H, forcing)\nauction: %s",
				got, sc.M.fr, formatAuction(calls))
		}
		break
	}

	contract, _, _ := finalContract(calls)
	if contract.Strain != SHearts {
		t.Fatalf("final contract = %s, want a heart contract\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestCompetitiveMinorMajorChoice checks the choice between the majors over
// 1D-X, mirroring the uncontested rule: the longer major from five cards
// (spades on a tie), hearts with two four-card majors.
func TestCompetitiveMinorMajorChoice(t *testing.T) {
	cases := []struct {
		south *Hand
		want  string
	}{
		{hand("A9763", "K9763", "4", "32"), "1P"},
		{hand("A976", "K9763", "432", "2"), "1C"},
		{hand("A976", "K976", "432", "32"), "1C"},
		{hand("A9763", "K976", "43", "32"), "1P"},
	}
	for _, tc := range cases {
		e := &Engine{opener: 0, openCall: bid(1, SDiamonds)}
		e.ps[0] = &playerState{seat: 0, hand: &Hand{}, shownMin: 12, shownMax: 23}
		e.ps[1] = &playerState{seat: 1, hand: &Hand{}, shownMax: 40}
		e.ps[2] = &playerState{seat: 2, hand: tc.south, shownMax: 40}
		e.ps[3] = &playerState{seat: 3, hand: &Hand{}, shownMax: 40}
		e.calls = []SeatCall{
			{Seat: 0, Call: bid(1, SDiamonds)},
			{Seat: 1, Call: doubleCall},
		}
		c, mn := e.respondCompetitive(e.ps[2])
		if got := c.Format("fr"); got != tc.want {
			t.Errorf("%v: call = %s (%q), want %s", tc.south, got, mn.fr, tc.want)
		}
	}
}
