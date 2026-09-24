package engine

import (
	"strings"
	"testing"
)

// TestBothMajors4D checks the 4D response to a 1NT opening: 5-5 in the
// majors with 9 H and more. Opener, being regular, always holds three cards
// at least in one major, so he picks it and the fit is of eight cards at
// least; on equal length the honours decide. Below 9 H the ordinary ladder
// still applies -- Stayman from 8 H [N-3].
func TestBothMajors4D(t *testing.T) {
	const north, south = 0, 2
	responder := hand("KT976", "AQ432", "7", "84") // 9 H, 5-5 majeur

	t.Run("5-5 majeur, 9 H : 4K puis le choix de l'ouvreur", func(t *testing.T) {
		// 3-3 in the majors: the honours decide, and AD sec beats R76.
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: hand("AQ4", "K76", "K652", "QJ3"), // 15 H, régulière
			south: responder,
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "4K" {
			t.Fatalf("South's call = %s (%s), want 4K (5-5 major two-suiter)\nauction: %s", got, comment, formatAuction(calls))
		}
		if !strings.Contains(comment, "bicolore majeur 5-5") {
			t.Fatalf("comment %q does not read as the major two-suiter", comment)
		}
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "4P" {
			t.Fatalf("North's answer = %s (%s), want 4P (better major)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
		southGot, southComment := southsCall(calls, south, 1)
		if southGot != "Passe" {
			t.Fatalf("South's call = %s (%s), want Passe (the major is chosen)\nauction: %s", southGot, southComment, formatAuction(calls))
		}
	})

	t.Run("5-5 majeur : l'ouvreur prend sa majeure la plus longue", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: hand("A4", "KJ76", "A652", "QJ3"), // 15 H, 4 ♥ contre 2 ♠
			south: responder,
		})
		calls := NewEngine(d).Run()
		northGot, northComment := southsCall(calls, north, 1)
		if northGot != "4C" {
			t.Fatalf("North's answer = %s (%s), want 4C (longer major)\nauction: %s", northGot, northComment, formatAuction(calls))
		}
	})

	t.Run("5-5 majeur sous le seuil : l'echelle ordinaire", func(t *testing.T) {
		d := dealWithQuietOpponents(north, map[int]*Hand{
			north: hand("AQ4", "K76", "K652", "QJ3"),
			south: hand("KT976", "AJ432", "7", "84"), // 8 H : Stayman, pas 4♦
		})
		calls := NewEngine(d).Run()
		got, comment := southsCall(calls, south, 0)
		if got != "2T" {
			t.Fatalf("South's call = %s (%s), want 2T (Stayman)\nauction: %s", got, comment, formatAuction(calls))
		}
	})
}
