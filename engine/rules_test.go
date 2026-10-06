package engine

// The Go engine against the Python reference of tools/python_tools: the
// expansion of the YAML against rules.json, and the 9 326 cases of the SEF's
// tests.json (hand features and chosen rule) against features/choose. Both
// run for every system of cli/systems/index.json, each against its own files
// in testdata/systems/<id>/ (see systems_test.go).

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// rulesPath is the SEF, the page's default system: the rules every test of
// the package bids with unless it says otherwise.
const rulesPath = "../cli/systems/sef/rules.yaml"

var (
	testRulesOnce sync.Once
	testRules     *RuleSet
	testRulesErr  error
)

// loadTestRules loads cli/systems/sef/rules.yaml once and installs it, so every
// test of the package bids with the rules the page ships.
func loadTestRules(t testing.TB) *RuleSet {
	t.Helper()
	testRulesOnce.Do(func() {
		data, err := os.ReadFile(rulesPath)
		if err != nil {
			testRulesErr = err
			return
		}
		testRules, testRulesErr = LoadRules(data)
		if testRulesErr == nil {
			SetRules(testRules)
		}
	})
	if testRulesErr != nil {
		t.Fatal(testRulesErr)
	}
	return testRules
}

func TestMain(m *testing.M) {
	// Every test bids with the shipped rules, whichever runs first.
	if data, err := os.ReadFile(rulesPath); err == nil {
		if rs, err := LoadRules(data); err == nil {
			SetRules(rs)
		}
	}
	os.Exit(m.Run())
}

// plain turns the loader's ordered maps into what encoding/json decodes.
func plain(v any) any {
	switch x := v.(type) {
	case *omap:
		m := map[string]any{}
		for _, k := range x.keys {
			if !strings.HasPrefix(k, "_") {
				m[k] = plain(x.vals[k])
			}
		}
		return m
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = plain(e)
		}
		return out
	case int:
		return float64(x)
	}
	return v
}

func TestExpansionMatchesPython(t *testing.T) {
	forEachSystem(t, testExpansionMatchesPython)
}

func testExpansionMatchesPython(t *testing.T, s *testSystem) {
	data, err := os.ReadFile(s.file("rules.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := expandRules(data)
	if err != nil {
		t.Fatal(err)
	}
	ref := s.data("rules.json")
	pyData, err := os.ReadFile(ref)
	if err != nil {
		t.Fatalf("%v (générer : %s)", err, s.regenerateCmd())
	}
	var want []map[string]any
	if err := json.Unmarshal(pyData, &want); err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(ref)
	if len(raw) != len(want) {
		t.Fatalf("%d règles expansées, %s en compte %d (regénérer : %s)", len(raw), name, len(want), s.regenerateCmd())
	}
	for i, r := range raw {
		got := plain(r).(map[string]any)
		if !reflect.DeepEqual(got, want[i]) {
			t.Fatalf("règle %d diffère de %s :\n go : %v\n py : %v\n(regénérer : %s)", i+1, name, got, want[i], s.regenerateCmd())
		}
		var keys []string
		for _, k := range r.keys {
			if !strings.HasPrefix(k, "_") {
				keys = append(keys, k)
			}
		}
		if len(keys) != len(want[i]) {
			t.Fatalf("règle %s : champs %v", got["id"], keys)
		}
	}
	if len(s.Rules.Rules) != len(want) {
		t.Fatalf("LoadRules garde %d règles sur %d", len(s.Rules.Rules), len(want))
	}
}

// parseRuleHand reads a hand written S.H.D.C, "-" for a void.
func parseRuleHand(s string) *Hand {
	parts := strings.Split(s, ".")
	h := &Hand{}
	for i, p := range parts {
		if p == "-" {
			p = ""
		}
		h.Suits[ruleSuits[i]] = sortRanks(p)
	}
	return h
}

// featureDump is sef_rules.py's feature_dump(): what each test case records.
func featureDump(f *features) map[string]any {
	out := map[string]any{}
	for _, n := range featureNames {
		v, _ := f.scalar(n)
		out[n] = jsonValue(v)
	}
	for _, n := range funcNames {
		m := map[string]any{}
		for _, s := range ruleSuits {
			m[suitLetter[s]] = jsonValue(suitFuncs[n](f, s))
		}
		out[n] = m
	}
	return out
}

func jsonValue(v value) any {
	switch v.kind {
	case kBool:
		return v.i != 0
	case kInt:
		return float64(v.i)
	case kFloat:
		return v.f
	}
	return v.s
}

type sefCase struct {
	ID       int      `json:"id"`
	Hand     string   `json:"hand"`
	Seq      string   `json:"seq"`
	Trump    *string  `json:"trump"`
	Options  []string `json:"options"`
	Vul      bool     `json:"vul"`
	OppVul   bool     `json:"opp_vul"`
	Seat     int      `json:"seat"`
	Lvl      int      `json:"lvl"`
	PSuit    string   `json:"p_suit"`
	PLen     int      `json:"p_len"`
	PForcing bool     `json:"p_forcing"`
	Expected struct {
		Call *string `json:"call"`
		Rule *string `json:"rule"`
	} `json:"expected"`
	Features map[string]any `json:"features"`
}

func TestConformancePython(t *testing.T) {
	forEachSystem(t, testConformancePython)
}

func testConformancePython(t *testing.T, s *testSystem) {
	rs := s.Rules
	ref := s.data("tests.json")
	data, err := os.ReadFile(ref)
	if err != nil {
		t.Fatalf("%v (générer : %s)", err, s.regenerateCmd())
	}
	var suite struct {
		SHA   string    `json:"source_sha256"`
		Count int       `json:"count"`
		Cases []sefCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	yamlData, _ := os.ReadFile(s.file("rules.yaml"))
	// The fingerprint ignores line endings, as sef_rules.py computes it: a
	// Windows checkout (CRLF) and the CI's (LF) hold the same rules.
	sum := sha256.Sum256(bytes.ReplaceAll(yamlData, []byte("\r\n"), []byte("\n")))
	if hex.EncodeToString(sum[:]) != suite.SHA {
		t.Fatalf("%s a été généré depuis une autre version de %s : regénérer (%s)", ref, s.file("rules.yaml"), s.regenerateCmd())
	}
	if len(suite.Cases) != suite.Count {
		t.Fatalf("%d cas lus, l'en-tête en annonce %d", len(suite.Cases), suite.Count)
	}
	bad := 0
	for _, c := range suite.Cases {
		f := newFeatures(parseRuleHand(c.Hand))
		if got := featureDump(f); !reflect.DeepEqual(got, c.Features) {
			bad++
			if bad <= 10 {
				t.Errorf("cas %d %s : features\n go : %v\n py : %v", c.ID, c.Hand, got, c.Features)
			}
			continue
		}
		f.vul, f.oppVul, f.seat = c.Vul, c.OppVul, c.Seat
		f.lvl, f.pSuit, f.pLen, f.pForcing = c.Lvl, c.PSuit, c.PLen, c.PForcing
		opts := map[string]bool{}
		for _, o := range c.Options {
			opts[o] = true
		}
		trump := ""
		if c.Trump != nil {
			trump = *c.Trump
		}
		r := rs.choose(strings.Fields(c.Seq), f, opts, trump)
		gotRule, wantRule := "", ""
		if r != nil {
			gotRule = r.ID
		}
		if c.Expected.Rule != nil {
			wantRule = *c.Expected.Rule
		}
		if gotRule != wantRule || (r != nil && r.CallText != *c.Expected.Call) {
			bad++
			if bad <= 10 {
				t.Errorf("cas %d %s | %q (atout %q) : règle %q, attendu %q", c.ID, c.Hand, c.Seq, trump, gotRule, wantRule)
			}
		}
	}
	if bad > 0 {
		t.Fatalf("%d cas sur %d en échec", bad, len(suite.Cases))
	}
	t.Logf("%d cas conformes", len(suite.Cases))
}
