package engine

// Loading the bidding rules: the YAML file of cli/rules/ (format described in
// tools/python_tools/SEF_2024_spec.md §2-3), its `for:` templates expanded and
// every rule validated -- a port of load() and validate() in
// tools/python_tools/sef_rules.py, down to the order the substitutions run in.

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"

	"gopkg.in/yaml.v3"
)

// Rule is one expanded, validated rule.
type Rule struct {
	ID        string
	Src       string   // id of the template it was expanded from
	Seq       []string // alternative patterns (a plain `seq` is a one-item list)
	CallText  string   // P, X, XX, 1C ... 7NT, as written in the rules
	Call      Call
	CondText  string
	Forcing   string
	Meaning   string
	MeaningEN string
	Status    string
	Alert     bool
	Option    string
	Trump     string // S, H, D, C or ""

	cond *node
}

// RuleSet is the ordered rule list: the first rule that applies wins.
type RuleSet struct {
	Rules []*Rule
}

var (
	callRe        = regexp.MustCompile(`^(P|X|XX|[1-7](C|D|H|S|NT))$`)
	placeholderRe = regexp.MustCompile(`\{[A-Za-z]+\}`)
	forcingSet    = map[string]bool{"NF": true, "F1": true, "FM": true, "SO": true, "INV": true, "REL": true, "ASK": true, "TO": true, "PEN": true}
	statusSet     = map[string]bool{"sef": true, "choix": true, "sef2018": true, "infere": true, "a_verifier": true}
	ruleFields    = map[string]bool{"id": true, "seq": true, "call": true, "cond": true, "forcing": true, "meaning": true,
		"meaning_en": true, "status": true, "alert": true, "option": true, "for": true, "trump": true}
	requiredFields = []string{"id", "seq", "call", "cond", "forcing", "meaning", "meaning_en", "status"}
)

// ---------------------------------------------------------------- ordered YAML values

// omap is a YAML mapping that keeps its key order, as a Python dict does.
type omap struct {
	keys []string
	vals map[string]any
}

func (m *omap) set(k string, v any) {
	if m.vals == nil {
		m.vals = map[string]any{}
	}
	if _, ok := m.vals[k]; !ok {
		m.keys = append(m.keys, k)
	}
	m.vals[k] = v
}

func (m *omap) get(k string) (any, bool) { v, ok := m.vals[k]; return v, ok }

func yamlValue(n *yaml.Node) (any, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return yamlValue(n.Content[0])
	case yaml.AliasNode:
		return yamlValue(n.Alias)
	case yaml.ScalarNode:
		var v any
		if err := n.Decode(&v); err != nil {
			return nil, fmt.Errorf("ligne %d : %v", n.Line, err)
		}
		return v, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := yamlValue(c)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.MappingNode:
		m := &omap{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, err := yamlValue(n.Content[i])
			if err != nil {
				return nil, err
			}
			v, err := yamlValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			m.set(pyStr(k), v)
		}
		return m, nil
	}
	return nil, fmt.Errorf("ligne %d : nœud YAML inattendu", n.Line)
}

// pyStr is Python's str() on the scalars a YAML file holds.
func pyStr(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case float64:
		return pyFloat(x)
	}
	return fmt.Sprint(v)
}

// ---------------------------------------------------------------- expansion

// expandRules reads the YAML text and expands the `for:` templates, giving
// the same list of rules (field order included) sef_rules.py's load() does.
func expandRules(data []byte) ([]*omap, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("YAML invalide : %v", err)
	}
	root, err := yamlValue(&doc)
	if err != nil {
		return nil, fmt.Errorf("YAML invalide : %v", err)
	}
	if root == nil {
		return nil, nil
	}
	list, ok := root.([]any)
	if !ok {
		return nil, errors.New("le fichier de règles doit être une liste de règles")
	}
	var rules []*omap
	for i, item := range list {
		r, ok := item.(*omap)
		if !ok {
			return nil, fmt.Errorf("entrée %d : une règle doit être un dictionnaire", i+1)
		}
		id, ok := r.get("id")
		if !ok {
			return nil, fmt.Errorf("entrée %d : champ id manquant", i+1)
		}
		subs := []*omap{{}}
		if f, _ := r.get("for"); f != nil {
			fl, ok := f.([]any)
			if !ok {
				return nil, fmt.Errorf("%s : for doit être une liste", pyStr(id))
			}
			if len(fl) > 0 {
				subs = subs[:0]
				for _, s := range fl {
					sm, ok := s.(*omap)
					if !ok {
						return nil, fmt.Errorf("%s : chaque élément de for doit être un dictionnaire", pyStr(id))
					}
					subs = append(subs, sm)
				}
			}
		}
		for _, s := range subs {
			e := &omap{}
			for _, k := range r.keys {
				if k == "for" {
					continue
				}
				v := r.vals[k]
				switch x := v.(type) {
				case string:
					for _, sk := range s.keys {
						x = strings.ReplaceAll(x, "{"+sk+"}", pyStr(s.vals[sk]))
					}
					v = x
				case []any:
					if k == "seq" {
						out := make([]any, len(x))
						for j, item := range x {
							str, ok := item.(string)
							if !ok {
								return nil, fmt.Errorf("%s : les motifs de seq doivent être des chaînes", pyStr(id))
							}
							for _, sk := range s.keys {
								str = strings.ReplaceAll(str, "{"+sk+"}", pyStr(s.vals[sk]))
							}
							out[j] = str
						}
						v = out
					}
				}
				e.set(k, v)
			}
			e.set("_src", id)
			rules = append(rules, e)
		}
	}
	return rules, nil
}

// ---------------------------------------------------------------- validation

func parseRuleCall(s string) Call {
	switch s {
	case "P":
		return passCall
	case "X":
		return doubleCall
	case "XX":
		return Call{Kind: KindRedouble}
	}
	level := int(s[0] - '0')
	for st, name := range strainEN {
		if s[1:] == name {
			return bid(level, Strain(st))
		}
	}
	return passCall
}

// LoadRules reads, expands and validates a rules file. Every problem is
// reported at once, one line per rule, as `sef_rules.py --validate` does.
func LoadRules(data []byte) (*RuleSet, error) {
	raw, err := expandRules(data)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, errors.New("le fichier de règles ne contient aucune règle")
	}
	ids := map[string]int{}
	for _, r := range raw {
		if id, ok := r.get("id"); ok {
			ids[pyStr(id)]++
		}
	}
	var errs []string
	set := &RuleSet{}
	for _, r := range raw {
		rid := "?"
		if id, ok := r.get("id"); ok {
			rid = pyStr(id)
		}
		var miss []string
		for _, f := range requiredFields {
			if _, ok := r.get(f); !ok {
				miss = append(miss, f)
			}
		}
		if len(miss) > 0 {
			errs = append(errs, fmt.Sprintf("%s : champs manquants %v", rid, miss))
			continue
		}
		bad := len(errs)
		var extra []string
		for _, k := range r.keys {
			if !ruleFields[k] && k != "_src" {
				extra = append(extra, k)
			}
		}
		if len(extra) > 0 {
			errs = append(errs, fmt.Sprintf("%s : champs inconnus %v", rid, extra))
		}
		if ids[rid] > 1 {
			errs = append(errs, fmt.Sprintf("%s : id en double", rid))
		}
		call := pyStr(r.vals["call"])
		if !callRe.MatchString(call) {
			errs = append(errs, fmt.Sprintf("%s : enchère invalide %q", rid, call))
		}
		forcing, _ := r.vals["forcing"].(string)
		if !forcingSet[forcing] {
			errs = append(errs, fmt.Sprintf("%s : forcing invalide %q", rid, pyStr(r.vals["forcing"])))
		}
		trump := ""
		if t, ok := r.get("trump"); ok && t != nil && pyTruthy(t) {
			trump, _ = t.(string)
			if trump != "S" && trump != "H" && trump != "D" && trump != "C" {
				errs = append(errs, fmt.Sprintf("%s : trump invalide %q", rid, pyStr(t)))
			}
		}
		status, _ := r.vals["status"].(string)
		if !statusSet[status] {
			errs = append(errs, fmt.Sprintf("%s : status invalide %q", rid, pyStr(r.vals["status"])))
		}
		for _, k := range r.keys {
			if s, ok := r.vals[k].(string); ok && k != "_src" && placeholderRe.MatchString(s) {
				errs = append(errs, fmt.Sprintf("%s : substitution non résolue", rid))
				break
			}
		}
		var pats []string
		switch x := r.vals["seq"].(type) {
		case []any:
			for _, p := range x {
				pats = append(pats, pyStr(p))
			}
		default:
			pats = []string{pyStr(x)}
		}
	tokens:
		for _, tok := range strings.Fields(strings.Join(pats, " ")) {
			if strings.HasPrefix(tok, "BW:") || tok == "*" || tok == "**" {
				continue
			}
			for _, alt := range strings.Split(strings.Trim(tok, "()"), "|") {
				if !callRe.MatchString(alt) {
					errs = append(errs, fmt.Sprintf("%s : séquence invalide %q", rid, strings.Join(pats, " / ")))
					break tokens
				}
			}
		}
		condText := pyStr(r.vals["cond"])
		cond, err := compileCond(strings.TrimSpace(condText))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s : condition %q -> %v", rid, condText, err))
		}
		if len(errs) > bad {
			continue
		}
		rule := &Rule{
			ID: rid, Src: pyStr(r.vals["_src"]), Seq: pats, CallText: call, Call: parseRuleCall(call),
			CondText: condText, Forcing: forcing, Status: status, Trump: trump, cond: cond,
			Meaning: pyStr(r.vals["meaning"]), MeaningEN: pyStr(r.vals["meaning_en"]),
		}
		if a, ok := r.get("alert"); ok {
			rule.Alert = pyTruthy(a)
		}
		if o, ok := r.get("option"); ok && pyTruthy(o) {
			rule.Option = pyStr(o)
		}
		set.Rules = append(set.Rules, rule)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%d erreur(s) dans les règles :\n  - %s", len(errs), strings.Join(errs, "\n  - "))
	}
	return set, nil
}

// pyTruthy is Python's bool() on a YAML value.
func pyTruthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int:
		return x != 0
	case float64:
		return x != 0
	case []any:
		return len(x) > 0
	case *omap:
		return len(x.keys) > 0
	}
	return true
}

// ---------------------------------------------------------------- the loaded rules

var activeRules atomic.Pointer[RuleSet]

// SetRules installs the rules every later auction uses.
func SetRules(rs *RuleSet) { activeRules.Store(rs) }

// LoadRulesText parses, validates and installs a rules file, returning the
// number of (expanded) rules.
func LoadRulesText(data []byte) (int, error) {
	rs, err := LoadRules(data)
	if err != nil {
		return 0, err
	}
	SetRules(rs)
	return len(rs.Rules), nil
}

var errNoRules = errors.New("no bidding rules loaded (rules/default.yaml)")

func currentRules() (*RuleSet, error) {
	rs := activeRules.Load()
	if rs == nil {
		return nil, errNoRules
	}
	return rs, nil
}
