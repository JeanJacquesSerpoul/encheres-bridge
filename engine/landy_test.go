package engine

import (
	"strings"
	"testing"
)

// TestLandy checks the Landy 2C intervention [I-3b]: the player seated right
// behind the 1NT opener shows, in one cheap bid, a major two-suiter of at
// least 5-4 worth about ten points. The convention only exists in that seat
// and over that opening, and the advance names the longer major -- hearts at
// equal length, so the hand with five spades can still correct at the two
// level.
func TestLandy(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	// 17H regular: the 1SA opening is the same in every case below.
	northHand := hand("J3", "K62", "KQ84", "AKJ2")

	t.Run("Bicolore majeur 5-4 : 2T Landy", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  hand("AKQ54", "AJ73", "72", "83"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, east, 0)
		if got != "2T" {
			t.Fatalf("East's call = %s (%s), want 2T (Landy)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "Landy") || !strings.Contains(comment, "bicolore majeur") {
			t.Fatalf("comment %q does not read as the Landy overcall", comment)
		}
	})

	t.Run("4-4 dans les majeures : pas de Landy", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  hand("AQ54", "AJ73", "Q72", "83"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, east, 0)
		if got == "2T" {
			t.Fatalf("East bid the Landy 2T (%s) on a 4-4 two-suiter\nauction: %s", comment, formatAuction(calls))
		}
	})

	t.Run("5-4 trop faible : pas de Landy", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: northHand,
			east:  hand("Q7543", "9732", "72", "83"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, east, 0)
		if got == "2T" {
			t.Fatalf("East bid the Landy 2T (%s) on a hand short of ten points\nauction: %s", comment, formatAuction(calls))
		}
	})

	t.Run("L'avancee nomme la majeure la plus longue", func(t *testing.T) {
		d := dealWith(north, map[int]*Hand{
			north: northHand,
			east:  hand("AKQ54", "AJ73", "72", "83"),
			south: hand("62", "QT98", "AT9", "QT94"),
			west:  hand("T987", "54", "J653", "765"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, west, 0)
		if got != "2P" {
			t.Fatalf("West's advance = %s (%s), want 2P (the longer major)\nauction: %s", got, comment, formatAuction(calls))
		}
	})

	t.Run("A egalite, 2C puis rectification a 2P", func(t *testing.T) {
		d := dealWith(north, map[int]*Hand{
			north: northHand,
			east:  hand("AKQ54", "AJ73", "72", "83"),
			south: hand("762", "Q98", "AT9", "QT94"),
			west:  hand("T98", "T54", "J653", "765"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, west, 0)
		if got != "2C" {
			t.Fatalf("West's advance = %s (%s), want 2C (hearts at equal length)\nauction: %s", got, comment, formatAuction(calls))
		}
		eastGot, eastComment := southsCall(calls, east, 1)
		if eastGot != "2P" {
			t.Fatalf("East's follow-up = %s (%s), want 2P (correction with five spades)\nauction: %s",
				eastGot, eastComment, formatAuction(calls))
		}
		if !strings.Contains(eastComment, "rectification") {
			t.Fatalf("comment %q does not read as the correction to spades", eastComment)
		}
	})

	t.Run("Aucune majeure chez l'avancee : 2T se joue tel quel", func(t *testing.T) {
		d := dealWith(north, map[int]*Hand{
			north: northHand,
			east:  hand("AKQ54", "AJ73", "72", "83"),
			south: hand("T762", "QT98", "AT9", "54"),
			west:  hand("98", "54", "J653", "QT976"),
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, west, 0)
		if got != "Passe" {
			t.Fatalf("West's advance = %s (%s), want Passe (no major, real clubs)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "Trèfle") {
			t.Fatalf("comment %q does not explain the pass on the club length", comment)
		}
	})

	t.Run("Le Landy n'existe que juste apres l'ouvreur", func(t *testing.T) {
		d := dealWith(north, map[int]*Hand{
			north: northHand,
			east:  hand("K98", "QJ8", "J65", "T976"),
			south: hand("762", "T54", "AT93", "Q54"),
			west:  hand("AQT54", "A973", "72", "83"),
		})
		calls := NewEngine(d).Run()
		if got, comment := southsCall(calls, east, 0); got != "Passe" {
			t.Fatalf("East's call = %s (%s), want Passe\nauction: %s", got, comment, formatAuction(calls))
		}
		if got, comment := southsCall(calls, south, 0); got != "Passe" {
			t.Fatalf("South's call = %s (%s), want Passe\nauction: %s", got, comment, formatAuction(calls))
		}
		got, comment := southsCall(calls, west, 0)
		if got == "2T" {
			t.Fatalf("West bid the Landy 2T (%s) from the balancing seat\nauction: %s", comment, formatAuction(calls))
		}
	})
}
