package engine

// Suit indexes follow the bidding ladder: clubs lowest, spades highest.
type Suit int

const (
	Clubs Suit = iota
	Diamonds
	Hearts
	Spades
)

// Strain adds NoTrump on top of the four suits.
type Strain int

const (
	SClubs Strain = iota
	SDiamonds
	SHearts
	SSpades
	SNoTrump
)

var honorValue = map[byte]int{'A': 4, 'K': 3, 'Q': 2, 'J': 1}

// Hand holds one player's thirteen cards, ranks in descending order per suit.
type Hand struct {
	Suits [4]string // indexed by Suit
}

func (h *Hand) Len(s Suit) int { return len(h.Suits[s]) }

func (h *Hand) sortedLens() [4]int {
	l := [4]int{h.Len(Clubs), h.Len(Diamonds), h.Len(Hearts), h.Len(Spades)}
	for i := range 3 {
		for j := i + 1; j < 4; j++ {
			if l[j] > l[i] {
				l[i], l[j] = l[j], l[i]
			}
		}
	}
	return l
}

// Type classifies the hand per the SEF categories.
func (h *Hand) Type() string {
	atLeast4 := 0
	for s := Clubs; s <= Spades; s++ {
		if h.Len(s) >= 4 {
			atLeast4++
		}
	}
	l := h.sortedLens()
	switch {
	case atLeast4 >= 3:
		return "three-suited"
	case l[0] >= 6 && l[1] < 4:
		return "single-suited"
	case l[0] >= 5 && atLeast4 >= 2:
		return "two-suited"
	default:
		return "regular"
	}
}
