package engine

// Non-regression bench against the par (see tools/par/README.md, « Banc de
// non-régression »). The double-dummy trick table and the par of a deal depend
// on the cards alone, never on the engine: tools/par/audit.js computed them
// once, tools/par/bench-export.js stored them in testdata/par_bench.jsonl.gz,
// and this test only replays the auctions. Twelve thousand deals take a few
// seconds instead of the half hour the solver needs.
//
// The score is the total distance to the par, in IMPs, over the whole bench.
// The engine is deterministic, so the figure is exact: the test fails as soon
// as it grows beyond testdata/par_bench_baseline.json. A lower figure passes,
// with a reminder to lock the gain in (PAR_BENCH_UPDATE=1 rewrites the
// baseline). PAR_BENCH_OUT=<file> writes the replayed deals in the par.json
// shape tools/par/compare.js reads, so two engine versions compare in seconds.

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

const (
	parBenchData     = "testdata/par_bench.jsonl.gz"
	parBenchBaseline = "testdata/par_bench_baseline.json"
)

// benchDeal is one line of the bench: the deal and everything that follows
// from the cards alone. dd is DDS's res_table: strain-major (S H D C NT), then
// seat (N E S W). par is seen from North-South, as in audit.js.
type benchDeal struct {
	ID     string `json:"id"`
	Dealer string `json:"dealer"`
	Vul    string `json:"vul"`
	Deal   string `json:"deal"`
	DD     [20]int
	Par    struct {
		Score  int `json:"score"`
		Level  int `json:"level"`
		Strain int `json:"strain"`
		Side   int `json:"side"`
	} `json:"par"`
}

func (b *benchDeal) UnmarshalJSON(data []byte) error {
	type plain benchDeal
	aux := struct {
		*plain
		DD []int `json:"dd"`
	}{plain: (*plain)(b)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.DD) != 20 {
		return fmt.Errorf("deal %s: %d double-dummy entries, want 20", b.ID, len(aux.DD))
	}
	copy(b.DD[:], aux.DD)
	return nil
}

// benchResult is one replayed deal, in the field names of audit.js's par.json
// so that tools/par/compare.js reads it as is.
type benchResult struct {
	Board    string `json:"board"`
	PBN      string `json:"pbn"`
	Auction  string `json:"auction"`
	Level    int    `json:"level"`
	Strain   int    `json:"strain"`
	Declarer int    `json:"declarer"`
	Doubled  bool   `json:"doubled"`
	Score    int    `json:"simScore"`
	ParScore int    `json:"parScore"`
	IMP      int    `json:"imp"`
	Cats     []int  `json:"cats"`
}

// benchSummary is what the baseline stores: the total the test gates on, and
// the six audit categories for information.
type benchSummary struct {
	Deals    int     `json:"deals"`
	TotalIMP int     `json:"totalImp"`
	MeanIMP  float64 `json:"meanImp"`
	Cats     [6]int  `json:"cats"`
}

// ddsStrain maps the engine's strain (C D H S NT) to DDS's row (S H D C NT).
var ddsStrain = [5]int{3, 2, 1, 0, 4}

func ddTricks(dd [20]int, strain Strain, seat int) int {
	return dd[ddsStrain[strain]*4+seat]
}

// sideTricks is the best of the side's two seats: the side picks its declarer.
func sideTricks(dd [20]int, strain Strain, side int) int {
	return max(ddTricks(dd, strain, side), ddTricks(dd, strain, side+2))
}

// playedScore is par.js's score(): the result of a contract for the side that
// plays it, overtricks, undoubled undertricks and the insult included.
func playedScore(level int, strain Strain, tricks int, vul, doubled bool) int {
	needed := level + 6
	if tricks < needed {
		down := needed - tricks
		switch {
		case !doubled && vul:
			return -100 * down
		case !doubled:
			return -50 * down
		case vul:
			return -(200 + 300*(down-1))
		case down == 1:
			return -100
		case down == 2:
			return -300
		default:
			return -(500 + 300*(down-3))
		}
	}
	ts := 20 * level
	switch {
	case strain == SNoTrump:
		ts = 40 + 30*(level-1)
	case strain >= SHearts:
		ts = 30 * level
	}
	over := 20
	if strain >= SHearts {
		over = 30
	}
	if doubled {
		ts *= 2
		over = 100
		if vul {
			over = 200
		}
	}
	total := ts + (tricks-needed)*over
	switch {
	case ts >= 100 && vul:
		total += 500
	case ts >= 100:
		total += 300
	default:
		total += 50
	}
	if level >= 6 {
		total += map[bool]int{false: 500, true: 750}[vul]
	}
	if level >= 7 {
		total += map[bool]int{false: 500, true: 750}[vul]
	}
	if doubled {
		total += 50
	}
	return total
}

// impTable holds the lower bound of each IMP step (par.js IMP_TABLE).
var impTable = []int{0, 10, 40, 80, 120, 160, 210, 260, 310, 360, 420, 490, 590, 740, 890,
	1090, 1290, 1490, 1740, 1990, 2240, 2490, 2990, 3490, 3990}

func imps(diff int) int {
	a := diff
	if a < 0 {
		a = -a
	}
	i := 0
	for i < len(impTable)-1 && a >= impTable[i+1] {
		i++
	}
	if diff < 0 {
		return -i
	}
	return i
}

// isGameContract: 3NT, four of a major, five of a minor, below slam.
func isGameContract(level int, strain Strain) bool {
	if level == 0 || level >= 6 {
		return false
	}
	switch {
	case strain == SNoTrump:
		return level >= 3
	case strain >= SHearts:
		return level >= 4
	}
	return level >= 5
}

// benchCategories is audit.js's classify(): the six ways the final contract
// can stand apart from the par. A deal may fall in several.
func benchCategories(b *benchDeal, level int, strain Strain, side, result int) []int {
	gameTricks := [5]int{11, 11, 10, 10, 9}
	canGame, canSlam := false, false
	for s := range 2 {
		for st := SClubs; st <= SNoTrump; st++ {
			t := sideTricks(b.DD, st, s)
			canGame = canGame || t >= gameTricks[st]
			canSlam = canSlam || t >= 12
		}
	}
	var cats []int
	played := level > 0
	if played && b.Par.Level > 0 && side != b.Par.Side {
		cats = append(cats, 1)
	}
	if played && b.Par.Level > 0 && int(strain) != b.Par.Strain {
		cats = append(cats, 2)
	}
	if level >= 6 && result < 0 {
		cats = append(cats, 3)
	}
	if level < 6 && canSlam {
		cats = append(cats, 4)
	}
	if isGameContract(level, strain) && result < 0 {
		cats = append(cats, 5)
	}
	if !isGameContract(level, strain) && level < 6 && canGame {
		cats = append(cats, 6)
	}
	return cats
}

func loadBench(t *testing.T) []benchDeal {
	t.Helper()
	f, err := os.Open(parBenchData)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var deals []benchDeal
	sc := bufio.NewScanner(zr)
	for sc.Scan() {
		var b benchDeal
		if err := json.Unmarshal(sc.Bytes(), &b); err != nil {
			t.Fatalf("%s: %v", parBenchData, err)
		}
		deals = append(deals, b)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	return deals
}

// replay bids one bench deal and scores the contract against the par.
func replay(t *testing.T, b *benchDeal) benchResult {
	t.Helper()
	d := mustParsePBN(t, fmt.Sprintf("[Dealer %q]\n[Vulnerable %q]\n[Deal %q]\n", b.Dealer, b.Vul, b.Deal))
	calls := NewEngine(d).Run()
	contract, declarer, doubled := finalContract(calls)
	r := benchResult{Board: b.ID, PBN: b.Deal, Auction: formatAuction(calls),
		Declarer: declarer, Doubled: doubled, ParScore: b.Par.Score}
	side, result := -1, 0
	if declarer >= 0 && contract.IsBid() {
		r.Level, r.Strain = contract.Level, int(contract.Strain)
		side = sideOf(declarer)
		tricks := ddTricks(b.DD, contract.Strain, declarer)
		result = tricks - (contract.Level + 6)
		s := playedScore(contract.Level, contract.Strain, tricks, d.Vul[side], doubled)
		if side == 1 {
			s = -s // seen from North-South, as the par
		}
		r.Score = s
	}
	r.IMP = imps(r.Score - r.ParScore)
	r.Cats = benchCategories(b, r.Level, Strain(r.Strain), side, result)
	if r.Cats == nil {
		r.Cats = []int{}
	}
	return r
}

func summarise(results []benchResult) benchSummary {
	s := benchSummary{Deals: len(results)}
	for _, r := range results {
		s.TotalIMP += max(r.IMP, -r.IMP)
		for _, c := range r.Cats {
			s.Cats[c-1]++
		}
	}
	if s.Deals > 0 {
		s.MeanIMP = float64(int(float64(s.TotalIMP)/float64(s.Deals)*1000+0.5)) / 1000
	}
	return s
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParBenchmark(t *testing.T) {
	deals := loadBench(t)
	results := make([]benchResult, len(deals))
	for i := range deals {
		results[i] = replay(t, &deals[i])
	}
	got := summarise(results)
	catNames := [6]string{"camp inversé", "couleur différente", "chelem impossible",
		"chelem manqué", "manche impossible", "manche manquée"}
	describe := func(s benchSummary) string {
		var b strings.Builder
		fmt.Fprintf(&b, "%d donnes, %d IMP (%.3f par donne)", s.Deals, s.TotalIMP, s.MeanIMP)
		for i, n := range s.Cats {
			fmt.Fprintf(&b, "\n  %d %-20s %5d", i+1, catNames[i], n)
		}
		return b.String()
	}
	t.Log(describe(got))

	if out := os.Getenv("PAR_BENCH_OUT"); out != "" {
		writeJSON(t, out, map[string]any{"meta": map[string]any{"source": "par_bench"}, "deals": results})
		t.Logf("donnes rejouées écrites dans %s", out)
	}
	if os.Getenv("PAR_BENCH_UPDATE") != "" {
		writeJSON(t, parBenchBaseline, got)
		t.Logf("référence réécrite : %s", parBenchBaseline)
		return
	}

	data, err := os.ReadFile(parBenchBaseline)
	if err != nil {
		t.Fatalf("%v (PAR_BENCH_UPDATE=1 crée la référence)", err)
	}
	var want benchSummary
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if want.Deals != got.Deals {
		t.Fatalf("la référence porte sur %d donnes, le banc en compte %d : relancer avec PAR_BENCH_UPDATE=1",
			want.Deals, got.Deals)
	}
	switch {
	case got.TotalIMP > want.TotalIMP:
		t.Fatalf("l'écart au par augmente : %d → %d IMP (+%d)\nréférence : %s\nactuel : %s\n"+
			"pour voir les donnes qui changent : PAR_BENCH_OUT=… sur les deux versions, puis tools/par/compare.js ;\n"+
			"une dégradation voulue se valide avec PAR_BENCH_UPDATE=1 (visible dans le diff de la PR)",
			want.TotalIMP, got.TotalIMP, got.TotalIMP-want.TotalIMP, describe(want), describe(got))
	case got.TotalIMP < want.TotalIMP:
		t.Logf("l'écart au par baisse : %d → %d IMP (%d). Verrouiller le gain : PAR_BENCH_UPDATE=1 go test -run TestParBenchmark ./engine",
			want.TotalIMP, got.TotalIMP, got.TotalIMP-want.TotalIMP)
	}
}
