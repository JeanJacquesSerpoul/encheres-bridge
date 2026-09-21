//go:build !(js && wasm)

// Tests for the HTTP server, which the WebAssembly build leaves out along
// with main.go. The engine's own tests carry no tag and run for every
// target.

package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRecoverMiddleware(t *testing.T) {
	panicky := func(w http.ResponseWriter, r *http.Request) { panic("boom") }
	req := httptest.NewRequest(http.MethodGet, "/whatever", nil)
	rec := httptest.NewRecorder()
	recoverMiddleware(panicky)(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected a non-empty error message")
	}
}

func TestBidHandler_BodyTooLarge(t *testing.T) {
	oversized := bytes.Repeat([]byte("A"), maxBodyBytes+1)
	req := newBidRequest(t, oversized, "en")
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLimitConcurrency_RejectsWhenFull(t *testing.T) {
	// Saturate every slot, then check the middleware rejects immediately
	// instead of blocking or letting the request through.
	for i := 0; i < cap(bidSlots); i++ {
		bidSlots <- struct{}{}
	}
	t.Cleanup(func() {
		for i := 0; i < cap(bidSlots); i++ {
			<-bidSlots
		}
	})

	called := false
	req := httptest.NewRequest(http.MethodPost, "/bid", nil)
	rec := httptest.NewRecorder()
	limitConcurrency(func(w http.ResponseWriter, r *http.Request) { called = true })(rec, req)

	if called {
		t.Fatalf("handler should not run when all slots are busy")
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestCORS_ConfigurableOrigin(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "https://example.com")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	cors(healthHandler)(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "https://example.com")
	}
}

func TestCORS_DefaultOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	cors(healthHandler)(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
}

// newBidRequest builds a multipart POST /bid request carrying pbnBody as the
// 'pbn' file field, optionally suffixed with a lang query parameter.
func newBidRequest(t *testing.T, pbnBody []byte, lang string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("pbn", "deal.pbn")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := fw.Write(pbnBody); err != nil {
		t.Fatalf("write pbn body: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	url := "/bid"
	if lang != "" {
		url += "?lang=" + lang
	}
	req := httptest.NewRequest(http.MethodPost, url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	cors(healthHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %v, want %q", body["status"], "ok")
	}
	for _, field := range []string{"uptime_s", "requests_total", "errors_total", "bids_run_total"} {
		if _, ok := body[field]; !ok {
			t.Fatalf("response missing field %q: %v", field, body)
		}
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}

func TestReadyHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	cors(readyHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["status"] != "ready" {
		t.Fatalf("status field = %q, want %q", body["status"], "ready")
	}
}

func TestVersionHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	cors(versionHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body versionInfo
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body.Revision == "" {
		t.Fatalf("expected a non-empty revision")
	}
	if body.Go == "" {
		t.Fatalf("expected a non-empty go version")
	}
}

// TestResolveBuildInfo checks the two sources of the version, in order: the
// revision the linker injects wins, and without it the VCS stamp the toolchain
// embeds takes over -- leaving "dev" only when neither is available (as under
// `go run`, which does not stamp).
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

func TestEmbeddedCLIAssets(t *testing.T) {
	sub, err := fs.Sub(cliFS, "cli")
	if err != nil {
		t.Fatalf("fs.Sub: %v", err)
	}
	data, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		t.Fatalf("embedded cli/index.html not readable: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("embedded cli/index.html is empty")
	}
}

func TestBidHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/bid", nil)
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if !strings.Contains(body["error"], "GET") {
		t.Fatalf("error message = %q, want it to mention GET", body["error"])
	}
}

func TestBidHandler_BadLang(t *testing.T) {
	pbn, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	req := newBidRequest(t, pbn, "de")
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBidHandler_MissingPBNField(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("other", "value"); err != nil {
		t.Fatalf("WriteField: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/bid", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if !strings.Contains(body["error"], "pbn") {
		t.Fatalf("error message = %q, want it to mention the missing 'pbn' field", body["error"])
	}
}

// TestBidHandler_RawTextBody covers the non-multipart path: a plain PBN body
// (as sent by `curl --data-binary @deal.pbn`), no multipart envelope needed.
func TestBidHandler_RawTextBody(t *testing.T) {
	pbn, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/bid?lang=en", bytes.NewReader(pbn))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

// TestBidHandler_RawTextBody_NoContentType checks the same raw-body path
// works with no Content-Type header at all, matching a bare
// `curl --data-binary` invocation.
func TestBidHandler_RawTextBody_NoContentType(t *testing.T) {
	pbn, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/bid", bytes.NewReader(pbn))
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
}

func TestBidHandler_RawTextBody_InvalidPBN(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/bid", strings.NewReader(`[Dealer "N"]`))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestBidHandler_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/bid", strings.NewReader(""))
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestBidHandler_InvalidPBN(t *testing.T) {
	pbn, err := os.ReadFile("testdata/bad.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	req := newBidRequest(t, pbn, "en")
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
	}
	if body["error"] == "" {
		t.Fatalf("expected a non-empty error message")
	}
}

// TestBidHandler_Golden pins the JSON contract of the /bid endpoint on the
// reference deal documented in the README, so a change to the response
// shape (field names, structure) breaks loudly instead of silently.
func TestBidHandler_Golden(t *testing.T) {
	pbn, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	req := newBidRequest(t, pbn, "fr")
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}

	var resp bidResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
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
	first := resp.Auction[0]
	if first.Player != "N" || first.Bid != "2SA" {
		t.Fatalf("first call = %+v, want opening 2SA by N", first)
	}
}

// TestBidsHandler_MultiBoard covers the tournament-file path: several
// [Board]/[Dealer]/[Deal] groups in one PBN file, one bidResponse per board.
func TestBidsHandler_MultiBoard(t *testing.T) {
	multi := `[Board "1"]
[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]
[Board "2"]
[Dealer "E"]
[Vulnerable "None"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]
`
	req := httptest.NewRequest(http.MethodPost, "/bids?lang=en", strings.NewReader(multi))
	rec := httptest.NewRecorder()
	cors(bidsHandler)(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body: %s", rec.Code, rec.Body.String())
	}
	var resp []bidResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON body: %v", err)
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

func TestBidsHandler_InvalidPBN(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/bids", strings.NewReader(`[Dealer "N"]`))
	rec := httptest.NewRecorder()
	cors(bidsHandler)(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestBidPreflight(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/bid", nil)
	rec := httptest.NewRecorder()
	cors(bidHandler)(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want %q", got, "*")
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("Access-Control-Allow-Methods = %q, want it to contain POST", got)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("preflight body = %q, want empty", rec.Body.String())
	}
}

// FuzzParsePBN exercises the only function in the server that parses
// arbitrary bytes coming straight from the network (the 'pbn' multipart
// field). It must never panic, whatever garbage is thrown at it.
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
