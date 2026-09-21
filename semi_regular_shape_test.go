package main

import "testing"

// TestSemiRegularShapes pins the shapes IsSemiRegular is meant to name. The
// predicate tested the two doubletons alone, which also let 7-2-2-2 through --
// the only longer shape three doubletons leave room for, since 8-2-2-2 needs
// fourteen cards.
func TestSemiRegularShapes(t *testing.T) {
	cases := []struct {
		name string
		h    *Hand
		want bool
	}{
		{"5-4-2-2", hand("AKQJT", "AKQJ", "AK", "AK"), true},
		{"6-3-2-2", hand("AKQJT9", "AKQ", "AK", "AK"), true},
		{"7-2-2-2", hand("AKQJT98", "AK", "AK", "AK"), false},
		{"4-4-3-2", hand("AKQJ", "AKQJ", "AKQ", "AK"), false},
		{"5-3-3-2", hand("AKQJT", "AKQ", "AKQ", "AK"), false},
		{"6-4-2-1", hand("AKQJT9", "AKQJ", "AK", "A"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.h.IsSemiRegular(); got != tc.want {
				t.Fatalf("IsSemiRegular(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// TestSevenCardMajorIsNotBalanced is the auction the defect produced (audit du
// par, donne 398). East opens 1C, South overcalls 1D, West answers 1H holding
// HAQJ9874 with 17 HL, East rebids 1SA. With no fit and no diamond stopper the
// count has no game to aim at, and the six-card-major promotions -- both gated
// on the hand being neither regular nor semi-regular -- skipped a seven-card
// suit as though it were balanced. West passed 1SA.
func TestSevenCardMajorIsNotBalanced(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "None"]
[Deal "E:AKQJ.53.875.QJ83 T96..AKQ943.9765 72.AQJ9874.T6.AK 8543.KT62.J2.T42"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const west = 3
	seen := false
	for _, sc := range calls {
		if sc.Seat == west && sc.Call.IsBid() && sc.Call.Strain == SNoTrump {
			t.Fatalf("West bids notrump (%s) on a 7-2-2-2\nauction: %s", sc.M.fr, formatAuction(calls))
		}
		if sc.Seat == partnerOf(west) && sc.Call.IsBid() && sc.Call.Strain == SNoTrump {
			seen = true
			continue
		}
		if !seen || sc.Seat != west {
			continue
		}
		if !sc.Call.IsBid() || sc.Call.Strain != SHearts {
			t.Fatalf("West answers the 1SA rebid with %s (%s), want the seven-card major\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		return
	}
	t.Fatalf("West never spoke after the 1SA rebid\nauction: %s", formatAuction(calls))
}
