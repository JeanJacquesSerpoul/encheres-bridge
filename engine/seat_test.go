package engine

import (
	"fmt"
	"testing"
)

// TestSeatInConditions: seat is the bidder's rank in the auction, counted
// from the dealer (1 … 4), whoever the dealer is.
func TestSeatInConditions(t *testing.T) {
	saved := activeRules.Load()
	defer SetRules(saved)

	// Only the fourth seat opens; everyone else passes.
	rs, err := LoadRules([]byte(`
- id: fourth
  seq: ""
  call: 1C
  cond: "seat == 4"
  forcing: NF
  meaning: m
  meaning_en: m
  status: sef
- id: pass
  seq: "**"
  call: P
  cond: "true"
  forcing: NF
  meaning: m
  meaning_en: m
  status: sef
`))
	if err != nil {
		t.Fatal(err)
	}
	SetRules(rs)
	const deal = "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"
	for _, dealer := range []string{"N", "E", "S", "W"} {
		d := mustParsePBN(t, fmt.Sprintf("[Dealer %q]\n[Deal %q]\n", dealer, deal))
		calls := NewEngine(d).Run()
		for i, c := range calls[:4] {
			want := "Pass"
			if i == 3 {
				want = "1C"
			}
			if got := c.Call.Format("en"); got != want {
				t.Errorf("donneur %s, %de position : %s, attendu %s", dealer, i+1, got, want)
			}
		}
	}
}
