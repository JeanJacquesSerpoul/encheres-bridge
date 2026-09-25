package engine

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// ---------- random deal generation (seeded, reproducible) ----------

func randomDeal(rng *rand.Rand) *Deal {
	type card struct {
		s Suit
		r byte
	}
	var deck []card
	for s := Clubs; s <= Spades; s++ {
		for i := 0; i < len(rankOrder); i++ {
			deck = append(deck, card{s, rankOrder[i]})
		}
	}
	rng.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	d := &Deal{Dealer: rng.Intn(4)}
	for seat := 0; seat < 4; seat++ {
		var bySuit [4][]byte
		for _, c := range deck[seat*13 : (seat+1)*13] {
			bySuit[c.s] = append(bySuit[c.s], c.r)
		}
		h := &Hand{}
		for s := Clubs; s <= Spades; s++ {
			h.Suits[s] = sortRanks(string(bySuit[s]))
		}
		d.Hands[seat] = h
	}
	return d
}

func dealPBN(d *Deal) string {
	// PBN lists the hands clockwise starting from the named seat.
	var hs []string
	for i := 0; i < 4; i++ {
		h := d.Hands[(d.Dealer+i)%4]
		hs = append(hs, fmt.Sprintf("%s.%s.%s.%s",
			h.Suits[Spades], h.Suits[Hearts], h.Suits[Diamonds], h.Suits[Clubs]))
	}
	return fmt.Sprintf("[Dealer %q]\n[Deal \"%s:%s\"]", seatNames[d.Dealer], seatNames[d.Dealer], strings.Join(hs, " "))
}

// maxHandValue is the most generous honest valuation of a hand: HL, or HLD in
// its best suit. No meaning may promise more points than this.
func maxHandValue(h *Hand) int {
	v := h.HL()
	for s := Clubs; s <= Spades; s++ {
		if hld := h.HLD(s); hld > v {
			v = hld
		}
	}
	return v
}

// maxFitValue adds to maxHandValue the point each trump beyond the eighth is
// worth once partner has shown his length [R-1]: HLD in a suit, plus the
// combined length over eight. partnerLens is what partner's calls have
// promised so far.
func maxFitValue(h *Hand, partnerLens [4]int) int {
	v := maxHandValue(h)
	for s := Clubs; s <= Spades; s++ {
		if fit := h.Len(s) + partnerLens[s]; fit > 8 {
			if hld := h.HLD(s) + fit - 8; hld > v {
				v = hld
			}
		}
	}
	return v
}

// TestConsistencyInvariants runs the engine over a large batch of seeded
// random deals and enforces the hard invariants every auction must satisfy:
// legality of each call, termination without the safety net, honesty of the
// recorded meanings (never promise a length, a stopper or a strength the hand
// does not hold), coherent ranges, and forcing bids never passed out by a
// partner left free to answer.
func TestConsistencyInvariants(t *testing.T) {
	const deals = 25000
	rng := rand.New(rand.NewSource(42))
	failures := 0
	report := func(d *Deal, calls []SeatCall, format string, args ...any) {
		failures++
		if failures <= 12 {
			t.Errorf("%s\ndeal:\n%s\nauction: %s", fmt.Sprintf(format, args...), dealPBN(d), formatAuction(calls))
		}
	}

	for n := 0; n < deals; n++ {
		d := randomDeal(rng)
		calls := NewEngine(d).Run()

		// Termination without the 40-call safety net.
		if len(calls) >= 40 {
			report(d, calls, "deal %d: auction hit the safety net (%d calls)", n, len(calls))
			continue
		}

		var lastBid Call
		hasBid := false
		lastBidSide := -1
		doubled, redoubled := false, false
		var shown [4][4]int // per seat, the longest length promised per suit
		for i, sc := range calls {
			h := d.Hands[sc.Seat]
			mn := sc.M
			partnerLens := shown[partnerOf(sc.Seat)]
			for s := Clubs; s <= Spades; s++ {
				shown[sc.Seat][s] = max(shown[sc.Seat][s], mn.lens[s])
			}

			// Basic auction legality.
			switch sc.Call.Kind {
			case KindBid:
				if hasBid && !sc.Call.higherThan(lastBid) {
					report(d, calls, "deal %d: call #%d %s is not higher than %s", n, i, sc.Call.Format("fr"), lastBid.Format("fr"))
				}
				lastBid, hasBid, lastBidSide = sc.Call, true, sideOf(sc.Seat)
				doubled, redoubled = false, false
			case KindDouble:
				if !hasBid || doubled || redoubled || lastBidSide == sideOf(sc.Seat) {
					report(d, calls, "deal %d: illegal double at call #%d", n, i)
				}
				doubled = true
			case KindRedouble:
				if !doubled || redoubled || lastBidSide != sideOf(sc.Seat) {
					report(d, calls, "deal %d: illegal redouble at call #%d", n, i)
				}
				redoubled = true
			}

			// Honesty of the meaning against the actual hand.
			for s := Clubs; s <= Spades; s++ {
				if mn.lens[s] > h.Len(s) {
					report(d, calls, "deal %d: call #%d (%s %s, %q) promises %d cards in suit %d, holds %d",
						n, i, seatNames[sc.Seat], sc.Call.Format("fr"), mn.fr, mn.lens[s], s, h.Len(s))
				}
				if mn.stops[s] && !h.Stopper(s) {
					report(d, calls, "deal %d: call #%d (%s %s, %q) promises a stopper in suit %d without one",
						n, i, seatNames[sc.Seat], sc.Call.Format("fr"), mn.fr, s)
				}
			}
			if mn.minPts >= 0 && mn.maxPts >= 0 && mn.minPts > mn.maxPts {
				report(d, calls, "deal %d: call #%d (%s %s, %q) has min %d > max %d",
					n, i, seatNames[sc.Seat], sc.Call.Format("fr"), mn.fr, mn.minPts, mn.maxPts)
			}
			if worth := maxFitValue(h, partnerLens); mn.minPts > worth {
				report(d, calls, "deal %d: call #%d (%s %s, %q) promises %d points, hand is worth at most %d",
					n, i, seatNames[sc.Seat], sc.Call.Format("fr"), mn.fr, mn.minPts, worth)
			}

			// A comment announcing "conclusion à la manche" must sit on a bid
			// that actually is a game contract, never on a mislabeled
			// partscore. Transfers are exempt: they announce the game in
			// another strain, which the completion then reaches.
			if sc.Call.Kind == KindBid && !isGame(sc.Call) && !mn.hasTexas && strings.Contains(mn.fr, "conclusion à la manche") {
				report(d, calls, "deal %d: call #%d (%s %s, %q) claims game but is a partscore",
					n, i, seatNames[sc.Seat], sc.Call.Format("fr"), mn.fr)
			}

			// A forcing bid must not be passed out by a partner whose RHO
			// passed (an intervening bid or double frees the partner).
			if mn.forcing && sc.Call.Kind == KindBid && i+2 < len(calls) {
				lho, partner := calls[i+1], calls[i+2]
				if lho.Call.Kind == KindPass && partner.Call.Kind == KindPass {
					report(d, calls, "deal %d: forcing bid #%d (%s %s, %q) passed out by partner",
						n, i, seatNames[sc.Seat], sc.Call.Format("fr"), mn.fr)
				}
			}
		}

		// A final suit contract at the three level or higher that both partners
		// bid (one named the suit, the other supported it), reached without
		// opposition interference, must rest on a genuine combined trump fit:
		// the engine must never raise itself into a 4-2 "fit" contract. A suit
		// only one hand ever bid (a rebid passed out, a unilateral game on a
		// long suit) is a judgment call, not a phantom raise, and stays out.
		if contract, declarer, _ := finalContract(calls); declarer >= 0 && contract.IsBid() &&
			contract.Strain <= SSpades && contract.Level >= 3 {
			side := sideOf(declarer)
			contested := false
			var bidBySeat [4]bool
			for _, sc := range calls {
				if sideOf(sc.Seat) != side && sc.Call.Kind != KindPass {
					contested = true
					break
				}
				if sc.Call.IsBid() && sc.Call.Strain == contract.Strain {
					bidBySeat[sc.Seat] = true
				}
			}
			s := Suit(contract.Strain)
			combined := d.Hands[declarer].Len(s) + d.Hands[partnerOf(declarer)].Len(s)
			if !contested && bidBySeat[declarer] && bidBySeat[partnerOf(declarer)] && combined < 7 {
				report(d, calls, "deal %d: uncontested raised %s contract on a %d-card combined fit",
					n, contract.Format("fr"), combined)
			}
		}

		// A game-forcing commitment must not die below game when the
		// opponents never interfered.
		if contract, declarer, _ := finalContract(calls); declarer >= 0 && !isGame(contract) {
			side := sideOf(declarer)
			contested := false
			gameForced := false
			for _, sc := range calls {
				if sideOf(sc.Seat) != side && sc.Call.Kind != KindPass {
					contested = true
					break
				}
				if sideOf(sc.Seat) == side && strings.Contains(sc.M.fr, "forcing de manche") {
					gameForced = true
				}
			}
			if gameForced && !contested {
				report(d, calls, "deal %d: game-forcing auction stopped below game", n)
			}
		}
	}
	if failures > 0 {
		t.Fatalf("%d invariant violations over %d deals", failures, deals)
	}
}
