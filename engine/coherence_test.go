package engine

// Coherence report of a bidding system: what no other test looks for. It is
// a harness, not an assertion: skipped unless COHERENCE_OUT names a folder,
// so `go test ./...` is unaffected. See tools/coherence/README.md.
//
//   A  rules never chosen: sequence never met, condition never true, or
//      always shadowed by an earlier rule (the most frequent one is named);
//   B  sequences without a rule (default passes), most frequent first;
//   C  passes given by a rule right after a forcing call of the partner;
//   D  meaning against condition: the "a-b H", "a+ HL", "n levées de jeu" of
//      the meaning that the condition does not bound, and the points ranges
//      of the French meaning missing from the English one;
//   E  English meanings left empty, identical to the French one, or French;
//   F  the "call | meaning" tables of SEF_2024.md, each row beside the rules
//      whose meaning shares its numbers, for a review by hand;
//   G  the system's thematic deals (pbn/*.pbn): the file's auction against
//      the engine's, call by call;
//   H  the rules that cost the most against the par, over the par bench.
//
// The corpus is the 12 000 deals of the par bench and COHERENCE_DEALS random
// deals (default 20 000, seed COHERENCE_SEED). Every decision is replayed
// against the whole rule list, so that the rules whose sequence and condition
// fit but come too late are seen; the call chosen is checked against the
// engine's own, and the auction against Run().

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// cPattern is one sequence pattern of a rule, split once: the matcher below is
// seqMatches (match.go) without the allocations, so that the whole rule list
// can be walked at every decision of tens of thousands of deals.
type cPattern struct {
	anyStart bool       // ** (or BW:x)
	bw       string     // BW:x : the agreed trump required
	leadP    bool       // the pattern names the initial passes
	alts     [][]string // per token: the alternatives, nil for *
	paren    []bool     // per token: an opponent's call
}

func compilePatterns(r *Rule) []cPattern {
	var out []cPattern
	for _, pat := range r.Seq {
		p := strings.Fields(pat)
		var c cPattern
		if len(p) > 0 && p[0] == "P" {
			c.leadP = true
		}
		if len(p) > 0 && strings.HasPrefix(p[0], "BW:") {
			c.bw = p[0][3:]
			c.anyStart = true
			p = p[1:]
		} else if len(p) > 0 && p[0] == "**" {
			c.anyStart = true
			p = p[1:]
		}
		for _, tok := range p {
			if tok == "*" {
				c.alts = append(c.alts, nil)
				c.paren = append(c.paren, false)
				continue
			}
			c.alts = append(c.alts, strings.Split(strings.Trim(tok, "()"), "|"))
			c.paren = append(c.paren, strings.HasPrefix(tok, "("))
		}
		out = append(out, c)
	}
	return out
}

func (c *cPattern) match(seq, stripped []string, trump string) bool {
	if c.bw != "" && (trump == "" || trump != c.bw) {
		return false
	}
	s := stripped
	if c.leadP {
		s = seq
	}
	if c.anyStart {
		if len(s) < len(c.alts) {
			return false
		}
		s = s[len(s)-len(c.alts):]
	}
	if len(s) != len(c.alts) {
		return false
	}
	for i, alts := range c.alts {
		if alts == nil {
			continue
		}
		st := s[i]
		paren := strings.HasPrefix(st, "(")
		if paren != c.paren[i] {
			return false
		}
		if paren {
			st = st[1 : len(st)-1]
		}
		found := false
		for _, a := range alts {
			if a == st {
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

// coherenceStats gathers the counts over the corpus.
type coherenceStats struct {
	rules    []*Rule
	pats     [][]cPattern
	seqSeen  []int
	condTrue []int
	chosen   []int
	masked   []map[int]int // rule -> earlier rule that took the decision -> times
	gaps     map[string]*gapStat
	passOnF  map[string]int // "rule (pass) after partner's rule" -> times
	illegal  map[string]int
	checked  int // decisions checked against the engine's choose
}

type gapStat struct {
	n, forced, hcp, hl int
}

func newCoherenceStats(rs *RuleSet) *coherenceStats {
	st := &coherenceStats{rules: rs.Rules, gaps: map[string]*gapStat{}, passOnF: map[string]int{}, illegal: map[string]int{}}
	n := len(rs.Rules)
	st.pats = make([][]cPattern, n)
	st.seqSeen, st.condTrue, st.chosen = make([]int, n), make([]int, n), make([]int, n)
	st.masked = make([]map[int]int, n)
	for i, r := range rs.Rules {
		st.pats[i] = compilePatterns(r)
	}
	return st
}

// bid replays Engine.Run, walking every rule at each decision.
func (st *coherenceStats) bid(t *testing.T, rs *RuleSet, d *Deal) []SeatCall {
	var feats [4]*features
	for seat, h := range d.Hands {
		feats[seat] = newFeatures(h)
		feats[seat].vul = d.Vul[sideOf(seat)]
		feats[seat].oppVul = d.Vul[1-sideOf(seat)]
		feats[seat].seat = (seat-d.Dealer+4)%4 + 1
	}
	trump := [2]string{}
	var history []SeatCall
	for i := range maxCalls {
		seat := (d.Dealer + i) % 4
		seq := pairSequence(history, seat)
		stripped := seq
		for len(stripped) > 0 && stripped[0] == "P" {
			stripped = stripped[1:]
		}
		tr := trump[sideOf(seat)]
		chosen := -1
		for idx, r := range st.rules {
			if r.Option != "" {
				continue
			}
			ok := false
			for k := range st.pats[idx] {
				if st.pats[idx][k].match(seq, stripped, tr) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
			st.seqSeen[idx]++
			v, err := r.cond.eval(feats[seat])
			if err != nil || !v.truthy() {
				continue
			}
			st.condTrue[idx]++
			if chosen < 0 {
				chosen = idx
			} else {
				if st.masked[idx] == nil {
					st.masked[idx] = map[int]int{}
				}
				st.masked[idx][chosen]++
			}
		}
		// The engine must have chosen the same rule.
		want := rs.choose(seq, feats[seat], map[string]bool{}, tr)
		if (want == nil) != (chosen < 0) || (want != nil && want != st.rules[chosen]) {
			t.Fatalf("le relevé diverge du moteur après %q", strings.Join(seq, " "))
		}
		st.checked++
		sc := SeatCall{Seat: seat, Call: passCall, Seq: seq}
		switch {
		case chosen < 0:
			sc.Why = noRule
			key := strings.Join(seq, " ")
			if key == "" {
				key = "(ouverture)"
			}
			g := st.gaps[key]
			if g == nil {
				g = &gapStat{}
				st.gaps[key] = g
			}
			g.n++
			g.hcp += feats[seat].hcp
			g.hl += feats[seat].hl
			if n := len(history); n >= 2 && history[n-2].Rule != nil && forcingKinds[history[n-2].Rule.Forcing] &&
				history[n-1].Call.Kind == KindPass {
				g.forced++
			}
		case !legalCall(st.rules[chosen].Call, history, seat):
			sc.Why, sc.Denied = illegalBy, st.rules[chosen]
			st.illegal[st.rules[chosen].ID]++
		default:
			r := st.rules[chosen]
			st.chosen[chosen]++
			sc.Rule, sc.Call = r, r.Call
			if r.Trump != "" {
				trump[sideOf(seat)] = r.Trump
			}
			if n := len(history); r.Call.Kind == KindPass && n >= 2 && history[n-2].Rule != nil &&
				forcingKinds[history[n-2].Rule.Forcing] && history[n-1].Call.Kind == KindPass {
				st.passOnF[fmt.Sprintf("%s après %s (%s)", r.ID, history[n-2].Rule.ID, history[n-2].Rule.Forcing)]++
			}
		}
		history = append(history, sc)
		if auctionOver(history) {
			break
		}
	}
	return history
}

// ---------------------------------------------------------------- D, E

var (
	meanRangeRe = regexp.MustCompile(`(\d+)\s*[-à]\s*(\d+)\s*(HL|H)\b`)
	meanPlusRe  = regexp.MustCompile(`(\d+)\s*\+\s*(HL|H)\b`)
	meanTricks  = regexp.MustCompile(`(\d+(?:[,.]5)?)\s*levées de jeu`)
	partnerCtx  = regexp.MustCompile(`(?i)(face à|à deux|ensemble|partenaire|du répondant|de l'ouvreur)\s*:?\s*$`)
	numRe       = regexp.MustCompile(`\d+`)
	hcpInRe     = regexp.MustCompile(`hcp_in\('[SHDC]'\)\s*(?:>=|<=|>|<|==)\s*(\d+)`)
	frenchRe    = regexp.MustCompile(`[éèêàùçâîô]|\b(avec|sans|ou|et|cartes|levées|couleur|main)\b`)
)

// condBounds collects the numbers a condition compares a feature with.
func condBounds(cond, feat string) map[string]bool {
	out := map[string]bool{}
	f := regexp.QuoteMeta(feat)
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`\b` + f + `\s*(?:>=|<=|>|<|==)\s*(\d+(?:\.\d+)?)`),
		regexp.MustCompile(`(\d+(?:\.\d+)?)\s*(?:>=|<=|>|<|==)\s*` + f + `\b`),
	} {
		for _, m := range re.FindAllStringSubmatch(cond, -1) {
			out[m[1]] = true
		}
	}
	return out
}

// enPointsRe: a points range or minimum of the English meaning, « 12-14 »,
// « 8+ », whatever its unit.
var enPointsRe = regexp.MustCompile(`(\d+)\s*(?:-|to)\s*(\d+)|(\d+)\s*\+`)

// pointsTexts: the ranges and minimums of a meaning, as « 12-14 » and « 8+ ».
func pointsTexts(s string, fr bool) map[string]bool {
	out := map[string]bool{}
	if fr {
		for _, m := range meanRangeRe.FindAllStringSubmatch(s, -1) {
			out[m[1]+"-"+m[2]] = true
		}
		for _, m := range meanPlusRe.FindAllStringSubmatch(s, -1) {
			out[m[1]+"+"] = true
		}
		return out
	}
	for _, m := range enPointsRe.FindAllStringSubmatch(s, -1) {
		if m[3] != "" {
			out[m[3]+"+"] = true
		} else {
			out[m[1]+"-"+m[2]] = true
		}
	}
	return out
}

// meaningIssues: a points range or minimum of the meaning that no bound of
// the condition takes up. H and HL are read together (a meaning in H often
// stands for a condition in hl, and « (4+ H) » for hcp_in), and one end of a
// range is enough: the other is often implied by the sequence. Points meant
// for the partner (« face à 18-21 ») are left out.
func meaningIssues(r *Rule) []string {
	var out []string
	cond := r.CondText
	pts := map[string]bool{}
	for _, f := range []string{"hcp", "hl", "hld", "dh"} {
		for k := range condBounds(cond, f) {
			pts[k] = true
		}
	}
	for _, m := range hcpInRe.FindAllStringSubmatch(cond, -1) {
		pts[m[1]] = true
	}
	seen := map[string]bool{}
	flag := func(text, issue string) {
		if !seen[text] {
			seen[text] = true
			out = append(out, fmt.Sprintf("« %s » : %s", text, issue))
		}
	}
	boundsList := func(b map[string]bool) string {
		keys := make([]string, 0, len(b))
		for k := range b {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if len(keys) == 0 {
			return "aucune borne"
		}
		return "bornes " + strings.Join(keys, ", ")
	}
	for _, m := range meanRangeRe.FindAllStringSubmatchIndex(r.Meaning, -1) {
		if partnerCtx.MatchString(r.Meaning[:m[0]]) {
			continue
		}
		if lo, hi := r.Meaning[m[2]:m[3]], r.Meaning[m[4]:m[5]]; !pts[lo] && !pts[hi] {
			flag(r.Meaning[m[0]:m[1]], "points de la condition : "+boundsList(pts))
		}
	}
	for _, m := range meanPlusRe.FindAllStringSubmatchIndex(r.Meaning, -1) {
		if partnerCtx.MatchString(r.Meaning[:m[0]]) {
			continue
		}
		if !pts[r.Meaning[m[2]:m[3]]] {
			flag(r.Meaning[m[0]:m[1]], "points de la condition : "+boundsList(pts))
		}
	}
	for _, m := range meanTricks.FindAllStringSubmatchIndex(r.Meaning, -1) {
		n := strings.Replace(r.Meaning[m[2]:m[3]], ",", ".", 1)
		if b := condBounds(cond, "ptricks"); !b[n] {
			flag(r.Meaning[m[0]:m[1]], "ptricks de la condition : "+boundsList(b))
		}
	}
	en := pointsTexts(r.MeaningEN, false)
	var missing []string
	for p := range pointsTexts(r.Meaning, true) {
		if !en[p] {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		out = append(out, fmt.Sprintf("tranche absente de l'anglais : %s (« %s »)", strings.Join(missing, ", "), r.MeaningEN))
	}
	return out
}

func translationIssue(r *Rule) string {
	switch {
	case strings.TrimSpace(r.MeaningEN) == "":
		return "meaning_en vide"
	case r.MeaningEN == r.Meaning && frenchRe.MatchString(r.Meaning):
		return "meaning_en identique au français"
	case frenchRe.MatchString(r.MeaningEN):
		return "meaning_en en français ? « " + r.MeaningEN + " »"
	}
	return ""
}

// ---------------------------------------------------------------- F

var (
	headingRe  = regexp.MustCompile(`^(#+)\s+(.*)`)
	tableRowRe = regexp.MustCompile(`^\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|`)
	callCellRe = regexp.MustCompile(`([1-7])\s*(♣|♦|♥|♠|SA)|\b(Passe|Contre|Surcontre)\b`)
)

var suitCode = map[string]string{"♣": "C", "♦": "D", "♥": "H", "♠": "S", "SA": "NT"}

func specTables(rules []*Rule, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "fiche introuvable : " + err.Error() + "\n"
	}
	var b strings.Builder
	heading := ""
	for _, line := range strings.Split(string(data), "\n") {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			heading = m[2]
			continue
		}
		m := tableRowRe.FindStringSubmatch(line)
		if m == nil || strings.HasPrefix(m[1], "---") {
			continue
		}
		var calls []string
		for _, c := range callCellRe.FindAllStringSubmatch(m[1], -1) {
			switch {
			case c[1] != "":
				calls = append(calls, c[1]+suitCode[c[2]])
			case c[3] == "Passe":
				calls = append(calls, "P")
			case c[3] == "Contre":
				calls = append(calls, "X")
			default:
				calls = append(calls, "XX")
			}
		}
		nums := numRe.FindAllString(m[2], -1)
		if len(calls) == 0 || len(nums) == 0 {
			continue
		}
		// Candidates: the rules giving one of these calls whose meaning shares
		// the most numbers with the row.
		type cand struct {
			r     *Rule
			score int
		}
		var cs []cand
		for _, r := range rules {
			ok := false
			for _, c := range calls {
				if r.CallText == c {
					ok = true
				}
			}
			if !ok {
				continue
			}
			rn := map[string]bool{}
			for _, n := range numRe.FindAllString(r.Meaning, -1) {
				rn[n] = true
			}
			s := 0
			for _, n := range nums {
				if rn[n] {
					s++
				}
			}
			if s > 0 {
				cs = append(cs, cand{r, s})
			}
		}
		sort.SliceStable(cs, func(i, j int) bool { return cs[i].score > cs[j].score })
		fmt.Fprintf(&b, "| %s | %s | %s | ", heading, m[1], m[2])
		if len(cs) == 0 {
			b.WriteString("**aucune règle aux mêmes chiffres**")
		}
		for i, c := range cs {
			if i == 3 {
				break
			}
			if i > 0 {
				b.WriteString("<br>")
			}
			fmt.Fprintf(&b, "`%s` (%s) : %s", c.r.ID, strings.Join(c.r.Seq, " ; "), c.r.Meaning)
		}
		b.WriteString(" |\n")
	}
	return b.String()
}

// ---------------------------------------------------------------- G

var (
	pbnAuctionRe = regexp.MustCompile(`(?s)\[Auction\s+"([NESW])"\]\s*(.*?)(?:\n\s*\n|\n\[|\z)`)
	pbnCommentRe = regexp.MustCompile(`(?s)\{[^}]*\}`)
)

// fileAuction reads the calls of a PBN block, comments left out.
func fileAuction(block string) []string {
	m := pbnAuctionRe.FindStringSubmatch(block)
	if m == nil {
		return nil
	}
	var out []string
	for _, tok := range strings.Fields(pbnCommentRe.ReplaceAllString(m[2], " ")) {
		if strings.HasPrefix(tok, "=") || tok == "*" || tok == "AP" {
			continue
		}
		out = append(out, strings.ToUpper(tok))
	}
	return out
}

func themeReport(t *testing.T, dir string) (string, int, int) {
	files, _ := filepath.Glob(filepath.Join(dir, "*.pbn"))
	sort.Strings(files)
	var b strings.Builder
	total, bad := 0, 0
	fmt.Fprintf(&b, "| Thème | Donnes | Identiques | Écart à l'ouverture | Écart ensuite |\n|---|---|---|---|---|\n")
	var detail strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		deals, err := ParsePBNBoards(data)
		if err != nil {
			t.Fatalf("%s : %v", f, err)
		}
		blocks := strings.Split(string(data), "[Event ")[1:]
		same, atOpen, later := 0, 0, 0
		name := filepath.Base(f)
		for i, d := range deals {
			if i >= len(blocks) {
				break
			}
			want := fileAuction(blocks[i])
			calls := NewEngine(d).Run()
			got := make([]string, len(calls))
			for k, c := range calls {
				got[k] = strings.ToUpper(c.Call.Format("en"))
			}
			k := 0
			for k < len(want) && k < len(got) && want[k] == got[k] {
				k++
			}
			if k == len(want) && k == len(got) {
				same++
				continue
			}
			opening := true
			for _, c := range want[:k] {
				if c != "PASS" {
					opening = false
				}
			}
			if opening {
				atOpen++
			} else {
				later++
			}
			fileCall, engCall, rule := "(fin)", "(fin)", ""
			if k < len(want) {
				fileCall = want[k]
			}
			if k < len(got) {
				engCall = got[k]
				if calls[k].Rule != nil {
					rule = calls[k].Rule.ID
				} else {
					rule = "passe par défaut"
				}
			}
			seat := "?"
			if k < len(calls) {
				seat = seatNames[calls[k].Seat]
			}
			fmt.Fprintf(&detail, "| %s | %d | %d | %s | %s | %s | `%s` |\n", name, i+1, k+1, seat, fileCall, engCall, rule)
		}
		total += len(deals)
		bad += len(deals) - same
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d |\n", name, len(deals), same, atOpen, later)
	}
	b.WriteString("\n#### Premier appel divergent\n\n| Thème | Donne | Appel n° | Siège | Fichier | Moteur | Règle du moteur |\n|---|---|---|---|---|---|---|\n")
	b.WriteString(detail.String())
	return b.String(), total, bad
}

// ---------------------------------------------------------------- report

type coherenceSummary struct {
	System        string         `json:"system"`
	Deals         int            `json:"deals"`
	Decisions     int            `json:"decisions"`
	Rules         int            `json:"rules"`
	NeverChosen   int            `json:"neverChosen"`
	Unreachable   int            `json:"unreachable"`
	CondNever     int            `json:"condNever"`
	Shadowed      int            `json:"shadowed"`
	OptionRules   int            `json:"optionRules"`
	GapSequences  int            `json:"gapSequences"`
	PassOnForcing int            `json:"passOnForcing"`
	MeaningIssues int            `json:"meaningIssues"`
	Translations  int            `json:"translations"`
	ThemeDeals    int            `json:"themeDeals"`
	ThemeDiffs    int            `json:"themeDiffs"`
	Illegal       map[string]int `json:"illegal,omitempty"`
}

func TestCoherenceReport(t *testing.T) {
	out := os.Getenv("COHERENCE_OUT")
	if out == "" {
		t.Skip("set COHERENCE_OUT to a folder to write the coherence report (see tools/coherence/README.md)")
	}
	// Un chemin relatif se lit depuis la racine du dépôt, pas depuis engine/.
	if !filepath.IsAbs(out) {
		out = filepath.Join("..", out)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	extra := int(envInt(t, "COHERENCE_DEALS", 20000))
	seed := envInt(t, "COHERENCE_SEED", 20261005)
	bench := loadBench(t)
	forEachSystem(t, func(t *testing.T, s *testSystem) {
		writeCoherence(t, s, bench, extra, seed, out)
	})
}

func writeCoherence(t *testing.T, s *testSystem, bench []benchDeal, extra int, seed int64, out string) {
	rs := s.Rules
	st := newCoherenceStats(rs)
	sum := coherenceSummary{System: s.ID, Rules: len(rs.Rules)}

	// Corpus 1: the par bench, with the cost of each deal (H).
	cost := map[string]int{}
	costCats := map[string]map[int]int{}
	costDeals := map[string]int{}
	for i := range bench {
		b := &bench[i]
		d := mustParsePBN(t, fmt.Sprintf("[Dealer %q]\n[Vulnerable %q]\n[Deal %q]\n", b.Dealer, b.Vul, b.Deal))
		calls := st.bid(t, rs, d)
		if got, want := formatAuction(calls), formatAuction(NewEngine(d).Run()); got != want {
			t.Fatalf("donne %s : le relevé donne %s, le moteur %s", b.ID, got, want)
		}
		r := replay(t, b)
		if r.IMP == 0 || r.Declarer < 0 {
			continue
		}
		key := "(aucune règle)"
		for k := len(calls) - 1; k >= 0; k-- {
			if sideOf(calls[k].Seat) == sideOf(r.Declarer) && calls[k].Rule != nil && calls[k].Call.Kind != KindPass {
				key = calls[k].Rule.ID
				break
			}
		}
		cost[key] += max(r.IMP, -r.IMP)
		costDeals[key]++
		if costCats[key] == nil {
			costCats[key] = map[int]int{}
		}
		for _, c := range r.Cats {
			costCats[key][c]++
		}
	}
	// Corpus 2: random deals, dealer and vulnerability by board number.
	rng := rand.New(rand.NewSource(seed))
	for board := 1; board <= extra; board++ {
		d := randomDeal(rng)
		d.Dealer = (board - 1) % 4
		d.Vul = boardVul[(board-1)%16]
		st.bid(t, rs, d)
	}
	sum.Deals = len(bench) + extra
	sum.Decisions = st.checked
	sum.Illegal = st.illegal

	var md strings.Builder
	fmt.Fprintf(&md, "# Rapport de cohérence — %s\n\n", s.ID)
	fmt.Fprintf(&md, "%d donnes (%d du banc du par, %d au hasard, graine %d), %d décisions, %d règles.\n\n",
		sum.Deals, len(bench), extra, seed, st.checked, len(rs.Rules))

	// A
	var unreach, never, shadow, options []string
	for i, r := range rs.Rules {
		if r.Option != "" {
			options = append(options, fmt.Sprintf("| `%s` | %s |", r.ID, r.Option))
			continue
		}
		if st.chosen[i] > 0 {
			continue
		}
		switch {
		case st.seqSeen[i] == 0:
			unreach = append(unreach, fmt.Sprintf("| `%s` | %s | %s |", r.ID, strings.Join(r.Seq, " ; "), r.CallText))
		case st.condTrue[i] == 0:
			never = append(never, fmt.Sprintf("| `%s` | %s | %s | %d | `%s` |", r.ID, strings.Join(r.Seq, " ; "), r.CallText, st.seqSeen[i], r.CondText))
		default:
			top, n := -1, 0
			for k, v := range st.masked[i] {
				if v > n || v == n && k < top {
					top, n = k, v
				}
			}
			masker := "?"
			if top >= 0 {
				masker = rs.Rules[top].ID
			}
			shadow = append(shadow, fmt.Sprintf("| `%s` | %s | %d | `%s` (%d fois) |", r.ID, r.CallText, st.condTrue[i], masker, n))
		}
	}
	sum.Unreachable, sum.CondNever, sum.Shadowed, sum.OptionRules = len(unreach), len(never), len(shadow), len(options)
	sum.NeverChosen = len(unreach) + len(never) + len(shadow)
	fmt.Fprintf(&md, "## A. Règles jamais choisies (%d)\n\n", sum.NeverChosen)
	fmt.Fprintf(&md, "### A1. Toujours masquées par une règle antérieure (%d)\n\nSéquence et condition vraies, mais une règle placée avant a toujours pris la décision.\n\n| Règle | Appel | Fois applicable | Masquée par |\n|---|---|---|---|\n%s\n\n", len(shadow), strings.Join(shadow, "\n"))
	fmt.Fprintf(&md, "### A2. Condition jamais vraie (%d)\n\nLa séquence a été rencontrée, la condition jamais remplie.\n\n| Règle | Séquence | Appel | Séquence vue | Condition |\n|---|---|---|---|---|\n%s\n\n", len(never), strings.Join(never, "\n"))
	fmt.Fprintf(&md, "### A3. Séquence jamais rencontrée (%d)\n\n| Règle | Séquence | Appel |\n|---|---|---|\n%s\n\n", len(unreach), strings.Join(unreach, "\n"))
	fmt.Fprintf(&md, "### A4. Règles d'option, non évaluées (%d)\n\n| Règle | Option |\n|---|---|\n%s\n\n", len(options), strings.Join(options, "\n"))

	// B
	type gapRow struct {
		key string
		g   *gapStat
	}
	var gaps []gapRow
	for k, g := range st.gaps {
		gaps = append(gaps, gapRow{k, g})
	}
	sum.GapSequences = len(gaps)
	sort.Slice(gaps, func(i, j int) bool { return gaps[i].g.n > gaps[j].g.n || gaps[i].g.n == gaps[j].g.n && gaps[i].key < gaps[j].key })
	writeGaps := func(title string, keep func(*gapStat) bool) {
		fmt.Fprintf(&md, "%s\n\n| Séquence de la paire | Fois | H moyens | HL moyens | Après un forcing |\n|---|---|---|---|---|\n", title)
		n := 0
		for _, g := range gaps {
			if !keep(g.g) {
				continue
			}
			fmt.Fprintf(&md, "| `%s` | %d | %.1f | %.1f | %d |\n", g.key, g.g.n, float64(g.g.hcp)/float64(g.g.n), float64(g.g.hl)/float64(g.g.n), g.g.forced)
			if n++; n == 50 {
				break
			}
		}
		md.WriteString("\n")
	}
	fmt.Fprintf(&md, "## B. Séquences sans règle (%d séquences distinctes)\n\nLe joueur passe par défaut. La plupart sont normales (rien à dire) ; à regarder : les mains fortes et les suites de forcing.\n\n", len(gaps))
	writeGaps("### B1. Les plus fréquentes", func(*gapStat) bool { return true })
	writeGaps("### B2. Mains de 12 H et plus en moyenne", func(g *gapStat) bool { return g.hcp >= 12*g.n })
	writeGaps("### B3. Juste après un forcing du partenaire", func(g *gapStat) bool { return g.forced > 0 })

	// C
	type kv struct {
		k string
		v int
	}
	sortKV := func(m map[string]int) []kv {
		var o []kv
		for k, v := range m {
			o = append(o, kv{k, v})
		}
		sort.Slice(o, func(i, j int) bool { return o[i].v > o[j].v || o[i].v == o[j].v && o[i].k < o[j].k })
		return o
	}
	pof := sortKV(st.passOnF)
	sum.PassOnForcing = len(pof)
	fmt.Fprintf(&md, "## C. Passe donné par une règle sur un forcing du partenaire (%d cas)\n\n| Règle (passe) après la règle forcing | Fois |\n|---|---|\n", len(pof))
	for _, e := range pof {
		fmt.Fprintf(&md, "| %s | %d |\n", e.k, e.v)
	}
	md.WriteString("\n")

	// D, E
	var dRows, eRows []string
	for _, r := range rs.Rules {
		for _, issue := range meaningIssues(r) {
			dRows = append(dRows, fmt.Sprintf("| `%s` | %s | `%s` | %s |", r.ID, r.Meaning, r.CondText, issue))
		}
		if e := translationIssue(r); e != "" {
			eRows = append(eRows, fmt.Sprintf("| `%s` | %s | %s |", r.ID, r.Meaning, e))
		}
	}
	sum.MeaningIssues, sum.Translations = len(dRows), len(eRows)
	fmt.Fprintf(&md, "## D. Libellé contre condition (%d)\n\n| Règle | Libellé | Condition | Constat |\n|---|---|---|---|\n%s\n\n", len(dRows), strings.Join(dRows, "\n"))
	fmt.Fprintf(&md, "## E. Traductions (%d)\n\n| Règle | Libellé | Constat |\n|---|---|---|\n%s\n\n", len(eRows), strings.Join(eRows, "\n"))

	// F
	fmt.Fprintf(&md, "## F. Fiche SEF : tableaux et règles aux mêmes chiffres\n\n| Section | Enchère | Fiche | Règles candidates |\n|---|---|---|---|\n%s\n",
		specTables(rs.Rules, "../tools/python_tools/SEF_2024.md"))

	// G
	themes, tDeals, tBad := themeReport(t, s.file("pbn"))
	sum.ThemeDeals, sum.ThemeDiffs = tDeals, tBad
	fmt.Fprintf(&md, "## G. Thèmes PBN : enchère du fichier contre enchère du moteur (%d donnes, %d écarts)\n\n%s\n", tDeals, tBad, themes)

	// H
	fmt.Fprintf(&md, "## H. Règles qui coûtent le plus au banc du par\n\nIMP (valeur absolue) des donnes du banc, attribués à la dernière enchère de la paire déclarante. Catégories : 1 camp inversé, 2 couleur, 3 chelem impossible, 4 chelem manqué, 5 manche impossible, 6 manche manquée.\n\n| Règle | IMP | Donnes | Catégories |\n|---|---|---|---|\n")
	for i, e := range sortKV(cost) {
		if i == 40 {
			break
		}
		var cats []string
		for c := 1; c <= 6; c++ {
			if n := costCats[e.k][c]; n > 0 {
				cats = append(cats, fmt.Sprintf("%d×%d", c, n))
			}
		}
		fmt.Fprintf(&md, "| `%s` | %d | %d | %s |\n", e.k, e.v, costDeals[e.k], strings.Join(cats, " "))
	}

	if err := os.WriteFile(filepath.Join(out, "coherence-"+s.ID+".md"), []byte(md.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	js, _ := json.MarshalIndent(sum, "", "  ")
	if err := os.WriteFile(filepath.Join(out, "coherence-"+s.ID+".json"), js, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%s : %d règles jamais choisies, %d séquences sans règle, %d passes sur forcing, %d libellés, %d traductions, %d/%d donnes de thème en écart",
		s.ID, sum.NeverChosen, sum.GapSequences, sum.PassOnForcing, sum.MeaningIssues, sum.Translations, tBad, tDeals)
}
