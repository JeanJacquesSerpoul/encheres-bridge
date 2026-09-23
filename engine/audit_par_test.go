package engine

// Dump half of the par audit (see tools/par/): a batch of seeded random
// boards, each with the auction the engine produces and the contract it
// settles in, written as JSON lines. The double-dummy half -- trick table,
// par contract, classification, report -- is done by tools/par/audit.js,
// which reads this file.
//
// This is a harness, not an assertion: it is skipped unless PAR_AUDIT_OUT
// names an output file, so `go test ./...` is unaffected.

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

// auctionRow is one board: the deal itself, the auction, and the contract it
// ended in. Strain follows the engine's own order (0=C 1=D 2=H 3=S 4=NT);
// declarer is a seat index, -1 when everybody passed.
type auctionRow struct {
	Board    int    `json:"board"`
	Dealer   string `json:"dealer"`
	Vul      string `json:"vul"`
	PBN      string `json:"pbn"`
	Auction  string `json:"auction"`
	Level    int    `json:"level"`
	Strain   int    `json:"strain"`
	Declarer int    `json:"declarer"`
	Doubled  bool   `json:"doubled"`
	Calls    int    `json:"calls"`
}

// boardVul is the standard sixteen-board vulnerability rotation, per side
// (index 0 = N/S, 1 = E/W). Board n uses entry (n-1)%16, and its dealer is
// (n-1)%4 -- the same cycle a duplicate session runs.
var boardVul = [16][2]bool{
	{false, false}, {true, false}, {false, true}, {true, true},
	{true, false}, {false, true}, {true, true}, {false, false},
	{false, true}, {true, true}, {false, false}, {true, false},
	{true, true}, {false, false}, {true, false}, {false, true},
}

// ddsPBN formats a deal the way DdTableDealPBN expects it: hands in the order
// N E S W, suits from spades down to clubs.
func ddsPBN(d *Deal) string {
	hands := make([]string, 4)
	for seat := range 4 {
		h := d.Hands[seat]
		hands[seat] = fmt.Sprintf("%s.%s.%s.%s",
			h.Suits[Spades], h.Suits[Hearts], h.Suits[Diamonds], h.Suits[Clubs])
	}
	return "N:" + strings.Join(hands, " ")
}

// envInt reads a positive integer from the environment, falling back to def.
func envInt(t *testing.T, key string, def int64) int64 {
	t.Helper()
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		t.Fatalf("%s = %q: expected a positive integer", key, v)
	}
	return n
}

func TestParAuditDump(t *testing.T) {
	out := os.Getenv("PAR_AUDIT_OUT")
	if out == "" {
		t.Skip("set PAR_AUDIT_OUT to dump the auctions (see tools/par/README.md)")
	}
	deals := int(envInt(t, "PAR_AUDIT_DEALS", 1000))
	seed := envInt(t, "PAR_AUDIT_SEED", 20260906)

	f, err := os.Create(out)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	rng := rand.New(rand.NewSource(seed))
	enc := json.NewEncoder(f)
	for board := 1; board <= deals; board++ {
		d := randomDeal(rng)
		// randomDeal picks a dealer of its own; the board number decides it
		// here, together with the vulnerability, so a run is a plausible
		// session rather than an unconditioned sample.
		d.Dealer = (board - 1) % 4
		d.Vul = boardVul[(board-1)%16]

		calls := NewEngine(d).Run()
		contract, declarer, doubled := finalContract(calls)
		row := auctionRow{
			Board:    board,
			Dealer:   seatNames[d.Dealer],
			Vul:      d.VulString(),
			PBN:      ddsPBN(d),
			Auction:  formatAuction(calls),
			Declarer: declarer,
			Doubled:  doubled,
			Calls:    len(calls),
		}
		if declarer >= 0 && contract.IsBid() {
			row.Level, row.Strain = contract.Level, int(contract.Strain)
		}
		if err := enc.Encode(row); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d boards written to %s (seed %d)", deals, out, seed)
}
