package main

import (
	"strings"
	"testing"
)

// TestNotrumpSlamShape pins the shape a quantitative 4SA may be built on
// [S-13b]: everything the notrump openings promise, plus the hands with a
// lone singleton -- but never a singleton beside a long suit, whose slam
// belongs in that suit rather than in notrump.
func TestNotrumpSlamShape(t *testing.T) {
	cases := []struct {
		name       string
		s, h, d, c string
		want       bool
	}{
		{"4-3-3-3", "AQJ7", "AK9", "Q63", "J54", true},
		{"5-3-3-2, régulière", "AQJ87", "AK9", "Q63", "J5", true},
		{"6-3-2-2, semi-régulière", "AQJ876", "AK9", "Q6", "J5", true},
		{"5-4-3-1, le singleton isolé", "AQJ87", "AK96", "Q63", "4", true},
		{"4-4-4-1", "AQJ7", "AK96", "Q632", "4", true},
		{"5-4-4-0 : la chicane", "AQJ87", "AK96", "Q632", "", false},
		{"5-5-2-1 : le singleton reste isolé", "AQJ87", "AK963", "Q6", "4", true},
		{"6-5-1-1 : deux singletons", "AQJ876", "AK963", "Q", "4", false},
		{"6-3-3-1 : la couleur avant les Sans-Atout", "AQJ876", "AK9", "Q63", "4", false},
		{"7-3-2-1 : encore moins", "AQJT876", "AK9", "Q6", "4", false},
		{"7-2-2-2", "AQJT876", "A9", "Q6", "J5", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hand(tc.s, tc.h, tc.d, tc.c).notrumpSlamShape(); got != tc.want {
				t.Fatalf("notrumpSlamShape = %v, attendu %v", got, tc.want)
			}
		})
	}
}

// TestQuantitative4NTOnUnbalancedHand pins the relaxation itself [S-13b]. Nord
// tient 20 H en 5-3-4-1 face à la redemande à 1SA de 12-14 : 32 combinés au
// plancher, 34 au plafond. L'ancienne règle exigeait une main régulière et
// posait 3SA ; le singleton ne change pourtant rien à l'arithmétique, et le
// camp a 33 points quand le partenaire est au sommet de sa zone.
func TestQuantitative4NTOnUnbalancedHand(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "N:AQJ87.AK9.AQ63.4 KT32.T5.T72.9875 95.86.KJ95.AKQT2 64.QJ7432.84.J63"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatal(err)
	}
	calls := NewEngine(d).Run()

	got, comment := nthCallText(t, calls, 0, 1) // deuxième enchère de Nord
	if got != "4SA" || !strings.Contains(comment, "quantitatif") {
		t.Fatalf("enchère de Nord = %s (%s), attendu 4SA quantitatif ; enchères : %s",
			got, comment, formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract.Format("fr") != "6SA" {
		t.Fatalf("contrat final = %s, attendu 6SA ; enchères : %s",
			contract.Format("fr"), formatAuction(calls))
	}
}
