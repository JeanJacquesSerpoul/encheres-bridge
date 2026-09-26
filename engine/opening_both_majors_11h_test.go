package engine

import "testing"

// TestOpeningBothMajors11H pins [O-6b]: at 11 H, a 5-4 or 4-5 in the majors
// opens the five-card major in any seat, where the same 11 H without the
// second major stays below [O-6] and passes in first seat.
func TestOpeningBothMajors11H(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		hcp        int
		seat       int
		want       string
	}{
		{"4-5, première : 1C", "QT52", "K8752", "73", "AQ", 11, 0, "1C"},
		{"5-4, deuxième : 1P", "K8752", "QT52", "73", "AQ", 11, 1, "1P"},
		{"10 H : sous le plancher", "QT52", "K8752", "73", "AJ", 10, 0, "Passe"},
		{"11 H, cœurs par cinq et trois piques : passe", "QT5", "K8752", "732", "AQ", 11, 0, "Passe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hd := hand(tc.s, tc.h, tc.d, tc.c)
			if hd.H() != tc.hcp {
				t.Fatalf("la main vaut %d H, le cas en annonce %d", hd.H(), tc.hcp)
			}
			c, mn := openingWith(hd, tc.seat)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("ouverture = %s (%s), attendu %s", got, mn.fr, tc.want)
			}
			if tc.want != "Passe" && (mn.minPts != 11 || mn.lens[Hearts] < 4 || mn.lens[Spades] < 4) {
				t.Fatalf("sens annoncé %d H, %d♥ %d♠ (%s)", mn.minPts, mn.lens[Hearts], mn.lens[Spades], mn.fr)
			}
		})
	}
}

// TestOpeningBothMajors11HDeal replays the reported deal: West, dealer, with
// QT52 K8752 73 AQ, opened nothing and North-South reached 3NT unopposed.
func TestOpeningBothMajors11HDeal(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "W"][Vulnerable "EW"][Deal "E:J964.T43.J4.9875 AK8.96.KT652.T62 QT52.K8752.73.AQ 73.AQJ.AQ98.KJ43"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	if got := calls[0].Call.Format("fr"); got != "1C" {
		t.Fatalf("West opened %s (%s), want 1C\nauction: %s", got, calls[0].M.fr, formatAuction(calls))
	}
}
