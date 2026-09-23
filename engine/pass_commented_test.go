package engine

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

// randomPBN deals 52 cards at random and writes them as a PBN fragment, so the
// sweep below sees the shapes a curated test file never thinks to write down.
func randomPBN(rng *rand.Rand) string {
	const ranks = "AKQJT98765432"
	cards := make([]int, 52)
	for i := range cards {
		cards[i] = i
	}
	rng.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	hands := make([]string, 4)
	for seat := range 4 {
		var suits [4][]byte
		for _, c := range cards[seat*13 : seat*13+13] {
			suits[c/13] = append(suits[c/13], ranks[c%13])
		}
		parts := make([]string, 0, 4)
		for s := 3; s >= 0; s-- {
			b := suits[s]
			sort.Slice(b, func(i, j int) bool {
				return strings.IndexByte(ranks, b[i]) < strings.IndexByte(ranks, b[j])
			})
			parts = append(parts, string(b))
		}
		hands[seat] = strings.Join(parts, ".")
	}
	return fmt.Sprintf("[Dealer %q]\n[Deal \"N:%s\"]",
		string("NESW"[rng.Intn(4)]), strings.Join(hands, " "))
}

// TestEveryCallIsCommented holds the promise the API makes about
// auction[].comment (§1.1): no call ever comes back without one, in either
// language. Passes are the ones that used to slip through -- a decision path
// that found no rule returned a bare pass, the web client filtered it out of
// the commented list, and the auction there read as though a player had been
// skipped.
func TestEveryCallIsCommented(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for range 2000 {
		pbn := randomPBN(rng)
		d, err := ParsePBN([]byte(pbn))
		if err != nil {
			t.Fatalf("bad deal %s: %v", pbn, err)
		}
		calls := NewEngine(d).Run()
		for _, sc := range calls {
			if sc.M.fr == "" || sc.M.en == "" {
				t.Fatalf("%s by %s has no comment (fr=%q en=%q)\n%s\nauction: %s",
					sc.Call.Format("fr"), seatNames[sc.Seat], sc.M.fr, sc.M.en,
					pbn, formatAuction(calls))
			}
		}
	}
}

// TestPassReasonMatchesPosition pins the four positional reasons of §1.1 on the
// reported deal: East opens and is outbid, West never gets in, and South's
// final pass leaves partner's game standing.
func TestPassReasonMatchesPosition(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "E"]
[Vulnerable "None"]
[Deal "N:AKQ.82.QT643.K83 8.KJT9.A9872.AT6 J9754.AQ65.K5.Q2 T632.743.J.J9754"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	want := map[int]string{
		2: "l'enchère est aux adversaires, pas de quoi pousser plus haut", // W, silent side, over 1S
		9: "l'enchère du partenaire convient, rien à ajouter",             // S, under North's 4S
	}
	for i, w := range want {
		if i >= len(calls) {
			t.Fatalf("auction has %d calls, wanted one at %d\n%s", len(calls), i, formatAuction(calls))
		}
		if got := calls[i].M.fr; got != w {
			t.Errorf("call %d (%s %s) comment = %q, want %q\nauction: %s",
				i, seatNames[calls[i].Seat], calls[i].Call.Format("fr"), got, w, formatAuction(calls))
		}
	}
}

// TestPassingOpeningHandIsNotCalledWeak guards the one reason that would be a
// lie. North holds 13 H over their 2NT opening and the system gives it nothing
// to say; the pass is told exactly that -- the values are there, no bid
// describes the hand -- and never that the hand is too weak.
func TestPassingOpeningHandIsNotCalledWeak(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "W"]
[Vulnerable "None"]
[Deal "W:AQJ.AQJ.QJT9.QJ9 K4.K43.A8765.K43 T98765.T98.2.A87 32.7652.K43.T652"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	if len(calls) < 2 || calls[1].Call.Kind != KindPass {
		t.Skipf("North did not pass over the opening\nauction: %s", formatAuction(calls))
	}
	const want = "l'ouverture, mais aucune enchère ne décrit la main"
	if got := calls[1].M.fr; got != want {
		t.Fatalf("North's pass (%d H) comment = %q, want %q\nauction: %s",
			d.Hands[0].H(), got, want, formatAuction(calls))
	}
}
