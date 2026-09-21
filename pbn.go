package main

import (
	"fmt"
	"regexp"
	"strings"
)

var seatNames = [4]string{"N", "E", "S", "W"}

func seatIndex(letter string) (int, bool) {
	for i, n := range seatNames {
		if n == letter {
			return i, true
		}
	}
	return 0, false
}

// Deal is the parsed content of a PBN file.
type Deal struct {
	Dealer int
	Hands  [4]*Hand
	Vul    [2]bool // vulnerability per side (index 0 = N/S, 1 = E/W)
	Board  string  // the [Board] tag verbatim, empty if absent
}

// VulString formats the deal's vulnerability the way PBN does: "None", "NS",
// "EW" or "All".
func (d *Deal) VulString() string {
	switch {
	case d.Vul[0] && d.Vul[1]:
		return "All"
	case d.Vul[0]:
		return "NS"
	case d.Vul[1]:
		return "EW"
	default:
		return "None"
	}
}

// parseVulnerable maps the PBN Vulnerable tag values to per-side flags.
func parseVulnerable(v string) [2]bool {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "NS":
		return [2]bool{true, false}
	case "EW":
		return [2]bool{false, true}
	case "ALL", "BOTH":
		return [2]bool{true, true}
	}
	return [2]bool{} // "None", "Love", "-" or absent
}

var tagRe = regexp.MustCompile(`\[\s*(\w+)\s+"([^"]*)"\s*\]`)

// ParsePBN extracts the Dealer and Deal tags from a PBN file body.
func ParsePBN(data []byte) (*Deal, error) {
	text := strings.TrimPrefix(string(data), "\ufeff")
	tags := map[string]string{}
	for _, match := range tagRe.FindAllStringSubmatch(text, -1) {
		key := strings.ToLower(match[1])
		if _, seen := tags[key]; !seen {
			tags[key] = match[2]
		}
	}
	return dealFromTags(tags)
}

// ParsePBNBoards splits a (possibly multi-board tournament) PBN file into one
// *Deal per board. Board/Dealer/Vulnerable/Deal tags are board-scoped: a new
// board starts wherever these were last seen, and closes on the [Deal] tag.
// Persistent tags (Event, Site...) are ignored, since a Deal carries only
// what the bidding engine needs.
func ParsePBNBoards(data []byte) ([]*Deal, error) {
	text := strings.TrimPrefix(string(data), "\ufeff")
	pending := map[string]string{}
	var deals []*Deal
	for _, match := range tagRe.FindAllStringSubmatch(text, -1) {
		key := strings.ToLower(match[1])
		switch key {
		case "board", "dealer", "vulnerable":
			pending[key] = match[2]
		case "deal":
			pending["deal"] = match[2]
			deal, err := dealFromTags(pending)
			if err != nil {
				return nil, fmt.Errorf("board %d: %w", len(deals)+1, err)
			}
			deals = append(deals, deal)
			pending = map[string]string{}
		}
	}
	if len(deals) == 0 {
		return nil, fmt.Errorf("no board found (missing [Dealer]/[Deal] tags)")
	}
	return deals, nil
}

// dealFromTags builds a Deal from a set of already-extracted PBN tags,
// shared by ParsePBN (single board) and ParsePBNBoards (multi-board).
func dealFromTags(tags map[string]string) (*Deal, error) {
	dealerVal, ok := tags["dealer"]
	if !ok {
		return nil, fmt.Errorf("missing [Dealer] tag")
	}
	dealer, ok := seatIndex(strings.ToUpper(strings.TrimSpace(dealerVal)))
	if !ok {
		return nil, fmt.Errorf("invalid Dealer %q (expected N, E, S or W)", dealerVal)
	}

	dealVal, ok := tags["deal"]
	if !ok {
		return nil, fmt.Errorf("missing [Deal] tag")
	}
	hands, err := parseDeal(dealVal)
	if err != nil {
		return nil, err
	}
	return &Deal{
		Dealer: dealer,
		Hands:  hands,
		Vul:    parseVulnerable(tags["vulnerable"]),
		Board:  strings.TrimSpace(tags["board"]),
	}, nil
}

const rankOrder = "AKQJT98765432"

func parseDeal(value string) ([4]*Hand, error) {
	var hands [4]*Hand
	parts := strings.SplitN(strings.TrimSpace(value), ":", 2)
	if len(parts) != 2 {
		return hands, fmt.Errorf("invalid Deal tag: missing \"<first>:\" prefix")
	}
	first, ok := seatIndex(strings.ToUpper(strings.TrimSpace(parts[0])))
	if !ok {
		return hands, fmt.Errorf("invalid Deal tag: unknown first seat %q", parts[0])
	}
	handStrs := strings.Fields(parts[1])
	if len(handStrs) != 4 {
		return hands, fmt.Errorf("invalid Deal tag: expected 4 hands, got %d", len(handStrs))
	}

	seen := map[string]string{} // card -> seat, to detect duplicates
	for i, hs := range handStrs {
		seat := (first + i) % 4
		if hs == "-" {
			return hands, fmt.Errorf("hand for %s is not given: all four hands are required", seatNames[seat])
		}
		suits := strings.Split(hs, ".")
		if len(suits) != 4 {
			return hands, fmt.Errorf("hand for %s: expected 4 suits separated by '.', got %d", seatNames[seat], len(suits))
		}
		h := &Hand{}
		total := 0
		// PBN order within a hand: spades, hearts, diamonds, clubs.
		pbnOrder := [4]Suit{Spades, Hearts, Diamonds, Clubs}
		for si, ranks := range suits {
			suit := pbnOrder[si]
			ranks = strings.ToUpper(strings.TrimSpace(ranks))
			if ranks == "-" {
				ranks = "" // some tools write a void as "-" instead of an empty suit
			}
			for j := 0; j < len(ranks); j++ {
				r := ranks[j]
				if !strings.ContainsRune(rankOrder, rune(r)) {
					return hands, fmt.Errorf("hand for %s: invalid rank %q", seatNames[seat], string(r))
				}
				card := string(r) + strainEN[suit]
				if owner, dup := seen[card]; dup {
					return hands, fmt.Errorf("card %s appears in both %s and %s", card, owner, seatNames[seat])
				}
				seen[card] = seatNames[seat]
			}
			h.Suits[suit] = sortRanks(ranks)
			total += len(ranks)
		}
		if total != 13 {
			return hands, fmt.Errorf("hand for %s has %d cards, expected 13", seatNames[seat], total)
		}
		hands[seat] = h
	}
	return hands, nil
}

func sortRanks(ranks string) string {
	var sb strings.Builder
	for i := 0; i < len(rankOrder); i++ {
		if strings.IndexByte(ranks, rankOrder[i]) >= 0 {
			sb.WriteByte(rankOrder[i])
		}
	}
	return sb.String()
}
