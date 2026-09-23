// Command serve publishes cli/ over HTTP on this machine, for run.ps1, run.sh
// and run-macos.command. It is a plain file server: the bidding engine runs in
// the page (cli/bids.wasm), so there is nothing to compute here.
//
// It exists rather than any off-the-shelf static server for two headers:
// Cross-Origin-Opener-Policy and Cross-Origin-Embedder-Policy make the page
// cross-origin isolated, which the double-dummy solver behind « Calcul du
// PAR » needs for SharedArrayBuffer. Without them the page still works, but
// cli/coi-serviceworker.js has to supply them, at the cost of a reload on the
// first visit.
//
// Usage: go run ./serve [-dir cli] [-port 9015]   (PORT also sets the port)
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9015"
	}
	dir := flag.String("dir", "cli", "directory to serve")
	flag.StringVar(&port, "port", port, "port to listen on (localhost only)")
	flag.Parse()

	if _, err := os.Stat(filepath.Join(*dir, "index.html")); err != nil {
		log.Fatalf("no index.html in %q: run from the repository root, or pass -dir", *dir)
	}

	files := http.FileServer(http.Dir(*dir))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Embedder-Policy", "require-corp")
		// Always revalidate: after an edit or a rebuild, a reload must show
		// the new app.js, not a cached one. Unchanged files cost a 304.
		h.Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	})

	// localhost only: this serves a working copy, not a site.
	addr := "localhost:" + port
	log.Printf("cli/ servi sur http://%s/ (Ctrl+C pour arrêter)", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
