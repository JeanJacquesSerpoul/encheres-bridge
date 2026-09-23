package engine

import (
	"fmt"
	"math/rand"
	"sort"
	"testing"
)

// TestAuditDetail groups the soft-audit misses by auction shape to expose
// systematic holes. Informational only.
func TestAuditDetail(t *testing.T) {
	const deals = 20000
	rng := rand.New(rand.NewSource(4242))
	missCat := map[string]int{}
	missEx := map[string]string{}
	thinGameCat := map[string]int{}
	thinSlamCat := map[string]int{}
	thinSlamEx := map[string]string{}

	for n := 0; n < deals; n++ {
		d := randomDeal(rng)
		calls := NewEngine(d).Run()
		contract, declarer, _ := finalContract(calls)
		if declarer < 0 {
			continue
		}
		side := declarer % 2
		a, b := d.Hands[side], d.Hands[side+2]
		best := a.HL() + b.HL()
		bestFit := -1
		for s := Clubs; s <= Spades; s++ {
			if a.Len(s)+b.Len(s) >= 8 {
				if v := a.HLD(s) + b.HLD(s); v > best {
					best, bestFit = v, int(s)
				}
			}
		}
		contested := false
		for _, sc := range calls {
			if sc.Call.Kind != KindPass && sideOf(sc.Seat) != side {
				contested = true
				break
			}
		}
		// signature: last two non-pass meanings of the declaring side
		sig := func() string {
			var ms []string
			for _, sc := range calls {
				if sideOf(sc.Seat) == side && sc.Call.Kind != KindPass {
					ms = append(ms, sc.M.fr)
				}
			}
			start := len(ms) - 2
			if start < 0 {
				start = 0
			}
			out := ""
			for _, m := range ms[start:] {
				if len(m) > 45 {
					m = m[:45]
				}
				out += " | " + m
			}
			return out
		}
		switch {
		case contract.Level >= 6:
			if best < 30 {
				k := sig()
				thinSlamCat[k]++
				if _, ok := thinSlamEx[k]; !ok {
					thinSlamEx[k] = dealPBN(d) + "\n" + formatAuction(calls)
				}
			}
		case isGame(contract):
			if best < 23 {
				thinGameCat[sig()]++
			}
		default:
			if !contested && best >= 28 {
				k := fmt.Sprintf("fit=%d%s", bestFit, sig())
				missCat[k]++
				if _, ok := missEx[k]; !ok {
					missEx[k] = dealPBN(d) + "\n" + formatAuction(calls)
				}
			}
		}
	}
	dump := func(name string, cat map[string]int, ex map[string]string) {
		type kv struct {
			k string
			v int
		}
		var all []kv
		total := 0
		for k, v := range cat {
			all = append(all, kv{k, v})
			total += v
		}
		sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
		t.Logf("==== %s: total %d", name, total)
		for i, e := range all {
			if i >= 8 {
				break
			}
			t.Logf("%4d  %s", e.v, e.k)
			if ex != nil && i < 4 {
				t.Logf("      example:\n%s", ex[e.k])
			}
		}
	}
	dump("MISSED GAMES (uncontested)", missCat, missEx)
	dump("THIN GAMES", thinGameCat, nil)
	dump("THIN SLAMS", thinSlamCat, thinSlamEx)
}
