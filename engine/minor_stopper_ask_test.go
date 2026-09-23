package engine

import (
	"strings"
	"testing"
)

// TestMinorStopperAsk checks the stopper-ask convention after opener's plain
// or jump repeat of the opening minor over a 1NT response (1m - 1SA - 2m/3m,
// docs/addon_10.md): a new major there cannot be natural (with real length
// responder would have shown it directly over the opening instead of routing
// through 1NT), so lacking a stopper in a major, responder asks for one at
// the same level as the rebid, never as a jump. Opener answers with notrump
// (holding the stopper) or a denial back to the minor; once denied,
// responder judges pass, game or slam from the combined count alone.
//
// South opens 1D (13H, six diamonds, too shapely to be balanced). North
// answers 1NT (6-10H, no four-card major, fewer than four diamonds). South's
// plain 2D repeat (13-16H) leaves North, at the top of its range with no
// heart stopper (but a spade stopper), asking 2H rather than guessing
// notrump.
func TestMinorStopperAsk(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3

	south13 := hand("54", "83", "AKJ962", "KQ4")        // 13H, 6D, no heart stopper
	south13stopped := hand("54", "K3", "AKJ962", "Q42") // 13H, 6D, heart stopper (Kx)
	north8 := hand("KJ7", "976", "853", "A963")         // 8H (top of 1SA range), spade stopper, no heart stopper
	northGame := hand("KJ", "76", "854", "A98763")      // 8H, 12 HLD via shortness: the best the 1SA response can hold

	seatCall := func(calls []SeatCall, seat int, n int) SeatCall {
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

	t.Run("demande d'arrêt puis dénégation, pas assez pour la manche", func(t *testing.T) {
		d := dealWith(south, map[int]*Hand{south: south13, north: north8,
			west: hand("A986", "KJ54", "T7", "J82"),
			east: hand("QT32", "AQT2", "Q4", "T75")})
		calls := NewEngine(d).Run()

		ask := seatCall(calls, north, 2) // North's second call: the 1NT response is its first
		if got := ask.Call.Format("fr"); got != "2C" {
			t.Fatalf("North's ask = %s (%s), want 2C\nauction: %s", got, ask.M.fr, formatAuction(calls))
		}
		if !strings.Contains(ask.M.fr, "arrêt") {
			t.Fatalf("comment %q does not mention the stopper ask", ask.M.fr)
		}

		deny := seatCall(calls, south, 3) // South's third call: opening, then the 2D rebid, then the denial
		if got := deny.Call.Format("fr"); got != "3K" {
			t.Fatalf("South's denial = %s (%s), want 3K\nauction: %s", got, deny.M.fr, formatAuction(calls))
		}
		if !strings.Contains(deny.M.fr, "pas d'arrêt") {
			t.Fatalf("comment %q does not mention the denial", deny.M.fr)
		}

		final := seatCall(calls, north, 3)
		if final.Call.Kind != KindPass {
			t.Fatalf("North's final call = %s (%s), want Passe\nauction: %s", final.Call.Format("fr"), final.M.fr, formatAuction(calls))
		}

		contract, _, _ := finalContract(calls)
		if !(contract.Level == 3 && contract.Strain == SDiamonds) {
			t.Fatalf("final contract = %s, want 3K\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})

	t.Run("demande d'arrêt confirmée : Sans-Atout", func(t *testing.T) {
		d := dealWith(south, map[int]*Hand{south: south13stopped, north: north8,
			west: hand("A986", "J854", "T7", "K85"),
			east: hand("QT32", "AQT2", "Q4", "JT7")})
		calls := NewEngine(d).Run()

		confirm := seatCall(calls, south, 3)
		if got := confirm.Call.Format("fr"); got != "2SA" {
			t.Fatalf("South's answer = %s (%s), want 2SA\nauction: %s", got, confirm.M.fr, formatAuction(calls))
		}
		if !strings.Contains(confirm.M.fr, "arrêt à Cœur") {
			t.Fatalf("comment %q does not mention the confirmed heart stopper", confirm.M.fr)
		}

		contract, declarer, _ := finalContract(calls)
		if !(contract.Level == 2 && contract.Strain == SNoTrump && declarer == north) {
			t.Fatalf("final contract = %s by %d, want 2SA by North (first to name notrump)\nauction: %s", contract.Format("fr"), declarer, formatAuction(calls))
		}
	})

	// [C-10] settles the contract on the combined count alone, and the game
	// it names is five of the minor -- eleven tricks, 30 HLD [E-9]. North is
	// as good as the 1SA response allows (8H, 12 HLD once the fit lends it
	// two doubletons) and South has repeated a 13-16 hand: 25 combined, five
	// short. The auction stops in the partscore, which is exactly what the
	// two hands take -- nine tricks, not eleven.
	t.Run("demande d'arrêt puis dénégation : la manche mineure demande 30", func(t *testing.T) {
		d := dealWith(south, map[int]*Hand{south: south13, north: northGame,
			west: hand("A987", "K542", "T73", "J5"),
			east: hand("QT632", "AQJ9", "Q", "T2")})
		calls := NewEngine(d).Run()

		deny := seatCall(calls, south, 3)
		if got := deny.Call.Format("fr"); got != "3K" {
			t.Fatalf("South's denial = %s (%s), want 3K\nauction: %s", got, deny.M.fr, formatAuction(calls))
		}

		for _, sc := range calls {
			if sc.Call == bid(5, SDiamonds) {
				t.Fatalf("the minor game was bid on 25 combined HLD\nauction: %s", formatAuction(calls))
			}
		}
		contract, _, _ := finalContract(calls)
		if !(contract.Level == 3 && contract.Strain == SDiamonds) {
			t.Fatalf("final contract = %s, want 3K\nauction: %s", contract.Format("fr"), formatAuction(calls))
		}
	})
}
