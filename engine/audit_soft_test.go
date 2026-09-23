package main

import (
	"math/rand"
	"testing"
)

// TestAuditSoft: informational statistics, never fails.
func TestAuditSoft(t *testing.T) {
	const deals = 20000
	rng := rand.New(rand.NewSource(4242))
	var passedOut, partscore, game, slam, grand int
	var missedGameStrong, thinGame, thinSlam int
	examplesMissed, examplesThinSlam := []string{}, []string{}

	for n := 0; n < deals; n++ {
		d := randomDeal(rng)
		calls := NewEngine(d).Run()
		contract, declarer, _ := finalContract(calls)
		if declarer < 0 {
			passedOut++
			continue
		}
		side := declarer % 2
		// combined best valuation for the declaring side
		a, b := d.Hands[side], d.Hands[side+2]
		best := a.HL() + b.HL()
		for s := Clubs; s <= Spades; s++ {
			if a.Len(s)+b.Len(s) >= 8 {
				if v := a.HLD(s) + b.HLD(s); v > best {
					best = v
				}
			}
		}
		switch {
		case contract.Level >= 7:
			grand++
		case contract.Level >= 6:
			slam++
			if best < 30 {
				thinSlam++
				if len(examplesThinSlam) < 3 {
					examplesThinSlam = append(examplesThinSlam, dealPBN(d)+"\n"+formatAuction(calls))
				}
			}
		case isGame(contract):
			game++
			if best < 23 {
				thinGame++
			}
		default:
			partscore++
			// Only count a "missed game" on uncontested auctions: with
			// interference the opponents may simply have crowded the side out.
			contested := false
			for _, sc := range calls {
				if sc.Call.Kind != KindPass && sideOf(sc.Seat) != side {
					contested = true
					break
				}
			}
			if !contested && best >= 28 {
				missedGameStrong++
				if len(examplesMissed) < 3 {
					examplesMissed = append(examplesMissed, dealPBN(d)+"\n"+formatAuction(calls))
				}
			}
		}
	}
	t.Logf("deals=%d passedOut=%d partscore=%d game=%d slam=%d grand=%d", deals, passedOut, partscore, game, slam, grand)
	t.Logf("missed game (partscore with 28+ combined)=%d ; thin game (<23)=%d ; thin slam (<30)=%d", missedGameStrong, thinGame, thinSlam)
	for _, ex := range examplesMissed {
		t.Logf("MISSED GAME EXAMPLE:\n%s", ex)
	}
	for _, ex := range examplesThinSlam {
		t.Logf("THIN SLAM EXAMPLE:\n%s", ex)
	}
}
