package engine

// Whole auctions against the Python reference: tools/python_tools/gen_golden.py
// bids the deals of testdata/*.pbn and the start of the par bench with
// pbn_auction.py (all four hands, so competition, legality, the end of the
// auction and the agreed trump are all exercised), and this test replays them.

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

const goldenPython = "testdata/golden_python.json"

type goldenCase struct {
	Name       string `json:"name"`
	Dealer     string `json:"dealer"`
	Vulnerable string `json:"vulnerable"`
	Deal       string `json:"deal"`
	Bids       []struct {
		Seat string  `json:"seat"`
		Call string  `json:"call"`
		Rule *string `json:"rule"`
	} `json:"bids"`
	MeaningFR []string `json:"meaning_fr"`
	MeaningEN []string `json:"meaning_en"`
}

// pyCall is pbn_auction.normalize(): Pass, X, XX, 1C ... 7NT.
func pyCall(c Call) string {
	if c.Kind == KindPass {
		return "Pass"
	}
	return ruleToken(c)
}

func TestGoldenPython(t *testing.T) {
	loadTestRules(t)
	data, err := os.ReadFile(goldenPython)
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		Cases []goldenCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	if len(suite.Cases) < 300 {
		t.Fatalf("%d donnes seulement dans %s", len(suite.Cases), goldenPython)
	}
	for _, c := range suite.Cases {
		d := mustParsePBN(t, fmt.Sprintf("[Dealer %q]\n[Vulnerable %q]\n[Deal %q]\n", c.Dealer, c.Vulnerable, c.Deal))
		calls := NewEngine(d).Run()
		if len(calls) != len(c.Bids) {
			t.Errorf("%s : %d enchères, Python en fait %d\n go : %s", c.Name, len(calls), len(c.Bids), formatAuction(calls))
			continue
		}
		for i, sc := range calls {
			want := c.Bids[i]
			rule := ""
			if sc.Rule != nil {
				rule = sc.Rule.ID
			}
			wantRule := ""
			if want.Rule != nil {
				wantRule = *want.Rule
			}
			switch {
			case seatNames[sc.Seat] != want.Seat || pyCall(sc.Call) != want.Call || rule != wantRule:
				t.Errorf("%s enchère %d : %s %s (%s), Python %s %s (%s)", c.Name, i+1,
					seatNames[sc.Seat], pyCall(sc.Call), rule, want.Seat, want.Call, wantRule)
			case sc.comment("fr") != c.MeaningFR[i] || sc.comment("en") != c.MeaningEN[i]:
				t.Errorf("%s enchère %d : commentaire %q / %q, Python %q / %q", c.Name, i+1,
					sc.comment("fr"), sc.comment("en"), c.MeaningFR[i], c.MeaningEN[i])
			default:
				continue
			}
			break
		}
	}
}
