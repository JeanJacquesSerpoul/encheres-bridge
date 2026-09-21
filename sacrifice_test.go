package main

import (
	"strings"
	"testing"
)

// TestScoringTable pins the scoring helpers to the classic duplicate table:
// contract values (made exactly, undoubled) and doubled undertricks, both
// sides of the vulnerability.
func TestScoringTable(t *testing.T) {
	scores := []struct {
		c    Call
		vul  bool
		want int
	}{
		{bidSuit(1, Clubs), false, 70},
		{bidSuit(2, Spades), false, 110},
		{bid(3, SNoTrump), false, 400},
		{bid(3, SNoTrump), true, 600},
		{bidSuit(4, Spades), false, 420},
		{bidSuit(4, Spades), true, 620},
		{bidSuit(5, Hearts), false, 450},
		{bidSuit(5, Clubs), true, 600},
		{bidSuit(6, Hearts), false, 980},
		{bid(7, SNoTrump), true, 2220},
	}
	for _, tc := range scores {
		if got := contractScore(tc.c, tc.vul); got != tc.want {
			t.Errorf("contractScore(%s, vul=%v) = %d, want %d", tc.c.Format("fr"), tc.vul, got, tc.want)
		}
	}
	penalties := []struct {
		down int
		vul  bool
		want int
	}{
		{1, false, 100}, {2, false, 300}, {3, false, 500}, {4, false, 800}, {5, false, 1100},
		{1, true, 200}, {2, true, 500}, {3, true, 800}, {4, true, 1100},
	}
	for _, tc := range penalties {
		if got := doubledPenalty(tc.down, tc.vul); got != tc.want {
			t.Errorf("doubledPenalty(%d, vul=%v) = %d, want %d", tc.down, tc.vul, got, tc.want)
		}
	}
}

// TestCompetitiveSacrificeAndRaise replays the requested scenario: North-South
// (vulnerable) reach 4S (620 if it makes); West, with a ten-card heart fit
// and a weak hand, sacrifices at 5H non-vulnerable -- the law of total tricks
// expects ten tricks, so one down doubled (100) costs far less than 620.
// North, whose combined count covers the five level (game threshold plus
// three points for the extra level), then bids the probable 5S over the
// sacrifice ("NS peut surenchérir si le contrat 5P est probable").
func TestCompetitiveSacrificeAndRaise(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AKJ87", "54", "KJ2", "K54"), // 15H, opens 1SA
		east:  hand("3", "KQJT97", "A54", "Q96"), // 12H, six hearts
		south: hand("QT94", "A", "QT98", "A873"), // 12H, four spades
		west:  hand("652", "8632", "763", "JT2"), // 1H, four hearts
	})
	d.Vul = [2]bool{true, false} // NS vulnerable, EO not
	calls := NewEngine(d).Run()

	sawSacrifice, sawRaise := false, false
	for _, sc := range calls {
		if sc.Seat == west && sc.Call == bid(5, SHearts) {
			sawSacrifice = true
			if !strings.HasPrefix(sc.M.fr, "sacrifice") {
				t.Fatalf("West's 5C meaning = %q, want a sacrifice explanation", sc.M.fr)
			}
		}
		if sc.Seat == north && sc.Call == bid(5, SSpades) {
			sawRaise = true
		}
	}
	if !sawSacrifice {
		t.Fatalf("West never sacrificed at 5C\nauction: %s", formatAuction(calls))
	}
	if !sawRaise {
		t.Fatalf("North never raised to 5P over the sacrifice\nauction: %s", formatAuction(calls))
	}
	final, _, _ := finalContract(calls)
	if final.Format("fr") != "5P" {
		t.Fatalf("final contract = %s, want 5P\nauction: %s", final.Format("fr"), formatAuction(calls))
	}
}

// TestSacrificeVulnerabilityGate checks that the very same deal flips on
// vulnerability alone: with a nine-card fit the law expects two down at 5H.
// Doubled, that costs 300 non-vulnerable -- cheaper than the 620 of the
// opponents' vulnerable 4S, so West sacrifices; but vulnerable against
// non-vulnerable opponents it costs 500 against their 420, so West passes.
func TestSacrificeVulnerabilityGate(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	mk := func(vul [2]bool) []SeatCall {
		d := dealWith(north, map[int]*Hand{
			north: hand("AKJ87", "542", "KJ", "K54"),
			east:  hand("3", "KQJT97", "A54", "Q96"),
			south: hand("QT94", "A", "T987", "A873"),
			west:  hand("652", "863", "Q632", "JT2"), // three hearts: nine-card fit
		})
		d.Vul = vul
		return NewEngine(d).Run()
	}

	calls := mk([2]bool{true, false}) // NS vul: 300 < 620, sacrifice
	sawSacrifice := false
	for _, sc := range calls {
		if sc.Seat == west && sc.Call == bid(5, SHearts) {
			sawSacrifice = true
		}
	}
	if !sawSacrifice {
		t.Fatalf("NS vul: West never sacrificed at 5C\nauction: %s", formatAuction(calls))
	}

	calls = mk([2]bool{false, true}) // EO vul: 500 > 420, no sacrifice
	if final, _, _ := finalContract(calls); final.Format("fr") != "4P" {
		t.Fatalf("EO vul: final contract = %s, want 4P (sacrifice too expensive)\nauction: %s",
			final.Format("fr"), formatAuction(calls))
	}
}

// TestPenaltyDoubleOfSacrifice: North-South, on genuine game values, are
// outbid by West's 5H save. North's combined count no longer covers the five
// level (29 < 30), so bidding 5S would be a guess -- but letting 5H play
// undoubled hands the opponents a bargain. With the opponents' shown
// strength marking the call as a sacrifice, North doubles for penalties.
func TestPenaltyDoubleOfSacrifice(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("AKJ87", "54", "KJ2", "K54"), // 15H, opens 1SA
		east:  hand("3", "KQJT97", "A54", "Q96"), // 12H, six hearts
		south: hand("QT94", "A", "T987", "A873"), // 11H, four spades
		west:  hand("652", "8632", "Q63", "JT2"), // ten-card heart fit with East
	})
	d.Vul = [2]bool{true, false}
	calls := NewEngine(d).Run()

	sawDouble := false
	for _, sc := range calls {
		if sideOf(sc.Seat) == sideOf(north) && sc.Call.Kind == KindDouble {
			sawDouble = true
			if !strings.Contains(sc.M.fr, "punitif") {
				t.Fatalf("double meaning = %q, want the penalty double of the sacrifice", sc.M.fr)
			}
		}
	}
	if !sawDouble {
		t.Fatalf("North-South never doubled the 5C sacrifice\nauction: %s", formatAuction(calls))
	}
	final, _, doubled := finalContract(calls)
	if final.Format("fr") != "5C" || !doubled {
		t.Fatalf("final contract = %s (doubled=%v), want 5C doubled\nauction: %s",
			final.Format("fr"), doubled, formatAuction(calls))
	}
}
