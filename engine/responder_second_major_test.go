package engine

import (
	"strings"
	"testing"
)

// TestResponderShowsSecondMajor checks that a responder with a game-forcing
// two-suiter names its second major before concluding, rather than jumping
// straight to game on the first suit. After 1C - 1H - 2C, East holds 6 hearts
// and 5 spades (13H): a change of suit by responder is forcing (docs/bidings.md,
// "le changement de couleur par le répondant"), so 2S — the "bicolore cher du
// répondant" — describes the two-suiter and lets opener give an accurate
// preference, instead of the undescriptive 4H jump. The auction still reaches
// the 4H game (opener holds three-card heart support).
func TestResponderShowsSecondMajor(t *testing.T) {
	// Donneur West (opener). E = K8753.AKQT52.J2. : 13H, 6 hearts, 5 spades,
	// void clubs. W opens 1C, E responds 1H, W rebids 2C (own suit).
	pbn := `[Dealer "W"]
[Deal "W:6.J86.AK64.AQ743 Q92.93.T7.KJT862 K8753.AKQT52.J2. AJT4.74.Q9853.95"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1 // seats: 0=N, 1=E, 2=S, 3=W
	n := 0
	for _, sc := range calls {
		if sc.Seat != east {
			continue
		}
		if n == 1 { // East's rebid over 2C
			if got := sc.Call.Format("fr"); got != "2P" {
				t.Fatalf("East rebid = %s (%s), want 2P (responder's second suit)\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "bicolore") {
				t.Fatalf("comment %q does not describe the two-suiter", sc.M.fr)
			}
			break
		}
		n++
	}

	// The two-suiter route must still land the heart game.
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "4C" {
		t.Fatalf("final contract = %s, want 4C (heart game)\nauction: %s", got, formatAuction(calls))
	}
}
