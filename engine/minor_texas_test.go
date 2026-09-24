package engine

import (
	"strings"
	"testing"
)

// TestMinorTexas checks the minor-suit Texas over a 1SA opening
// (docs/bidings.md, "TEXAS MINEURS", "théorie du singleton"): 2S transfers
// to clubs (with two possible opener answers: 2SA declines with a good fit,
// else the plain rectification to 3C), 3C transfers to diamonds (mandatory
// rectification to 3D, no decline option). A weak hand (<=7HL) just signs
// off, correcting a declined club Texas back to 3C. A game-going hand
// (10HL+) with a genuine singleton/void announces it next: a major singleton
// is named naturally (3H = short hearts, 3S = short spades), 3NT shows a
// short in the other minor, and 3D a 5-5 clubs-diamonds two-suiter.
func TestMinorTexas(t *testing.T) {
	const north, east, south = 0, 1, 2
	// North has a 4-card club suit: every club Texas below sees the "good
	// fit" decline (2SA) rather than the plain rectification (3C).
	northHand := hand("K32", "A32", "KQ8", "A543")

	t.Run("Texas Trefle faible, refus du texas, correction a 3T", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			south: hand("92", "876", "96", "QJT862"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "2P" {
			t.Fatalf("South's call = %s (%s), want 2P (club Texas)\nauction: %s", got, comment, formatAuction(calls))
		}
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "2SA" {
			t.Fatalf("North's answer = %s (%s), want 2SA (good fit, declines)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "3T" {
			t.Fatalf("South's correction = %s (%s), want 3T (absolute sign-off)\nauction: %s", southGot, southComment, formatAuction(calls))
		}
		if !strings.Contains(southComment, "arrêt") {
			t.Fatalf("comment %q does not read as the sign-off", southComment)
		}
	})

	t.Run("Texas Trefle faible, transformation simple", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: hand("KQ2", "AQ32", "KQ8", "754"), // 16H, no club fit: 3T is mandatory
			south: hand("93", "876", "96", "QJT862"),
		})
		calls := NewEngine(d).Run()
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "3T" {
			t.Fatalf("North's answer = %s (%s), want 3T (plain rectification)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "Passe" {
			t.Fatalf("South's call = %s (%s), want Passe\nauction: %s", southGot, southComment, formatAuction(calls))
		}
	})

	t.Run("Texas Carreau faible, rectification obligatoire", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			south: hand("864", "985", "T97654", "2"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "3T" {
			t.Fatalf("South's call = %s (%s), want 3T (diamond Texas)\nauction: %s", got, comment, formatAuction(calls))
		}
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "3K" {
			t.Fatalf("North's answer = %s (%s), want 3K (mandatory rectification)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "Passe" {
			t.Fatalf("South's call = %s (%s), want Passe\nauction: %s", southGot, southComment, formatAuction(calls))
		}
	})

	t.Run("Manche, singleton Pique annonce naturellement par 3P", func(t *testing.T) {
		// East is pinned: left to the filler it would draw both majors
		// five-five and open the Landy 2C intervention [I-3b], which is
		// correct bridge but not what this test is about.
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			south: hand("Q", "A32", "T97", "KJ9876"),
			east:  hand("A987", "KJ54", "J65", "Q2"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "2P" {
			t.Fatalf("South's call = %s (%s), want 2P (club Texas)\nauction: %s", got, comment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "3P" {
			t.Fatalf("South's announcement = %s (%s), want 3P (singleton spade, named naturally)\nauction: %s", southGot, southComment, formatAuction(calls))
		}
		if !strings.Contains(southComment, "Pique") {
			t.Fatalf("comment %q does not mention spades", southComment)
		}
	})

	t.Run("Manche, singleton Coeur annonce naturellement par 3C", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			south: hand("A32", "Q", "T97", "KJ9876"),
			east:  hand("J987", "KJ54", "A65", "Q2"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "2P" {
			t.Fatalf("South's call = %s (%s), want 2P (club Texas)\nauction: %s", got, comment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "3C" {
			t.Fatalf("South's announcement = %s (%s), want 3C (singleton heart, named naturally)\nauction: %s", southGot, southComment, formatAuction(calls))
		}
		if !strings.Contains(southComment, "Cœur") {
			t.Fatalf("comment %q does not mention hearts", southComment)
		}
	})

	t.Run("Manche, bicolore 5T-5K", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			south: hand("9", "84", "AT754", "KJ986"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "2P" {
			t.Fatalf("South's call = %s (%s), want 2P (club Texas)\nauction: %s", got, comment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "3K" {
			t.Fatalf("South's announcement = %s (%s), want 3K (5-5 two-suiter)\nauction: %s", southGot, southComment, formatAuction(calls))
		}
		if !strings.Contains(southComment, "bicolore") {
			t.Fatalf("comment %q does not mention the two-suiter", southComment)
		}
	})
}

// TestMinorTexasMiddleZone pins the strength boundary of [N-4]. The club
// Texas is a weak hand's escape (HL <= 7, transfer and pass out in 3C) or a
// game-going one's shortness (10 HL and up); in between, the six-card minor
// takes the notrump ladder [N-6] whatever the shape, singleton included --
// eight or nine points opposite 15-17 cannot reach game, and 1NT asks seven
// tricks where 3C asks nine. Only the strength moves the line: the same shape
// two points lighter does transfer.
func TestMinorTexasMiddleZone(t *testing.T) {
	const north, south = 0, 2
	opener := hand("QJ965", "Q94", "A87", "AQ") // 15H, régulière
	cases := []struct {
		name      string
		responder *Hand
		want      string
	}{
		{
			name:      "8 HL avec un singleton : la zone intermédiaire passe",
			responder: hand("T83", "K76", "5", "K97632"), // 6H + 2L
			want:      "Passe",
		},
		{
			name:      "la même forme à 7 HL : la fuite se fait",
			responder: hand("T83", "Q76", "5", "K97632"), // 5H + 2L
			want:      "2P",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWithQuietOpponents(north, map[int]*Hand{north: opener, south: tc.responder})
			calls := NewEngine(d).Run()
			got, comment := southsCall(calls, south, 0)
			if got != tc.want {
				t.Fatalf("responder's call = %s (%s), want %s\nauction: %s", got, comment, tc.want, formatAuction(calls))
			}
		})
	}
}
