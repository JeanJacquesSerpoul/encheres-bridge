package engine

// Hand features the rule conditions read (tools/python_tools/SEF_2024_spec.md
// §4). Every formula is a line-for-line port of features(), ptricks() and
// quick_tricks() in tools/python_tools/sef_rules.py: the conformance test
// replays the feature dump of tests.json, so any drift shows at once.

import "sort"

// ruleSuits lists the suits in the rules' own order, spades first, with the
// letter a condition names them by.
var ruleSuits = [4]Suit{Spades, Hearts, Diamonds, Clubs}

var suitLetter = [4]string{"C", "D", "H", "S"} // indexed by Suit

func suitByLetter(l string) (Suit, bool) {
	for s, n := range suitLetter {
		if n == l {
			return Suit(s), true
		}
	}
	return 0, false
}

// features is everything a condition can name, computed once per hand.
type features struct {
	h            *Hand
	lens         [4]int     // indexed by Suit
	hcpIn        [4]int     // indexed by Suit
	qt           [4]float64 // quick tricks per suit
	hcp, hl      int
	dh, hld      int
	aces, kings  int
	losers       int
	ptricks      float64
	qtricks      float64
	sidetricks   float64
	shape        string
	balanced     bool
	semibalanced bool

	// Context, not the hand: the vulnerability of the bidder's side and of
	// the opponents, read by the conditions as vul and opp_vul, and the
	// bidder's rank in the auction (1 = dealer … 4), read as seat.
	vul, oppVul bool
	seat        int
}

func has(cards string, rank byte) bool {
	for i := 0; i < len(cards); i++ {
		if cards[i] == rank {
			return true
		}
	}
	return false
}

func count(cards string, rank byte) int {
	n := 0
	for i := 0; i < len(cards); i++ {
		if cards[i] == rank {
			n++
		}
	}
	return n
}

// suitPtricks is one suit's share of ptricks(): the sure tricks of its top
// honours, capped by its length, plus half a trick for the fourth card and
// one per card from the fifth.
func suitPtricks(c string, n int) float64 {
	a, k, q := has(c, 'A'), has(c, 'K'), has(c, 'Q')
	t := 0.0
	if a {
		t = 1
	}
	if k && n >= 2 {
		if a {
			t += 1
		} else {
			t += 0.5
		}
	}
	if q && n >= 3 {
		switch {
		case a && k:
			t += 1
		case a || k:
			t += 0.5
		}
	}
	t = min(float64(n), t)
	if n >= 4 {
		t += 0.5 + float64(n-4)
	}
	return t
}

// isHonor: the A, K, Q and J of the evaluation.
func isHonor(r byte) bool { return r == 'A' || r == 'K' || r == 'Q' || r == 'J' }

// lengthPoints: one point per card from the fifth, in a suit headed by at
// least Q-J (two of its A, K, Q, J).
func lengthPoints(c string, n int) int {
	h := 0
	for i := 0; i < n; i++ {
		if isHonor(c[i]) {
			h++
		}
	}
	if h < 2 {
		return 0
	}
	return max(0, n-4)
}

// devalued: a bare honour (the ace too) or two bare honours cost a point.
func devalued(c string, n int) bool {
	switch n {
	case 1:
		return isHonor(c[0])
	case 2:
		return isHonor(c[0]) && isHonor(c[1])
	}
	return false
}

// twoSuiterBonus: a 6-5 two-suiter or longer (the two longest suits make
// eleven cards or more, the second has at least five) is worth 4 more HLD
// once a fit is found, its second suit being set up by ruffs.
func twoSuiterBonus(longest, second int) int {
	if second >= 5 && longest+second >= 11 {
		return 4
	}
	return 0
}

// quickTricks: AK 2; AQ 1.5; A 1; KQ 1; K (at least second) 0.5.
func quickTricks(c string, n int) float64 {
	switch {
	case has(c, 'A') && has(c, 'K'):
		return 2
	case has(c, 'A') && has(c, 'Q'):
		return 1.5
	case has(c, 'A'):
		return 1
	case has(c, 'K') && has(c, 'Q'):
		return 1
	case has(c, 'K') && n >= 2:
		return 0.5
	}
	return 0
}

func newFeatures(h *Hand) *features {
	f := &features{h: h}
	shortp, deval := 0, 0
	for s := Clubs; s <= Spades; s++ {
		c := h.Suits[s]
		n := len(c)
		f.lens[s] = n
		for i := 0; i < n; i++ {
			f.hcpIn[s] += honorValue[c[i]]
		}
		f.hcp += f.hcpIn[s]
		switch n {
		case 0:
			shortp += 3
		case 1:
			shortp += 2
		case 2:
			shortp += 1
		}
		f.hl += lengthPoints(c, n)
		if devalued(c, n) {
			deval++
		}
		f.aces += count(c, 'A')
		f.kings += count(c, 'K')
		f.ptricks += suitPtricks(c, n)
		f.qt[s] = quickTricks(c, n)
		f.qtricks += f.qt[s]

		// Losers: the top min(n, 3) cards against the A, K, Q they should be.
		top := min(n, 3)
		for _, w := range []byte("AKQ")[:top] {
			if !has(c[:top], w) {
				f.losers++
			}
		}
	}
	f.hl += f.hcp - deval
	f.dh = f.hcp + shortp - deval
	f.hld = f.hl + shortp

	lens := []int{f.lens[0], f.lens[1], f.lens[2], f.lens[3]}
	sort.Sort(sort.Reverse(sort.IntSlice(lens)))
	for _, n := range lens {
		f.shape += string(rune('0' + n))
	}
	f.hld += twoSuiterBonus(lens[0], lens[1])
	f.balanced = f.shape == "4333" || f.shape == "4432" || f.shape == "5332"
	f.semibalanced = f.shape == "5422" || f.shape == "6322"

	// Longest suit, ties to the higher one (spades first, as max() over "SHDC").
	longest := ruleSuits[0]
	for _, s := range ruleSuits[1:] {
		if f.lens[s] > f.lens[longest] {
			longest = s
		}
	}
	f.sidetricks = f.qtricks - f.qt[longest]
	return f
}

// scalar returns the value of a plain (non-function) feature name.
func (f *features) scalar(name string) (value, bool) {
	switch name {
	case "S", "H", "D", "C":
		s, _ := suitByLetter(name)
		return intVal(f.lens[s]), true
	case "hcp":
		return intVal(f.hcp), true
	case "hl":
		return intVal(f.hl), true
	case "dh":
		return intVal(f.dh), true
	case "hld":
		return intVal(f.hld), true
	case "shape":
		return strVal(f.shape), true
	case "balanced":
		return boolVal(f.balanced), true
	case "semibalanced":
		return boolVal(f.semibalanced), true
	case "aces":
		return intVal(f.aces), true
	case "kings":
		return intVal(f.kings), true
	case "losers":
		return intVal(f.losers), true
	case "ptricks":
		return floatVal(f.ptricks), true
	case "qtricks":
		return floatVal(f.qtricks), true
	case "sidetricks":
		return floatVal(f.sidetricks), true
	case "vul":
		return boolVal(f.vul), true
	case "opp_vul":
		return boolVal(f.oppVul), true
	case "seat":
		return intVal(f.seat), true
	}
	return value{}, false
}

// suitFuncs are the per-suit functions a condition calls with 'S', 'H', ...
var suitFuncs = map[string]func(f *features, s Suit) value{
	"ace":   func(f *features, s Suit) value { return boolVal(has(f.h.Suits[s], 'A')) },
	"king":  func(f *features, s Suit) value { return boolVal(has(f.h.Suits[s], 'K')) },
	"queen": func(f *features, s Suit) value { return boolVal(has(f.h.Suits[s], 'Q')) },
	"top": func(f *features, s Suit) value {
		n := 0
		for _, r := range []byte("AKQ") {
			if has(f.h.Suits[s], r) {
				n++
			}
		}
		return intVal(n)
	},
	"solid": func(f *features, s Suit) value {
		c := f.h.Suits[s]
		return boolVal(has(c, 'A') && has(c, 'K') && has(c, 'Q'))
	},
	"stop": func(f *features, s Suit) value {
		c, n := f.h.Suits[s], f.lens[s]
		return boolVal(has(c, 'A') || (has(c, 'K') && n >= 2) || (has(c, 'Q') && n >= 3) || (has(c, 'J') && n >= 4))
	},
	"short":  func(f *features, s Suit) value { return boolVal(f.lens[s] <= 1) },
	"hcp_in": func(f *features, s Suit) value { return intVal(f.hcpIn[s]) },
	"keycards": func(f *features, s Suit) value {
		n := f.aces
		if has(f.h.Suits[s], 'K') {
			n++
		}
		return intVal(n)
	},
	"ctrl1": func(f *features, s Suit) value { return boolVal(has(f.h.Suits[s], 'A') || f.lens[s] == 0) },
	"ctrl2": func(f *features, s Suit) value {
		c := f.h.Suits[s]
		return boolVal(has(c, 'A') || has(c, 'K') || f.lens[s] <= 1)
	},
}

// featureNames are the plain names, in the order of sef_rules.py's FEATS.
var featureNames = []string{"S", "H", "D", "C", "hcp", "hl", "dh", "hld", "shape", "balanced", "semibalanced",
	"aces", "kings", "losers", "ptricks", "qtricks", "sidetricks"}

// contextNames describe the table rather than the hand: they are not part
// of a hand's feature dump.
var contextNames = []string{"vul", "opp_vul", "seat"}

// funcNames are the per-suit functions, in the order of sef_rules.py's FUNCS.
var funcNames = []string{"ace", "king", "queen", "top", "solid", "stop", "short", "hcp_in", "keycards", "ctrl1", "ctrl2"}
