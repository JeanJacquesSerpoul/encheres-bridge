package engine

import (
	"strings"
	"testing"
)

// TestRubensohl checks the Rubensohl convention (docs/bidings.md, "LE
// RUBENSOHL", "Développements simplifiés") for responder over partner's
// 1SA opening after a natural suit intervention: double is positive
// (6H+, 2+ cards in the intervened suit, no 5+-card suit of their own), a
// suit above the intervention at the two level is natural and weak, the
// step that would transfer into the intervened suit itself asks for the
// other major(s) ("Texas impossible"), and any other 5+-card suit still
// transfers normally ("Texas").
//
// North (1SA, 15H) and East (natural 2H overcall, 6 hearts) are fixed
// across all cases; only South (responder) varies.
func TestRubensohl(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	northHand := hand("K32", "A3", "AK98", "J876")
	eastHand := hand("A65", "KQJT98", "54", "43")

	t.Run("Contre positif, tendance Stayman", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  eastHand,
			south: hand("Q94", "765", "QJ7", "AKT9"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "Contre" {
			t.Fatalf("South's call = %s (%s), want Contre (double)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "Rubensohl") || !strings.Contains(comment, "positif") {
			t.Fatalf("comment %q does not read as the Rubensohl positive double", comment)
		}
	})

	t.Run("Naturel faible au palier de 2", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  eastHand,
			south: hand("QJT98", "76", "7632", "52"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "2P" {
			t.Fatalf("South's call = %s (%s), want 2P (natural, weak)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "faible") {
			t.Fatalf("comment %q does not read as the natural weak overcall response", comment)
		}
	})

	t.Run("Texas impossible, chicane a Coeur", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  eastHand,
			south: hand("QJT9", "2", "QJ76", "AKQT"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "3K" {
			t.Fatalf("South's call = %s (%s), want 3K (impossible Texas)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "impossible") {
			t.Fatalf("comment %q does not read as the impossible Texas", comment)
		}
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "3SA" {
			t.Fatalf("North's answer = %s (%s), want 3SA (no major, stopper in hearts)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
	})

	t.Run("Texas normal pour une autre couleur", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  eastHand,
			south: hand("QJT98", "76", "Q76", "AK9"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "3C" {
			t.Fatalf("South's call = %s (%s), want 3C (Texas to spades)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "Pique") {
			t.Fatalf("comment %q does not mention spades", comment)
		}
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "3P" {
			t.Fatalf("North's rectification = %s (%s), want 3P\nauction: %s", northGot, northComment, formatAuction(calls))
		}
	})

	t.Run("3P demande d'arret pour 3SA (SEF 2018 p.29)", func(t *testing.T) {
		// South's own long suit is hearts too (a misfit with East's overcall):
		// no route above applies, so South asks for a stopper instead of
		// guessing 3NT.
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  eastHand,
			south: hand("QJ4", "76542", "QJ7", "AK"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "3P" {
			t.Fatalf("South's call = %s (%s), want 3P (stopper ask)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "arrêt") {
			t.Fatalf("comment %q does not read as the stopper ask", comment)
		}
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "3SA" {
			t.Fatalf("North's answer = %s (%s), want 3SA (stopper in hearts)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
	})
}

// southsCall returns the French-formatted call and its comment for the nth
// (0-indexed) call made by the given seat.
func southsCall(calls []SeatCall, seat, nth int) (string, string) {
	n := 0
	for _, sc := range calls {
		if sc.Seat != seat {
			continue
		}
		if n == nth {
			return sc.Call.Format("fr"), sc.M.fr
		}
		n++
	}
	return "", ""
}
