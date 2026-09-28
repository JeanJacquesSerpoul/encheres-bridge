package engine

import "testing"

// TestGameInFitShowsTrumpControl: once the control exchange is under way, the
// game in the fit held with the trump ace or king is itself a control -- the
// trump one -- and partner reads it inside the exchange rather than as a
// request to play there. 1H - 1S - 3D - 3NT - 4C - 4D - 4H: South (2 AKJ743
// KQ42 K7) shows the trump ace; North (Q874 62 AT9 AJ64), with no spade
// control, stops there and says so.
func TestGameInFitShowsTrumpControl(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(east, map[int]*Hand{
		north: hand("Q874", "62", "AT9", "AJ64"),
		east:  hand("AJ53", "Q", "763", "QT953"),
		south: hand("2", "AKJ743", "KQ42", "K7"),
		west:  hand("KT96", "T985", "J85", "82"),
	})

	calls := NewEngine(d).Run()

	var game, answer *SeatCall
	for i := range calls {
		if calls[i].Seat == south && calls[i].Call.Format("fr") == "4C" {
			game = &calls[i]
			if i+2 < len(calls) {
				answer = &calls[i+2]
			}
		}
	}
	if game == nil || !game.M.controlBid || game.M.controlSuit != Hearts {
		t.Fatalf("South's 4H should show the trump control\nauction: %s", formatAuction(calls))
	}
	if answer == nil || answer.Seat != north || answer.Call.Kind != KindPass || answer.M.fr == "" ||
		answer.M.fr == "l'enchère du partenaire convient, rien à ajouter" {
		t.Fatalf("North should answer 4H inside the control exchange\nauction: %s", formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract.Format("fr") != "4C" {
		t.Fatalf("final contract = %s, want 4C\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}
