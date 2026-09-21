package main

import (
	"strings"
	"testing"
)

// TestDisambiguatePair checks the pure disambiguation rule used to resolve
// an ambiguous keycard/king-key step (each grouping two possible counts into
// one bid): assume the least favorable count unless the asker's own count
// makes the favorable one mathematically impossible, or the partnership
// already disclosed (via a prior control bid) more than the unfavorable
// count allows.
func TestDisambiguatePair(t *testing.T) {
	cases := []struct {
		name                        string
		lo, hi, own, disclosed, max int
		want                        int
	}{
		{"no evidence either way: assume the worst", 0, 3, 1, 0, 5, 0},
		{"own count rules out the favorable reading", 0, 3, 4, 0, 5, 0},
		{"prior disclosure rules out the unfavorable reading", 0, 3, 1, 1, 5, 3},
		{"1-or-4 step, no evidence: assume the worst", 1, 4, 1, 0, 5, 1},
		{"1-or-4 step, own count forces the low reading", 1, 4, 2, 0, 5, 1},
		{"1-or-4 step, disclosure forces the high reading", 1, 4, 0, 2, 5, 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := disambiguatePair(c.lo, c.hi, c.own, c.disclosed, c.max); got != c.want {
				t.Fatalf("disambiguatePair(%d,%d,own=%d,disclosed=%d,max=%d) = %d, want %d",
					c.lo, c.hi, c.own, c.disclosed, c.max, got, c.want)
			}
		})
	}
}

// TestBlackwoodAssumesWorstCaseOnAmbiguousStep is an end-to-end regression
// test for a reported case: East asks Blackwood, West's step is the
// ambiguous "0 or 3 keycards", and nothing in the auction rules out 0 (East
// holds only one keycard itself, and no control bid disclosed anything about
// West's hand beforehand). West's true keycards happen to be 3, but a real
// asker cannot tell that from the auction alone and must not gamble on the
// favorable reading -- East must sign off at the five level, not bid slam.
func TestBlackwoodAssumesWorstCaseOnAmbiguousStep(t *testing.T) {
	const east, west = 1, 3

	eastH := hand("A432", "432", "43", "432") // 1 keycard: the spade ace
	westH := hand("2", "AK543", "AKQ2", "K5") // trump=hearts: true keycards = 3

	e := &Engine{}
	e.ps[east] = &playerState{seat: east, hand: eastH, shownMax: 40}
	e.ps[west] = &playerState{seat: west, hand: westH, shownMax: 40}
	e.bw[sideOf(east)] = bwState{asked: true, asker: east, trump: Hearts}
	e.calls = []SeatCall{{Seat: east, Call: bid(4, SNoTrump)}}

	answerCall, answerMn := e.keycardAnswer(e.ps[west], Hearts)
	if !strings.Contains(answerMn.fr, "0 ou 3") {
		t.Fatalf("West's answer %q is not the ambiguous 0-or-3 step", answerMn.fr)
	}
	e.record(west, answerCall, answerMn)

	c, mn := e.afterKeycards(e.ps[east])
	if c.higherThan(bidSuit(5, Hearts)) || strings.Contains(mn.fr, "chelem") {
		t.Fatalf("East continues %s (%s), want a sign-off at the five level rather than a gamble on the favorable reading",
			c.Format("fr"), mn.fr)
	}
}
