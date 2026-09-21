// Everything both builds of the engine share: the JSON shapes the API answers
// with, the build stamp, and the one encoder they both go through. main.go
// carries the HTTP server and is compiled out of the WebAssembly build, so
// anything it and main_js.go both need lives here rather than in either of
// them.

package main

import (
	"bytes"
	"encoding/json"
	"runtime"
	"runtime/debug"
)

// buildRevision identifies the running binary; override at build time with
// -ldflags "-X main.buildRevision=<sha>". The build-server and build-docker
// scripts both pass `git rev-parse --short HEAD` — the latter through the
// Dockerfile's REVISION build arg, which defaults to "docker" when the image
// is built by hand. Left alone, it falls back to the VCS stamp the Go
// toolchain embeds, so an ordinary `go build` still reports the commit it was
// built from; only `go run`, which does not stamp, falls through to "dev".
var buildRevision = ""

// buildInfo is the version the /version endpoint serves, resolved once at
// startup.
var buildInfo = resolveBuildInfo()

type versionInfo struct {
	Revision string `json:"revision"`
	Time     string `json:"time,omitempty"` // commit date, RFC 3339
	Modified bool   `json:"modified"`       // built from a dirty tree
	Go       string `json:"go"`
}

// resolveBuildInfo prefers the revision injected at link time, and otherwise
// reads the one the toolchain stamped into the binary. A dirty tree is marked
// as such: a bare sha would claim the binary matches a commit it does not.
func resolveBuildInfo() versionInfo {
	v := versionInfo{Revision: buildRevision, Go: runtime.Version()}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if v.Revision == "" && len(s.Value) >= 7 {
					v.Revision = s.Value[:7]
				}
			case "vcs.time":
				v.Time = s.Value
			case "vcs.modified":
				v.Modified = s.Value == "true"
			}
		}
	}
	if v.Revision == "" {
		v.Revision = "dev"
	}
	return v
}

type handJSON struct {
	Spades   string `json:"spades"`
	Hearts   string `json:"hearts"`
	Diamonds string `json:"diamonds"`
	Clubs    string `json:"clubs"`
	HLPoints int    `json:"hl_points"`
	HPoints  int    `json:"h_points"`
	Type     string `json:"type"`
}

type auctionJSON struct {
	Player  string `json:"player"`
	Bid     string `json:"bid"`
	Comment string `json:"comment"`
}

type bidResponse struct {
	Board      string              `json:"board,omitempty"`
	Dealer     string              `json:"dealer"`
	Vulnerable string              `json:"vulnerable"`
	Lang       string              `json:"lang"`
	Hands      map[string]handJSON `json:"hands"`
	Auction    []auctionJSON       `json:"auction"`
	Contract   string              `json:"contract"`
	Declarer   string              `json:"declarer"`
	Doubled    bool                `json:"doubled"`
}

var typeFR = map[string]string{
	"regular":       "régulière",
	"single-suited": "unicolore",
	"two-suited":    "bicolore",
	"three-suited":  "tricolore",
}

// localizeHandType translates a Hand.Type() key ("regular", ...) for the
// response's language.
func localizeHandType(t, lang string) string {
	if lang == "fr" {
		return typeFR[t]
	}
	return t
}

func buildResponse(deal *Deal, calls []SeatCall, lang string) bidResponse {
	resp := bidResponse{
		Board:      deal.Board,
		Dealer:     seatNames[deal.Dealer],
		Vulnerable: deal.VulString(),
		Lang:       lang,
		Hands:      map[string]handJSON{},
	}
	for seat, h := range deal.Hands {
		resp.Hands[seatNames[seat]] = handJSON{
			Spades:   h.Suits[Spades],
			Hearts:   h.Suits[Hearts],
			Diamonds: h.Suits[Diamonds],
			Clubs:    h.Suits[Clubs],
			HLPoints: h.HL(),
			HPoints:  h.H(),
			Type:     localizeHandType(h.Type(), lang),
		}
	}
	for _, sc := range calls {
		comment := sc.M.en
		if lang == "fr" {
			comment = sc.M.fr
		}
		resp.Auction = append(resp.Auction, auctionJSON{
			Player:  seatNames[sc.Seat],
			Bid:     sc.Call.Format(lang),
			Comment: comment,
		})
	}
	contract, declarer, doubled := finalContract(calls)
	if contract.IsBid() {
		resp.Contract = contract.Format(lang)
		resp.Declarer = seatNames[declarer]
	} else {
		resp.Contract = passCall.Format(lang)
	}
	resp.Doubled = doubled
	return resp
}

// finalContract extracts the contract, declarer and doubled status.
func finalContract(calls []SeatCall) (Call, int, bool) {
	var contract Call
	contractSeat := -1
	doubled := false
	for _, sc := range calls {
		switch sc.Call.Kind {
		case KindBid:
			contract, contractSeat = sc.Call, sc.Seat
			doubled = false
		case KindDouble, KindRedouble:
			doubled = true
		}
	}
	if contractSeat < 0 {
		return Call{}, -1, false
	}
	// Declarer: first player of the winning side to have named the strain.
	for _, sc := range calls {
		if sc.Call.IsBid() && sc.Call.Strain == contract.Strain && sideOf(sc.Seat) == sideOf(contractSeat) {
			return contract, sc.Seat, doubled
		}
	}
	return contract, contractSeat, doubled
}

// referencePBN is a fixed, known-good deal used by the /ready endpoint and by
// the WebAssembly module's self-check to exercise the full parse-and-bid path
// without depending on testdata/ being present alongside the binary in
// production.
const referencePBN = `[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`

// encodeJSON renders v the way the HTTP handlers put it on the wire: same
// encoder, same SetEscapeHTML(false), same trailing newline. Both transports
// go through it, so the WebAssembly answer and the /bid body for a given deal
// are the same bytes by construction rather than by coincidence.
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
