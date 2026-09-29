package engine

// Choosing a rule: the sequence patterns and the first-match walk of
// tools/python_tools/sef_rules.py (_match, rule_matches, choose), in the
// semantics of SEF_2024_spec.md §6.

import "strings"

// matchPattern matches one pattern against a sequence, both as tokens.
func matchPattern(p, s []string) bool {
	if len(p) > 0 && p[0] == "**" { // ** : any start (zero or more calls)
		p = p[1:]
		if len(s) < len(p) {
			return false
		}
		s = s[len(s)-len(p):]
	}
	if len(p) != len(s) {
		return false
	}
	for i, pt := range p {
		if pt == "*" {
			continue
		}
		st := s[i]
		if strings.HasPrefix(pt, "(") != strings.HasPrefix(st, "(") {
			return false
		}
		found := false
		for _, alt := range strings.Split(strings.Trim(pt, "()"), "|") {
			if alt == strings.Trim(st, "()") {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// seqMatches reports whether one of the rule's patterns fits the sequence seen
// by the bidding pair. trump is the pair's agreed suit, "" if none.
func (r *Rule) seqMatches(seq []string, trump string) bool {
	stripped := seq
	for len(stripped) > 0 && stripped[0] == "P" {
		stripped = stripped[1:]
	}
	for _, pat := range r.Seq {
		p := strings.Fields(pat)
		target := stripped
		if len(p) > 0 && p[0] == "P" {
			target = seq // a pattern that names the initial passes sees them
		}
		if len(p) > 0 && strings.HasPrefix(p[0], "BW:") { // BW:x : any start, agreed trump x
			if trump == "" || trump != p[0][3:] {
				continue
			}
			p = append([]string{"**"}, p[1:]...)
		}
		if matchPattern(p, target) {
			return true
		}
	}
	return false
}

// choose returns the first rule that applies, or nil: the sequence uncoded.
func (rs *RuleSet) choose(seq []string, f *features, options map[string]bool, trump string) *Rule {
	for _, r := range rs.Rules {
		if r.Option != "" && !options[r.Option] {
			continue
		}
		if !r.seqMatches(seq, trump) {
			continue
		}
		v, err := r.cond.eval(f)
		if err == nil && v.truthy() {
			return r
		}
	}
	return nil
}
