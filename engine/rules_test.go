package engine

// The Go engine against the Python reference of tools/python_tools: the
// expansion of the YAML against sef_rules.json, and the 9 326 cases of
// sef_tests.json (hand features and chosen rule) against features/choose.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
)

const (
	rulesPath     = "../cli/rules/sef_rules.yaml"
	pyRulesJSON   = "../tools/python_tools/sef_rules.json"
	pyTestsJSON   = "../tools/python_tools/sef_tests.json"
	regenerateCmd = "cd tools/python_tools && python sef_rules.py ../../cli/rules/sef_rules.yaml --json sef_rules.json " +
		"--gen-tests 1500 sef_tests.json --seed 2024"
)

var (
	testRulesOnce sync.Once
	testRules     *RuleSet
	testRulesErr  error
)

// loadTestRules loads cli/rules/sef_rules.yaml once and installs it, so every
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
	data, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := expandRules(data)
	if err != nil {
		t.Fatal(err)
	}
	pyData, err := os.ReadFile(pyRulesJSON)
	if err != nil {
		t.Fatal(err)
	}
	var want []map[string]any
	if err := json.Unmarshal(pyData, &want); err != nil {
		t.Fatal(err)
	}
	if len(raw) != len(want) {
		t.Fatalf("%d règles expansées, sef_rules.json en compte %d (regénérer : %s)", len(raw), len(want), regenerateCmd)
	}
	for i, r := range raw {
		got := plain(r).(map[string]any)
		if !reflect.DeepEqual(got, want[i]) {
			t.Fatalf("règle %d diffère de sef_rules.json :\n go : %v\n py : %v\n(regénérer : %s)", i+1, got, want[i], regenerateCmd)
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
	rs := loadTestRules(t)
	if len(rs.Rules) != len(want) {
		t.Fatalf("LoadRules garde %d règles sur %d", len(rs.Rules), len(want))
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
	Expected struct {
		Call *string `json:"call"`
		Rule *string `json:"rule"`
	} `json:"expected"`
	Features map[string]any `json:"features"`
}

func TestConformancePython(t *testing.T) {
	rs := loadTestRules(t)
	data, err := os.ReadFile(pyTestsJSON)
	if err != nil {
		t.Fatal(err)
	}
	var suite struct {
		SHA   string    `json:"source_sha256"`
		Count int       `json:"count"`
		Cases []sefCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &suite); err != nil {
		t.Fatal(err)
	}
	yamlData, _ := os.ReadFile(rulesPath)
	// The fingerprint ignores line endings, as sef_rules.py computes it: a
	// Windows checkout (CRLF) and the CI's (LF) hold the same rules.
	sum := sha256.Sum256(bytes.ReplaceAll(yamlData, []byte("\r\n"), []byte("\n")))
	if hex.EncodeToString(sum[:]) != suite.SHA {
		t.Fatalf("sef_tests.json a été généré depuis une autre version de sef_rules.yaml : regénérer (%s)", regenerateCmd)
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
		f.vul, f.oppVul = c.Vul, c.OppVul
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
