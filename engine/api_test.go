package engine

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Tests of the engine's public API (api.go), the one the WebAssembly module
// exposes to the page. They took over from the HTTP handler tests when the
// server was removed: same deals, same expectations, minus the transport.

// TestBidJSON_Golden pins the JSON contract on the reference deal, so a
// change to the response shape (field names, structure) breaks loudly
// instead of silently — the page reads these fields by name.
func TestBidJSON_Golden(t *testing.T) {
	pbn, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	data, err := BidJSON(pbn, "fr")
	if err != nil {
		t.Fatalf("BidJSON: %v", err)
	}

	var resp bidResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Dealer != "N" {
		t.Fatalf("dealer = %q, want %q", resp.Dealer, "N")
	}
	if resp.Lang != "fr" {
		t.Fatalf("lang = %q, want %q", resp.Lang, "fr")
	}
	if resp.Vulnerable != "None" {
		t.Fatalf("vulnerable = %q, want %q", resp.Vulnerable, "None")
	}
	if resp.Board != "" {
		t.Fatalf("board = %q, want empty (testdata/d1.pbn has no [Board] tag)", resp.Board)
	}
	north, ok := resp.Hands["N"]
	if !ok {
		t.Fatalf("response missing hand for N")
	}
	wantNorth := handJSON{
		Spades: "AKQ", Hearts: "KJ4", Diamonds: "AQ54", Clubs: "J32",
		HLPoints: 20, HPoints: 20, Type: "régulière",
	}
	if north != wantNorth {
		t.Fatalf("hand N = %+v, want %+v", north, wantNorth)
	}
	if resp.Contract != "3SA" {
		t.Fatalf("contract = %q, want %q", resp.Contract, "3SA")
	}
	if resp.Declarer != "N" {
		t.Fatalf("declarer = %q, want %q", resp.Declarer, "N")
	}
	if resp.Doubled {
		t.Fatalf("doubled = true, want false")
	}
	if len(resp.Auction) == 0 {
		t.Fatalf("expected a non-empty auction")
	}
	if first := resp.Auction[0]; first.Player != "N" || first.Bid != "2SA" {
		t.Fatalf("first call = %+v, want opening 2SA by N", first)
	}
}

// TestBidsJSON_MultiBoard covers the tournament-file path: several
// [Board]/[Dealer]/[Deal] groups in one PBN file, one response per board.
func TestBidsJSON_MultiBoard(t *testing.T) {
	multi := `[Board "1"]
[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]
[Board "2"]
[Dealer "E"]
[Vulnerable "None"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]
`
	data, err := BidsJSON([]byte(multi), "en")
	if err != nil {
		t.Fatalf("BidsJSON: %v", err)
	}
	var resp []bidResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("got %d boards, want 2", len(resp))
	}
	if resp[0].Board != "1" || resp[0].Dealer != "N" || resp[0].Vulnerable != "NS" {
		t.Fatalf("board 1 = %+v, want Board=1 Dealer=N Vulnerable=NS", resp[0])
	}
	if resp[1].Board != "2" || resp[1].Dealer != "E" || resp[1].Vulnerable != "None" {
		t.Fatalf("board 2 = %+v, want Board=2 Dealer=E Vulnerable=None", resp[1])
	}
}

func TestBidJSON_Errors(t *testing.T) {
	bad, err := os.ReadFile("testdata/bad.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	good, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	cases := []struct {
		name string
		pbn  []byte
		lang string
		want string
	}{
		{"invalid PBN", bad, "en", "invalid PBN file"},
		{"empty PBN", nil, "en", "empty PBN text"},
		{"unsupported lang", good, "de", "unsupported lang"},
	}
	for _, c := range cases {
		if _, err := BidJSON(c.pbn, c.lang); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to mention %q", c.name, err, c.want)
		}
	}
	if _, err := BidsJSON([]byte(`[Dealer "N"]`), "en"); err == nil {
		t.Errorf("BidsJSON on a deal without [Deal]: want an error")
	}
}

func TestSelfCheck(t *testing.T) {
	if err := SelfCheck(); err != nil {
		t.Fatalf("SelfCheck: %v", err)
	}
}

func TestVersionJSON(t *testing.T) {
	var v versionInfo
	if err := json.Unmarshal([]byte(VersionJSON()), &v); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v.Revision == "" || v.Go == "" {
		t.Fatalf("version = %+v, want a revision and a Go version", v)
	}
}

// TestResolveBuildInfo: the revision injected at link time wins; otherwise
// the VCS stamp the toolchain embeds takes over — leaving "dev" only when
// neither is available (as under `go test`, which does not stamp).
func TestResolveBuildInfo(t *testing.T) {
	saved := buildRevision
	defer func() { buildRevision = saved }()

	buildRevision = "abc1234"
	if v := resolveBuildInfo(); v.Revision != "abc1234" {
		t.Fatalf("revision = %q, want the linker value abc1234", v.Revision)
	}

	buildRevision = ""
	v := resolveBuildInfo()
	if v.Revision == "" {
		t.Fatalf("revision must never be empty")
	}
	if v.Go == "" {
		t.Fatalf("go version must be reported")
	}
	if v.Revision != "dev" && len(v.Revision) != 7 {
		t.Fatalf("revision = %q, want \"dev\" or a seven-character sha", v.Revision)
	}
}

// FuzzParsePBN: the parser reads whatever text the user pastes or loads, and
// must never panic, whatever garbage is thrown at it.
func FuzzParsePBN(f *testing.F) {
	seeds, err := os.ReadDir("testdata")
	if err != nil {
		f.Fatalf("read testdata dir: %v", err)
	}
	for _, entry := range seeds {
		data, err := os.ReadFile("testdata/" + entry.Name())
		if err != nil {
			f.Fatalf("read seed %s: %v", entry.Name(), err)
		}
		f.Add(data)
	}
	f.Add([]byte(""))
	f.Add([]byte(`[Dealer "N"`))
	f.Add([]byte(`[Deal "N:...."]`))
	f.Fuzz(func(t *testing.T, data []byte) {
		deal, err := ParsePBN(data)
		if err != nil {
			return
		}
		// A successfully parsed deal must be usable by the engine without
		// panicking either.
		NewEngine(deal).Run()
	})
}
