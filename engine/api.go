package engine

import (
	"errors"
	"fmt"
)

// The engine's public face: what the WebAssembly entry point (wasm/main.go)
// hands to the page. Each call takes PBN text and returns the JSON the client
// reads, so the entry point only has to cross the JavaScript boundary.

// checkLang validates the language of the comments and hand types.
func checkLang(lang string) error {
	if lang != "en" && lang != "fr" {
		return fmt.Errorf("unsupported lang %q (use 'en' or 'fr')", lang)
	}
	return nil
}

// BidJSON runs the auction of a single-deal PBN file and returns the response
// as JSON: hands, auction with comments, final contract.
func BidJSON(pbn []byte, lang string) ([]byte, error) {
	if err := checkLang(lang); err != nil {
		return nil, err
	}
	if len(pbn) == 0 {
		return nil, errors.New("empty PBN text")
	}
	deal, err := ParsePBN(pbn)
	if err != nil {
		return nil, fmt.Errorf("invalid PBN file: %w", err)
	}
	return encodeJSON(buildResponse(deal, NewEngine(deal).Run(), lang))
}

// BidsJSON does the same for every board of a tournament file, returning a
// JSON array with one response per board.
func BidsJSON(pbn []byte, lang string) ([]byte, error) {
	if err := checkLang(lang); err != nil {
		return nil, err
	}
	if len(pbn) == 0 {
		return nil, errors.New("empty PBN text")
	}
	deals, err := ParsePBNBoards(pbn)
	if err != nil {
		return nil, fmt.Errorf("invalid PBN file: %w", err)
	}
	responses := make([]bidResponse, len(deals))
	for i, deal := range deals {
		responses[i] = buildResponse(deal, NewEngine(deal).Run(), lang)
	}
	return encodeJSON(responses)
}

// SelfCheck replays the reference deal: the page calls it on load, both to
// show the engine's state and to instantiate the module before the first
// real request.
func SelfCheck() error {
	deal, err := ParsePBN([]byte(referencePBN))
	if err != nil {
		return fmt.Errorf("engine self-check failed: %w", err)
	}
	if len(NewEngine(deal).Run()) == 0 {
		return errors.New("engine self-check produced no calls")
	}
	return nil
}

// VersionJSON describes the build (revision, commit date, Go version) as JSON,
// for the page footer.
func VersionJSON() string {
	data, err := encodeJSON(buildInfo)
	if err != nil {
		return `{"revision":"dev"}`
	}
	return string(data)
}
