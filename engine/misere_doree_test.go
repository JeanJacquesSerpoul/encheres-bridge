package engine

import (
	"strings"
	"testing"
)

// TestMisereDoree checks the "misère dorée" (Bridgeur n°813, J-P. Desmoulins,
// docs/addon_11.md): a limited hand (7-8H) with a genuine singleton and a
// five-card major, routed through Stayman instead of Texas over a 1NT
// opening so a nine-card fit can be found before committing to game. The
// worked examples all use South = A Q T 7 4 . 2 . J 8 6 3 . T 9 2 (7H,
// singleton heart, five spades) or, for the reversed case, South = T 9 3 .
// A Q J T 6 . T 8 4 2 . 2 (7H, singleton club, five hearts).
func TestMisereDoree(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3

	southSpades := hand("AQT74", "2", "J863", "T98")
	southHearts := hand("T93", "AQJT6", "T842", "2")

	seatCall := func(t *testing.T, calls []SeatCall, seat int, n int) SeatCall {
		count := 0
		for _, sc := range calls {
			if sc.Seat != seat {
				continue
			}
			count++
			if count == n {
				return sc
			}
		}
		t.Fatalf("seat %d never made call #%d\nauction: %s", seat, n, formatAuction(calls))
		return SeatCall{}
	}

	t.Run("l'ouvreur montre la même majeure : bondit à la manche", func(t *testing.T) {
		// The article's own example: North = J853.AJ8.KQ5.AJ4 (16H, 4 spades).
		north1 := hand("J853", "AJ8", "KQ5", "AJ4")
		d := dealWith(north, map[int]*Hand{north: north1, south: southSpades})
		calls := NewEngine(d).Run()

		ask := seatCall(t, calls, south, 1)
		if got := ask.Call.Format("fr"); got != "2T" {
			t.Fatalf("South's Stayman = %s (%s), want 2T\nauction: %s", got, ask.M.fr, formatAuction(calls))
		}
		jump := seatCall(t, calls, south, 2)
		if got := jump.Call.Format("fr"); got != "4P" {
			t.Fatalf("South's conclusion = %s (%s), want 4P (fit neuvième)\nauction: %s", got, jump.M.fr, formatAuction(calls))
		}
		if !strings.Contains(jump.M.fr, "fit neuvième") {
			t.Fatalf("comment %q does not mention the nine-card fit", jump.M.fr)
		}
		contract, _, _ := finalContract(calls)
		if !(contract.Level == 4 && contract.Strain == SSpades) {
			t.Fatalf("final contract = %s, want 4P\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})

	t.Run("l'ouvreur dénie la majeure : misère dorée directe, acceptée", func(t *testing.T) {
		north2 := hand("K92", "KJ94", "A74", "AQ2") // 17H maximum, 3-card spade fit
		d := dealWith(north, map[int]*Hand{north: north2, south: southSpades})
		calls := NewEngine(d).Run()

		show := seatCall(t, calls, south, 2)
		if got := show.Call.Format("fr"); got != "2P" {
			t.Fatalf("South's misère dorée = %s (%s), want 2P\nauction: %s", got, show.M.fr, formatAuction(calls))
		}
		if !strings.Contains(show.M.fr, "misère dorée") {
			t.Fatalf("comment %q does not mention the misère dorée", show.M.fr)
		}
		contract, _, _ := finalContract(calls)
		if !(contract.Level == 4 && contract.Strain == SSpades) {
			t.Fatalf("final contract = %s, want 4P (maximum, fit)\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})

	t.Run("l'ouvreur dénie la majeure : misère dorée directe, refusée", func(t *testing.T) {
		north3 := hand("98", "KJ94", "A74", "AQJ2") // 15H minimum, no spade fit
		d := dealWith(north, map[int]*Hand{north: north3, south: southSpades})
		calls := NewEngine(d).Run()

		contract, _, _ := finalContract(calls)
		if !(contract.Level == 2 && contract.Strain == SSpades) {
			t.Fatalf("final contract = %s, want 2P (minimum, declines)\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})

	t.Run("majeures inversées : 2SA puis fit confirmé, conclusion à la manche", func(t *testing.T) {
		// The article's exact worked auction: 1SA-2C-2S-2SA-3H-4H.
		north4 := hand("KQJ4", "K85", "AJ5", "K54") // 17H, 4 spades, 3 hearts
		d := dealWith(north, map[int]*Hand{north: north4, south: southHearts,
			west: hand("A876", "97", "K96", "QT83"),
			east: hand("52", "432", "Q73", "AJ976")})
		calls := NewEngine(d).Run()

		ask := seatCall(t, calls, south, 2)
		if got := ask.Call.Format("fr"); got != "2SA" {
			t.Fatalf("South's reversed ask = %s (%s), want 2SA\nauction: %s", got, ask.M.fr, formatAuction(calls))
		}
		fit := seatCall(t, calls, north, 3)
		if got := fit.Call.Format("fr"); got != "3C" {
			t.Fatalf("North's fit confirmation = %s (%s), want 3C\nauction: %s", got, fit.M.fr, formatAuction(calls))
		}
		if !strings.Contains(fit.M.fr, "trois cartes") {
			t.Fatalf("comment %q does not mention the three-card fit", fit.M.fr)
		}
		conclusion := seatCall(t, calls, south, 3)
		if got := conclusion.Call.Format("fr"); got != "4C" {
			t.Fatalf("South's conclusion = %s (%s), want 4C\nauction: %s", got, conclusion.M.fr, formatAuction(calls))
		}
		contract, _, _ := finalContract(calls)
		if !(contract.Level == 4 && contract.Strain == SHearts) {
			t.Fatalf("final contract = %s, want 4C\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})

	t.Run("l'ouvreur montre les deux majeures : Texas pour le fit connu", func(t *testing.T) {
		north5 := hand("KJ92", "AQ54", "K75", "Q5") // 15H, 4-4 majors
		d := dealWith(north, map[int]*Hand{north: north5, south: southSpades})
		calls := NewEngine(d).Run()

		texas := seatCall(t, calls, south, 2)
		if got := texas.Call.Format("fr"); got != "4K" {
			t.Fatalf("South's Texas = %s (%s), want 4K\nauction: %s", got, texas.M.fr, formatAuction(calls))
		}
		contract, _, _ := finalContract(calls)
		if !(contract.Level == 4 && contract.Strain == SSpades) {
			t.Fatalf("final contract = %s, want 4P\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})
}
