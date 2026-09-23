package engine

import (
	"strings"
	"testing"
)

// TestStaymanStandardResponses checks the standard Stayman response ladder
// (docs/bidings.md, "LE STAYMAN"): the suit just above the 2C/3C ask denies
// both majors, the majors themselves show one without the other, and
// notrump shows both. The engine previously used a scrambled "SEF" ladder
// that didn't match this convention.
func TestStaymanStandardResponses(t *testing.T) {
	const north, south = 0, 2
	cases := []struct {
		name  string
		north *Hand // opener, 1SA
		want  string
		hint  string
	}{
		{
			name:  "ni 4 cartes a Coeur ni 4 cartes a Pique",
			north: hand("KJ2", "Q32", "AK43", "A98"), // 17H, 3=3-4-3
			want:  "2K",
			hint:  "pas de majeure quatrième",
		},
		{
			name:  "4 cartes a Coeur sans 4 cartes a Pique",
			north: hand("A32", "KQ32", "K54", "A54"), // 16H, 3-4-3-3
			want:  "2C",
			hint:  "sans 4 cartes à Pique",
		},
		{
			name:  "4 cartes a Pique sans 4 cartes a Coeur",
			north: hand("KQ32", "A32", "K54", "A54"), // 16H, 4-3-3-3
			want:  "2P",
			hint:  "sans 4 cartes à Cœur",
		},
		{
			name:  "4 cartes a Coeur et 4 cartes a Pique",
			north: hand("KQ32", "AJ32", "K5", "Q54"), // 15H, 4-4-2-3
			want:  "2SA",
			hint:  "4 cartes à Cœur et 4 cartes à Pique",
		},
	}
	southHand := hand("83", "K876", "Q1082", "K32")
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := dealWithQuietOpponents(north, map[int]*Hand{
				north: tc.north,
				south: southHand,
			})
			calls := NewEngine(d).Run()
			n := 0
			for _, sc := range calls {
				if sc.Seat != north {
					continue
				}
				if n == 1 {
					if got := sc.Call.Format("fr"); got != tc.want {
						t.Fatalf("North's Stayman answer = %s (%s), want %s\nauction: %s",
							got, sc.M.fr, tc.want, formatAuction(calls))
					}
					if !strings.Contains(sc.M.fr, tc.hint) {
						t.Fatalf("comment %q does not mention %q", sc.M.fr, tc.hint)
					}
					return
				}
				n++
			}
			t.Fatalf("North never answered Stayman\nauction: %s", formatAuction(calls))
		})
	}
}

// TestStaymanBothMajorsOver2NT checks the same "both majors" case one level
// up, over a 2SA opening (3C Stayman): the engine's base==2 branch never
// handled the h4&&s4 case at all and always answered with hearts only.
func TestStaymanBothMajorsOver2NT(t *testing.T) {
	const north, south = 0, 2
	d := dealWithQuietOpponents(north, map[int]*Hand{
		north: hand("KJ32", "QJ32", "AK", "AQ"),
		south: hand("83", "K876", "Q1082", "K32"),
	})
	calls := NewEngine(d).Run()
	n := 0
	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if n == 1 {
			if got := sc.Call.Format("fr"); got != "3SA" {
				t.Fatalf("North's Stayman answer = %s (%s), want 3SA\nauction: %s",
					got, sc.M.fr, formatAuction(calls))
			}
			if !strings.Contains(sc.M.fr, "4 cartes à Cœur et 4 cartes à Pique") {
				t.Fatalf("comment %q does not mention both majors", sc.M.fr)
			}
			return
		}
		n++
	}
	t.Fatalf("North never answered Stayman\nauction: %s", formatAuction(calls))
}
