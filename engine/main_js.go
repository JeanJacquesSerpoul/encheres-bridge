//go:build js && wasm

// The engine compiled to WebAssembly, so the test client computes auctions in
// the browser instead of asking the server for them. Same path as /bid --
// ParsePBN, NewEngine().Run(), buildResponse -- and the same encoder
// (response.go), so both answers are the same bytes for a given deal.
//
// cli/bids-wasm.js is the other half: it instantiates this module and turns
// the objects below into promises.

package main

import (
	"fmt"
	"syscall/js"
)

const (
	// wasmGlobal is the object this module installs on globalThis.
	wasmGlobal = "bidsWasm"
	// readyHook is the callback the page installs before go.run(), and which
	// main() invokes once the API is in place. It is the only reliable signal
	// that the module is usable: go.run()'s own promise settles when the
	// program exits, which here never happens.
	readyHook = "__bidsWasmReady"
)

func main() {
	api := js.Global().Get("Object").New()
	api.Set("bid", js.FuncOf(wasmBid))
	api.Set("bids", js.FuncOf(wasmBids))
	api.Set("selfCheck", js.FuncOf(wasmSelfCheck))
	api.Set("version", js.ValueOf(wasmVersion()))
	js.Global().Set(wasmGlobal, api)

	if hook := js.Global().Get(readyHook); hook.Type() == js.TypeFunction {
		hook.Invoke()
	}

	// js.FuncOf handles are only valid while main runs: returning from here
	// would make every later call raise "Go program has already exited". A
	// channel nobody ever feeds blocks without busying the scheduler, and the
	// js/wasm runtime knows callbacks are still pending, so it does not report
	// a deadlock.
	<-make(chan struct{})
}

// Every entry point answers the same shape: { ok: true, json: "<text>" } or
// { ok: false, error: "<message>" }. The payload is returned as *text*, not as
// a JavaScript object, because that is what the server puts on the wire down
// to the character -- which is what makes the two paths comparable. The caller
// does the JSON.parse.

func okResult(payload []byte) js.Value {
	r := js.Global().Get("Object").New()
	r.Set("ok", true)
	r.Set("json", string(payload))
	return r
}

func errResult(format string, args ...any) js.Value {
	r := js.Global().Get("Object").New()
	r.Set("ok", false)
	r.Set("error", fmt.Sprintf(format, args...))
	return r
}

// wasmArgs validates (pbn, lang) with the wording parseLangAndBody uses, so a
// bad call reads the same whichever transport produced it.
func wasmArgs(args []js.Value) (pbn, lang string, err error) {
	if len(args) == 0 || args[0].Type() != js.TypeString {
		return "", "", fmt.Errorf("missing PBN text (first argument)")
	}
	pbn = args[0].String()
	if pbn == "" {
		return "", "", fmt.Errorf("empty request body: send a PBN file (multipart field 'pbn', or raw PBN text as the body)")
	}
	lang = "en"
	if len(args) > 1 && args[1].Type() == js.TypeString && args[1].String() != "" {
		lang = args[1].String()
	}
	if lang != "en" && lang != "fr" {
		return "", "", fmt.Errorf("unsupported lang %q (use 'en' or 'fr')", lang)
	}
	return pbn, lang, nil
}

// wasmBid is the counterpart of bidHandler. The recover mirrors
// recoverMiddleware: a panic in the decision tree must not tear the module
// down and leave the page with no engine at all.
func wasmBid(_ js.Value, args []js.Value) (out any) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errResult("internal engine error: %v", rec)
		}
	}()
	pbn, lang, err := wasmArgs(args)
	if err != nil {
		return errResult("%v", err)
	}
	deal, err := ParsePBN([]byte(pbn))
	if err != nil {
		return errResult("invalid PBN file: %v", err)
	}
	data, err := encodeJSON(buildResponse(deal, NewEngine(deal).Run(), lang))
	if err != nil {
		return errResult("cannot encode response: %v", err)
	}
	return okResult(data)
}

// wasmBids is the counterpart of bidsHandler: a tournament file, one response
// per board.
func wasmBids(_ js.Value, args []js.Value) (out any) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errResult("internal engine error: %v", rec)
		}
	}()
	pbn, lang, err := wasmArgs(args)
	if err != nil {
		return errResult("%v", err)
	}
	deals, err := ParsePBNBoards([]byte(pbn))
	if err != nil {
		return errResult("invalid PBN file: %v", err)
	}
	responses := make([]bidResponse, len(deals))
	for i, deal := range deals {
		responses[i] = buildResponse(deal, NewEngine(deal).Run(), lang)
	}
	data, err := encodeJSON(responses)
	if err != nil {
		return errResult("cannot encode response: %v", err)
	}
	return okResult(data)
}

// wasmSelfCheck is the counterpart of readyHandler, on the same reference
// deal: the client's status dot then means the same thing in both modes.
func wasmSelfCheck(_ js.Value, _ []js.Value) (out any) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errResult("engine self-check failed: %v", rec)
		}
	}()
	deal, err := ParsePBN([]byte(referencePBN))
	if err != nil {
		return errResult("engine self-check failed: %v", err)
	}
	if len(NewEngine(deal).Run()) == 0 {
		return errResult("engine self-check produced no calls")
	}
	data, err := encodeJSON(map[string]string{"status": "ready"})
	if err != nil {
		return errResult("engine self-check failed: %v", err)
	}
	return okResult(data)
}

// wasmVersion returns what /version serves, as text. The revision comes from
// the same -ldflags the server binary is stamped with (see build-wasm.sh), so
// the client can show which engine it is actually running.
func wasmVersion() string {
	data, err := encodeJSON(buildInfo)
	if err != nil {
		return `{"revision":"dev"}`
	}
	return string(data)
}
