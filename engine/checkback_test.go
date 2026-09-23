package main

import (
	"strings"
	"testing"
)

// TestCheckbackStayman checks the 3C checkback convention after a jump 2NT
// rebid (18-19) over a one-level major response to a minor opening:
// opener names responder's major with 3 cards (no other four-card major),
// the other major with 4 cards (no 3-card support), 3D with both, 3NT with
// neither; responder then places the contract.
func TestCheckbackStayman(t *testing.T) {
	const north, south = 0, 2
	type check struct {
		seat int
		nth  int    // 0 = first call of that seat
		want string // French notation
		hint string // substring of the French comment
	}
	cases := []struct {
		name     string
		pbn      string
		checks   []check
		contract string
	}{
		{
			// S = KQ3 A52 KQ72 KJ5 (18H): 3 spades, no 4 hearts -> 3S.
			name: "3 cartes dans la majeure, sans l'autre majeure",
			pbn: `[Dealer "S"]
[Deal "S:KQ3.A52.KQ72.KJ5 98.KQ84.JT9.QT98 AJT54.76.853.762 762.JT93.A64.A43"]`,
			checks: []check{
				{south, 1, "2SA", "Checkback"},
				{north, 1, "3T", "Checkback"},
				{south, 2, "3P", "3 cartes dans votre majeure"},
				{north, 2, "4P", "fit 5-3"},
			},
			contract: "4P",
		},
		{
			// S = AQ72 K3 KQ5 KJ52 (18H): 4 spades, 2 hearts -> 3S over the
			// checkback, 4-4 spade fit found.
			name: "4 cartes dans l'autre majeure",
			pbn: `[Dealer "S"]
[Deal "S:AQ72.K3.KQ5.KJ52 K96.QT82.JT4.T96 T854.AJ64.862.Q7 J3.975.A973.A843"]`,
			checks: []check{
				{south, 1, "2SA", "Checkback"},
				{north, 1, "3T", "Checkback"},
				{south, 2, "3P", "4 cartes dans l'autre majeure"},
				{north, 2, "4P", "fit 4-4"},
			},
			contract: "4P",
		},
		{
			// S = KQ3 AQ72 K5 KJ52 (18H): 3 spades and 4 hearts -> 3D.
			name: "les deux conditions, réponse 3K",
			pbn: `[Dealer "S"]
[Deal "S:KQ3.AQ72.K5.KJ52 98.KT83.JT9.QT98 AJT54.64.8632.76 762.J95.AQ74.A43"]`,
			checks: []check{
				{south, 1, "2SA", "Checkback"},
				{north, 1, "3T", "Checkback"},
				{south, 2, "3K", "3 cartes dans votre majeure et 4 cartes dans l'autre"},
				{north, 2, "4P", "fit 5-3"},
			},
			contract: "4P",
		},
		{
			// S = K3 AQ7 KQ52 KJ52 (18H): 2 spades, 3 hearts -> 3NT.
			name: "aucune des deux conditions, réponse 3SA",
			pbn: `[Dealer "S"]
[Deal "S:K3.AQ7.KQ52.KJ52 987.KT83.JT9.T98 AJT54.642.863.76 Q62.J95.A74.AQ43"]`,
			checks: []check{
				{south, 1, "2SA", "Checkback"},
				{north, 1, "3T", "Checkback"},
				{south, 2, "3SA", "ni 3 cartes dans votre majeure"},
				{north, 2, "Passe", "pas de fit majeur"},
			},
			contract: "3SA",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := ParsePBN([]byte(tc.pbn))
			if err != nil {
				t.Fatalf("bad deal: %v", err)
			}
			calls := NewEngine(d).Run()
			for _, ck := range tc.checks {
				n := 0
				found := false
				for _, sc := range calls {
					if sc.Seat != ck.seat {
						continue
					}
					if n == ck.nth {
						if got := sc.Call.Format("fr"); got != ck.want {
							t.Fatalf("call #%d of seat %s = %s (%s), want %s\nauction: %s",
								ck.nth, seatNames[ck.seat], got, sc.M.fr, ck.want, formatAuction(calls))
						}
						if ck.hint != "" && !strings.Contains(sc.M.fr, ck.hint) {
							t.Fatalf("comment %q does not mention %q", sc.M.fr, ck.hint)
						}
						found = true
						break
					}
					n++
				}
				if !found {
					t.Fatalf("seat %s made fewer than %d calls\nauction: %s", seatNames[ck.seat], ck.nth+1, formatAuction(calls))
				}
			}
			contract, _, _ := finalContract(calls)
			if got := contract.Format("fr"); got != tc.contract {
				t.Fatalf("contract = %s, want %s\nauction: %s", got, tc.contract, formatAuction(calls))
			}
		})
	}
}
