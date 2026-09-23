package engine

import (
	"strings"
	"testing"
)

// TestHelpSuitTryLeavesRoomToDecline is the reported deal. West opens 1H, North
// overcalls 2C, East raises to 2H and South raises the clubs to 3C. West then
// tried game with 3S -- a "help suit" named above three of the trump, where a
// partner without the help has no sign-off left: East passed and the pair
// played 3S, a contract in West's weakest suit that nobody had agreed.
//
// A try is a question, and the answer must be able to be "no". The suit named
// must leave three of the trump above it; here it cannot, so there is no try
// to make.
func TestHelpSuitTryLeavesRoomToDecline(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "None"]
[Deal "S:AKT.72.Q8752.J96 J95.AJT843.AT9.K Q2.K9.J43.AQT743 87643.Q65.K6.852"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if sc.M.helpSuitTry && !bidSuit(3, Hearts).higherThan(sc.Call) {
			t.Fatalf("%s tried game with %s, above the sign-off in the fit\nauction: %s",
				seatNames[sc.Seat], sc.Call.Format("fr"), formatAuction(calls))
		}
	}
	contract, declarer, _ := finalContract(calls)
	if contract.Strain != SHearts || sideOf(declarer) != sideOf(west) {
		t.Fatalf("final contract = %s by %s, want a heart contract by East-West\nauction: %s",
			contract.Format("fr"), seatNames[declarer], formatAuction(calls))
	}
}

// TestSingletonHonourInTheirSuitIsNotWorthShortness is the same deal read from
// the point count. West holds a singleton king of clubs in the suit North bid
// and South raised: the ace sits over it, the king falls under it, and the
// shortness credit HLD grants on top of the honour is a value the auction has
// already denied. Counting it made West a 17 HLD hand -- the game-try zone --
// out of a hand worth 15, which is what produced the try in the first place.
func TestSingletonHonourInTheirSuitIsNotWorthShortness(t *testing.T) {
	const west = 3
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "None"]
[Deal "S:AKT.72.Q8752.J96 J95.AJT843.AT9.K Q2.K9.J43.AQT743 87643.Q65.K6.852"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	e := NewEngine(d)
	calls := e.Run()

	if got, want := e.ps[west].hand.HLD(Hearts), 17; got != want {
		t.Fatalf("raw HLD = %d, want %d (the test's premise)", got, want)
	}
	if got, want := e.hldAgainstTheirBidding(e.ps[west], Hearts), 15; got != want {
		t.Fatalf("valuation against their bidding = %d, want %d", got, want)
	}
	for _, sc := range calls {
		if sc.Seat == west && strings.Contains(sc.M.fr, "essai") {
			t.Fatalf("West still made a game try (%s) on a hand worth 15\nauction: %s",
				sc.M.fr, formatAuction(calls))
		}
	}
}
