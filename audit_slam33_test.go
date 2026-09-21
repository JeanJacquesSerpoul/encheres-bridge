package main

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

// TestSlamZoneAudit draws random deals until a thousand of them give N/S at
// least 33 combined honour points -- the small-slam zone [E-9] in its least
// arguable form, since E/W then hold seven points or fewer. Every one of
// those auctions ought to reach at least the small slam; the ones that stop
// short are reported, grouped by the call that shut the auction down, with
// one example auction each.
//
// The miss rate is mostly a measure of what the engine's own information
// model can see: a pair counts its own hand plus the *floor* partner
// promised [E-9], so a deal holding exactly 33 where both floors run a point
// low is invisible by construction. The ceiling below only guards against a
// regression, it is not a target.
func TestSlamZoneAudit(t *testing.T) {
	const want = 1000
	const maxMissRate = 0.20

	rng := rand.New(rand.NewSource(20260831))
	cat := map[string]int{}
	ex := map[string]string{}
	kept, tried, slams, grands, misses := 0, 0, 0, 0, 0

	for kept < want {
		tried++
		d := randomDeal(rng)
		hcp := d.Hands[0].H() + d.Hands[2].H()
		if hcp < 33 {
			continue
		}
		kept++
		calls := NewEngine(d).Run()
		contract, declarer, _ := finalContract(calls)
		nsPlays := declarer >= 0 && sideOf(declarer) == 0
		switch {
		case nsPlays && contract.Level == 7:
			grands++
		case nsPlays && contract.Level == 6:
			slams++
		default:
			misses++
			closing := "(passe général)"
			for _, sc := range calls {
				if sideOf(sc.Seat) == 0 && sc.Call.Kind != KindPass {
					closing = sc.M.fr
				}
			}
			if i := strings.Index(closing, " ("); i > 0 {
				closing = closing[:i]
			}
			cat[closing]++
			if _, seen := ex[closing]; !seen {
				ex[closing] = fmt.Sprintf("%d H en NS, contrat %s\n%s\n%s",
					hcp, contract.Format("fr"), dealPBN(d), formatAuction(calls))
			}
		}
	}

	t.Logf("donnes tirées : %d, retenues (33 H et plus en NS) : %d", tried, kept)
	t.Logf("petits chelems : %d · grands chelems : %d · MANQUÉS : %d (%.1f%%)",
		slams, grands, misses, 100*float64(misses)/float64(kept))

	type kv struct {
		k string
		v int
	}
	var all []kv
	for k, v := range cat {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].k < all[j].k
	})
	for i, e := range all {
		if i >= 10 {
			break
		}
		t.Logf("%4d  %s", e.v, e.k)
		if i < 3 {
			t.Logf("      exemple :\n%s", ex[e.k])
		}
	}

	if rate := float64(misses) / float64(kept); rate > maxMissRate {
		t.Fatalf("%.1f%% des donnes à 33 H en NS n'atteignent pas le chelem, plafond %.0f%%",
			100*rate, 100*maxMissRate)
	}
}
