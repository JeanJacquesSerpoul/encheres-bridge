//go:build !(js && wasm)

// The HTTP server. Compiled out of the WebAssembly build, which has no
// use for net/http, log/slog or the embedded client, and which brings its
// own main() in main_js.go. Everything both builds need is in response.go.

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// The test client, including cli/bids.wasm when build-wasm.sh has produced
// it. This directive must stay in this file: an untagged copy of it would
// also land in the WebAssembly build, which would then embed itself and
// double in size on every rebuild.
//
//go:embed all:cli
var cliFS embed.FS

var logger = newLogger()

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}
	var handler slog.Handler
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "text") {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

// Basic in-process counters, exposed via /health. Not a substitute for a
// real metrics backend, but enough to see traffic and error trends without
// standing up Prometheus for a single small binary.
var (
	startTime     = time.Now()
	requestsTotal atomic.Int64
	errorsTotal   atomic.Int64
	bidsRunTotal  atomic.Int64
)

// corsOrigin returns the configured Access-Control-Allow-Origin value.
// Defaults to "*" (the bundled test client can run from file://, another
// port...); set CORS_ORIGINS to restrict it to specific origin(s) in
// production.
func corsOrigin() string {
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		return v
	}
	return "*"
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", corsOrigin())
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

// crossOriginIsolated adds the COOP/COEP response headers the embedded test
// client needs to enable SharedArrayBuffer, which the double-dummy WASM solver
// behind the "Calcul du PAR" button relies on (see cli/par.js). Every CLI asset
// is same-origin, so require-corp costs nothing here; cli/coi-serviceworker.js
// is only a fallback for hosts that strip these headers.
func crossOriginIsolated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		next.ServeHTTP(w, r)
	})
}

// cachedAsset is one embedded file, compressed and fingerprinted once.
type cachedAsset struct {
	gz   []byte
	etag string
}

var gzipCache sync.Map // file name -> cachedAsset

func compressible(name string) bool {
	switch path.Ext(name) {
	case ".wasm", ".js", ".css", ".html", ".svg", ".json":
		return true
	}
	return false
}

// compressAsset gzips raw and derives the ETag from the *uncompressed* bytes,
// so the tag identifies the file's content rather than this build's choice of
// compression level.
func compressAsset(raw []byte) cachedAsset {
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	_, _ = zw.Write(raw)
	_ = zw.Close()
	sum := sha256.Sum256(raw)
	return cachedAsset{
		gz:   buf.Bytes(),
		etag: `"` + hex.EncodeToString(sum[:8]) + `"`,
	}
}

// warmGzipCache compresses every embedded asset up front, in a goroutine at
// startup. BestCompression over a multi-megabyte cli/bids.wasm takes a
// noticeable fraction of a second; computed lazily, that cost lands on the
// first visitor after each restart — who is already the one waiting on the
// largest download. Startup is where it belongs.
func warmGzipCache(root fs.FS) {
	start := time.Now()
	var files, rawBytes, gzBytes int
	_ = fs.WalkDir(root, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !compressible(name) {
			return nil
		}
		raw, err := fs.ReadFile(root, name)
		if err != nil {
			return nil
		}
		a := compressAsset(raw)
		gzipCache.Store(name, a)
		files++
		rawBytes += len(raw)
		gzBytes += len(a.gz)
		return nil
	})
	logger.Info("static assets pre-compressed",
		"files", files, "raw_bytes", rawBytes, "gzip_bytes", gzBytes,
		"duration_ms", time.Since(start).Milliseconds())
}

// cacheControl tells the browser to revalidate every asset on each load.
//
// The file names carry no content hash, so any freshness window leaves a
// client running yesterday's app.js against today's server: with the former
// five-minute max-age, rebuilding through run.ps1 and reloading the page
// still showed the previous client. Revalidation is cheap — every cli/ asset
// goes through gzipStatic, which tags it with an ETag, so an unchanged file
// costs a single 304 round trip, bids.wasm included.
//
// Without any header, the assets had no freshness information at all:
// embed.FS reports a zero modification time, so there was no Last-Modified
// either.
func cacheControl(string) string {
	return "no-cache"
}

// gzipStatic serves the bulky cli/ assets compressed, which http.FileServer
// never does: cli/bids.wasm is megabytes of Go code and shrinks by more than
// a factor of three. The gzip bytes and the ETag are computed on first request
// and kept — //go:embed fixes the file list at compile time, so the cache is
// bounded by the binary itself.
//
// The ETag also works around an embed.FS quirk: its files report a zero
// modification time, so http.ServeContent sends no Last-Modified and the
// browser has nothing to revalidate against — it would re-download the whole
// module on every single page load.
func gzipStatic(root fs.FS, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		// Naming index.html explicitly lets the page itself be served gzipped
		// and with an ETag; FileServer's directory handling would otherwise
		// send it raw and untagged.
		if r.URL.Path == "/" {
			name = "index.html"
		}
		w.Header().Set("Cache-Control", cacheControl(name))
		if !compressible(name) ||
			!strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		v, ok := gzipCache.Load(name)
		if !ok {
			raw, err := fs.ReadFile(root, name)
			if err != nil {
				next.ServeHTTP(w, r) // 404s, redirects, directories: FileServer's job
				return
			}
			v, _ = gzipCache.LoadOrStore(name, compressAsset(raw))
		}
		a := v.(cachedAsset)

		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("ETag", a.etag)
		if r.Header.Get("If-None-Match") == a.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		// Writing the body ourselves bypasses FileServer's content sniffing,
		// and WebAssembly.instantiateStreaming refuses a module that does not
		// arrive as application/wasm.
		if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Length", strconv.Itoa(len(a.gz)))
		w.WriteHeader(http.StatusOK)
		if r.Method != http.MethodHead {
			_, _ = w.Write(a.gz)
		}
	})
}

// recoverMiddleware turns a panic in next into a logged 500 instead of a
// crashed connection, so a bug anywhere in the decision tree can't take the
// whole server down.
func recoverMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic handling request", "method", r.Method, "path", r.URL.Path, "panic", rec, "stack", string(debug.Stack()))
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next(w, r)
	}
}

// statusRecorder captures the status code and byte count of a response so
// accessLog can report them after the handler has run.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func newRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// accessLog logs one structured line per request (method, path, status,
// duration, size) and stamps an X-Request-ID (reused if the caller sent
// one), so a single request can be traced through logs on both sides.
func accessLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = newRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)
		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()
		next(rec, r)

		requestsTotal.Add(1)
		if rec.status >= 400 {
			errorsTotal.Add(1)
		}
		logger.Info("request",
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		)
	}
}

// bidSlots bounds how many /bid and /bids requests run concurrently: the
// engine is CPU-bound, so letting an unbounded number of requests through
// would let a traffic spike starve the whole process.
var bidSlots = make(chan struct{}, max(4, runtime.GOMAXPROCS(0)))

func limitConcurrency(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case bidSlots <- struct{}{}:
			defer func() { <-bidSlots }()
			next(w, r)
		default:
			writeError(w, http.StatusServiceUnavailable, "server busy, try again shortly")
		}
	}
}

// writeJSON renders v through encodeJSON rather than straight to the socket,
// so the bytes it sends are the same ones the WebAssembly module hands the
// browser for the same deal. Encoding first also means a failure is still
// reportable, instead of truncating a response whose status line is gone.
func writeJSON(w http.ResponseWriter, status int, v any) {
	data, err := encodeJSON(v)
	if err != nil {
		logger.Error("cannot encode response", "error", err)
		http.Error(w, `{"error":"cannot encode response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, status int, format string, args ...any) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf(format, args...)})
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"uptime_s":       int(time.Since(startTime).Seconds()),
		"requests_total": requestsTotal.Load(),
		"errors_total":   errorsTotal.Load(),
		"bids_run_total": bidsRunTotal.Load(),
	})
}

// readyHandler replays a reference deal through the full engine so a broken
// decision tree fails the container's health check, not just a dead process.
func readyHandler(w http.ResponseWriter, _ *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			writeError(w, http.StatusServiceUnavailable, "engine self-check failed: %v", rec)
		}
	}()
	deal, err := ParsePBN([]byte(referencePBN))
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "engine self-check failed: %v", err)
		return
	}
	calls := NewEngine(deal).Run()
	if len(calls) == 0 {
		writeError(w, http.StatusServiceUnavailable, "engine self-check produced no calls")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func versionHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, buildInfo)
}

// maxBodyBytes bounds the whole /bid and /bids request body (PBN files are a
// few hundred bytes each; this leaves ample room while capping memory use
// per request under load).
const maxBodyBytes = 1 << 20

// parseLangAndBody validates the shared 'lang' query parameter and reads the
// PBN payload, accepting either a multipart 'pbn' file field (as the bundled
// test client sends) or a raw PBN body (e.g. `curl --data-binary @deal.pbn`).
// It writes an error response and returns ok=false on any failure.
func parseLangAndBody(w http.ResponseWriter, r *http.Request) (lang string, data []byte, ok bool) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method %s not allowed, use POST", r.Method)
		return "", nil, false
	}
	lang = r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}
	if lang != "en" && lang != "fr" {
		writeError(w, http.StatusBadRequest, "unsupported lang %q (use 'en' or 'fr')", lang)
		return "", nil, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	defer r.Body.Close()

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mediaType == "multipart/form-data" {
		if err := r.ParseMultipartForm(maxBodyBytes); err != nil {
			writeError(w, http.StatusBadRequest, "invalid or too large multipart body: %v", err)
			return "", nil, false
		}
		file, _, err := r.FormFile("pbn")
		if err != nil {
			writeError(w, http.StatusBadRequest, "missing field 'pbn' (multipart file)")
			return "", nil, false
		}
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			writeError(w, http.StatusBadRequest, "cannot read pbn file: %v", err)
			return "", nil, false
		}
		return lang, data, true
	}

	var err error
	data, err = io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "cannot read request body: %v", err)
		return "", nil, false
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "empty request body: send a PBN file (multipart field 'pbn', or raw PBN text as the body)")
		return "", nil, false
	}
	return lang, data, true
}

func bidHandler(w http.ResponseWriter, r *http.Request) {
	lang, data, ok := parseLangAndBody(w, r)
	if !ok {
		return
	}
	deal, err := ParsePBN(data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid PBN file: %v", err)
		return
	}
	calls := NewEngine(deal).Run()
	bidsRunTotal.Add(1)
	writeJSON(w, http.StatusOK, buildResponse(deal, calls, lang))
}

// bidsHandler is the multi-board counterpart of bidHandler: it accepts a
// tournament PBN file (several [Board]/[Dealer]/[Deal] groups) and returns
// one bidResponse per board.
func bidsHandler(w http.ResponseWriter, r *http.Request) {
	lang, data, ok := parseLangAndBody(w, r)
	if !ok {
		return
	}
	deals, err := ParsePBNBoards(data)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "invalid PBN file: %v", err)
		return
	}
	responses := make([]bidResponse, len(deals))
	for i, deal := range deals {
		calls := NewEngine(deal).Run()
		responses[i] = buildResponse(deal, calls, lang)
	}
	bidsRunTotal.Add(int64(len(deals)))
	writeJSON(w, http.StatusOK, responses)
}

// bidTimeout bounds how long a single /bid or /bids request may run. The
// engine is deterministic and normally completes in well under a second;
// this is a backstop against a decision-tree bug turning into a stuck
// connection. /bids gets more headroom since it runs the engine once per
// board in the file.
const (
	bidTimeout  = 5 * time.Second
	bidsTimeout = 20 * time.Second
)

// notFoundHandler answers unmatched paths with the same JSON error shape as
// the rest of the API, instead of net/http's plain-text default.
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "not found: %s", r.URL.Path)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9015"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", accessLog(cors(recoverMiddleware(healthHandler))))
	mux.HandleFunc("/ready", accessLog(cors(recoverMiddleware(readyHandler))))
	mux.HandleFunc("/version", accessLog(cors(recoverMiddleware(versionHandler))))

	bid := http.TimeoutHandler(recoverMiddleware(limitConcurrency(bidHandler)), bidTimeout, `{"error":"request timed out"}`)
	mux.HandleFunc("/bid", accessLog(cors(bid.ServeHTTP)))

	bids := http.TimeoutHandler(recoverMiddleware(limitConcurrency(bidsHandler)), bidsTimeout, `{"error":"request timed out"}`)
	mux.HandleFunc("/bids", accessLog(cors(bids.ServeHTTP)))

	// The test client (cli/) is embedded in the binary at compile time, so
	// serving it no longer depends on the process's working directory. Set
	// SERVE_CLI=false to disable it (e.g. in production, where only /health,
	// /ready, /version, /bid and /bids are meant to be reachable).
	if os.Getenv("SERVE_CLI") == "false" {
		mux.HandleFunc("/", accessLog(cors(notFoundHandler)))
	} else {
		cliRoot, err := fs.Sub(cliFS, "cli")
		if err != nil {
			logger.Error("cannot mount embedded cli assets", "error", err)
			mux.HandleFunc("/", accessLog(cors(notFoundHandler)))
		} else {
			mux.Handle("/", crossOriginIsolated(gzipStatic(cliRoot, http.FileServer(http.FS(cliRoot)))))
			// In the background: serving can start immediately, and any
			// request that beats the warm-up just compresses its own file.
			go warmGzipCache(cliRoot)
			logger.Info("test client served from embedded ./cli at /")
		}
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      bidsTimeout + 5*time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("bridge bidding server listening", "addr", srv.Addr, "revision", buildRevision)
		serverErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining in-flight requests")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		} else {
			logger.Info("shutdown complete")
		}
	}
}
