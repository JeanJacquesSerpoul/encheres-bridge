package engine

import (
	"fmt"
	"testing"
)

// TestVulnerabilityInConditions: vul and opp_vul give each seat its own
// side's vulnerability and the opponents', from the deal's [Vulnerable] tag.
func TestVulnerabilityInConditions(t *testing.T) {
	saved := activeRules.Load()
	defer SetRules(saved)

	// The first seat to speak opens 1C when vulnerable against not, 1D when
	// not vulnerable against vulnerable, 1H at equal vulnerability; everyone
	// else passes.
	rs, err := LoadRules([]byte(`
- id: unfav
  seq: ""
  call: 1C
  cond: "vul and not opp_vul"
  forcing: NF
  meaning: m
  meaning_en: m
  status: sef
- id: fav
  seq: ""
  call: 1D
  cond: "opp_vul and not vul"
  forcing: NF
  meaning: m
  meaning_en: m
  status: sef
- id: equal
  seq: ""
  call: 1H
  cond: "vul == opp_vul"
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
	for _, c := range []struct{ dealer, vul, want string }{
		{"N", "None", "1H"},
		{"N", "NS", "1C"},
		{"N", "EW", "1D"},
		{"N", "All", "1H"},
		{"E", "NS", "1D"}, // East's side is E/W: not vulnerable, opponents are
		{"E", "EW", "1C"},
	} {
		d := mustParsePBN(t, fmt.Sprintf("[Dealer %q]\n[Vulnerable %q]\n[Deal %q]\n", c.dealer, c.vul, deal))
		calls := NewEngine(d).Run()
		if got := calls[0].Call.Format("en"); got != c.want {
			t.Errorf("donneur %s, vulnérabilité %s : %s, attendu %s", c.dealer, c.vul, got, c.want)
		}
	}
}
