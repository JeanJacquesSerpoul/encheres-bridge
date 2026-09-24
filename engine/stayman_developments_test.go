package engine

import (
	"strings"
	"testing"
)

// TestStaymanDevelopments checks the continuations after a Stayman answer
// over a 1SA opening (docs/bidings.md, "Développements après la réponse au
// Stayman"): natural non-forcing raises and the same notrump ladder as a
// direct response; the "chassé-croisé" over a denial; a new-suit change
// forcing to game (denying a major fit, promising shortness except with
// slam values); the transfers/game-raise/slam-ambition ladder after both
// majors are shown; splinters (15-17 HLD); and the artificial "convention
// 2012" (18HLD+).
func TestStaymanDevelopments(t *testing.T) {
	const north, south = 0, 2
	northHearts := hand("K32", "AQ32", "KQJ", "542") // 15H, 3-4-3-3: answers 2H
	northDeny := hand("K32", "Q32", "AKJ4", "Q54")   // 15H, 3-3-4-3: answers 2D
	northBoth := hand("KQ32", "AJ32", "K5", "Q54")   // 15H, 4-4-2-3: answers 2SA

	// The fitted continuations follow the combined count [E-9]: a major game
	// needs 27 HLD, so facing 15-17 it is bid from 12 HLD, invited on 10-11
	// and not sought below.
	t.Run("fit apres 2C, 8 HLD : passe", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHearts,
			south: hand("987", "KJ98", "A65", "987"),
		})
		calls := NewEngine(d).Run()
		northGot, _ := southsCall(calls, north, 1)
		if northGot != "2C" {
			t.Fatalf("North's Stayman answer = %s, want 2C\nauction: %s", northGot, formatAuction(calls))
		}
		got, comment := southsCall(calls, south, 1)
		if got != "Passe" {
			t.Fatalf("South's call = %s (%s), want Passe (17 + 8 stays short of 27)\nauction: %s", got, comment, formatAuction(calls))
		}
	})

	t.Run("fit apres 2C, 10 HLD : proposition 3C", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHearts,
			south: hand("987", "KJ98", "A65", "Q87"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 1)
		if got != "3C" {
			t.Fatalf("South's call = %s (%s), want 3C (game invitation)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "proposition") {
			t.Fatalf("comment %q does not read as an invitation", comment)
		}
	})

	t.Run("convention 2012 apres 2C", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHearts,
			south: hand("65", "KJT9", "AT98", "AKQ"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 1)
		if got != "3P" {
			t.Fatalf("South's call = %s (%s), want 3P (convention 2012)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "2012") {
			t.Fatalf("comment %q does not mention the 2012 convention", comment)
		}
	})

	t.Run("splinter apres 2C, 15-17HLD", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHearts,
			south: hand("4", "KJ98", "AT98", "KQJ9"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 1)
		if got != "4P" {
			t.Fatalf("South's call = %s (%s), want 4P (splinter, singleton spade)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "Splinter") || !strings.Contains(comment, "Pique") {
			t.Fatalf("comment %q does not read as a spade splinter", comment)
		}
	})

	t.Run("nouvelle couleur 3T, pas de fit, forcing de manche", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHearts,
			south: hand("QJT9", "876", "2", "AKQ98"),
		})
		calls := NewEngine(d).Run()
		northGot, _ := southsCall(calls, north, 1)
		if northGot != "2C" {
			t.Fatalf("North's Stayman answer = %s, want 2C\nauction: %s", northGot, formatAuction(calls))
		}
		got, comment := southsCall(calls, south, 1)
		if got != "3T" {
			t.Fatalf("South's call = %s (%s), want 3T (new suit, game forcing)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "forcing de manche") || !strings.Contains(comment, "Carreau") {
			t.Fatalf("comment %q does not read as game forcing with a diamond singleton", comment)
		}
	})

	t.Run("chasse-croise apres 2K", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northDeny,
			south: hand("AQJ98", "KJT9", "2", "876"),
		})
		calls := NewEngine(d).Run()
		northGot, _ := southsCall(calls, north, 1)
		if northGot != "2K" {
			t.Fatalf("North's Stayman answer = %s, want 2K (denies both majors)\nauction: %s", northGot, formatAuction(calls))
		}
		got, comment := southsCall(calls, south, 1)
		if got != "3C" {
			t.Fatalf("South's call = %s (%s), want 3C (chassé-croisé)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "chassé-croisé") {
			t.Fatalf("comment %q does not mention the chassé-croisé", comment)
		}
	})

	t.Run("transfert apres 2SA (les deux majeures)", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northBoth,
			south: hand("J987", "98", "AJ76", "K98"),
		})
		calls := NewEngine(d).Run()
		northGot, _ := southsCall(calls, north, 1)
		if northGot != "2SA" {
			t.Fatalf("North's Stayman answer = %s, want 2SA (both majors)\nauction: %s", northGot, formatAuction(calls))
		}
		got, comment := southsCall(calls, south, 1)
		if got != "3K" {
			t.Fatalf("South's call = %s (%s), want 3K (transfer to spades)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "transfert") {
			t.Fatalf("comment %q does not read as the transfer", comment)
		}
		northRect, northComment := southsCall(calls, north, 2)
		if northRect != "3P" {
			t.Fatalf("North's rectification = %s (%s), want 3P\nauction: %s", northRect, northComment, formatAuction(calls))
		}
	})
}
