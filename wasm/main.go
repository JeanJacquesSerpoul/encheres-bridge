//go:build js && wasm

// The bidding engine compiled to WebAssembly: the only way the application
// computes auctions, right in the browser. build-wasm.sh builds this package
// into cli/bids.wasm; the engine itself lives in ../engine, and this file only
// crosses the JavaScript boundary.
//
// cli/bids-wasm.js is the other half: it instantiates this module and turns
// the objects below into promises.

package main

import (
	"bids/engine"
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
	api.Set("version", js.ValueOf(engine.VersionJSON()))
	js.Global().Set(wasmGlobal, api)

	if hook := js.Global().Get(readyHook); hook.Type() == js.TypeFunction {
		hook.Invoke()
	}

	// Keep the Go runtime alive: returning from main would tear down the
	// functions the page is about to call.
	<-make(chan struct{})
}

// okResult and errResult are the two shapes every call returns: the page
// never has to catch a Go panic crossing into JavaScript.
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

// wasmArgs reads (pbn, lang) from the call; lang defaults to English.
func wasmArgs(args []js.Value) (pbn []byte, lang string, err error) {
	if len(args) == 0 || args[0].Type() != js.TypeString {
		return nil, "", fmt.Errorf("missing PBN text (first argument)")
	}
	lang = "en"
	if len(args) > 1 && args[1].Type() == js.TypeString && args[1].String() != "" {
		lang = args[1].String()
	}
	return []byte(args[0].String()), lang, nil
}

// call runs one engine function, turning its result — or a panic — into the
// object the page expects.
func call(args []js.Value, run func(pbn []byte, lang string) ([]byte, error)) (out any) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errResult("internal engine error: %v", rec)
		}
	}()
	pbn, lang, err := wasmArgs(args)
	if err != nil {
		return errResult("%v", err)
	}
	data, err := run(pbn, lang)
	if err != nil {
		return errResult("%v", err)
	}
	return okResult(data)
}

func wasmBid(_ js.Value, args []js.Value) any {
	return call(args, engine.BidJSON)
}

func wasmBids(_ js.Value, args []js.Value) any {
	return call(args, engine.BidsJSON)
}

func wasmSelfCheck(_ js.Value, _ []js.Value) (out any) {
	defer func() {
		if rec := recover(); rec != nil {
			out = errResult("engine self-check failed: %v", rec)
		}
	}()
	if err := engine.SelfCheck(); err != nil {
		return errResult("%v", err)
	}
	return okResult([]byte(`{"status":"ready"}`))
}
