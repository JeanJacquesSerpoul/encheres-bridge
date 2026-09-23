package main

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

func (s Suit) Strain() Strain { return Strain(s) }

func (s Suit) IsMajor() bool { return s == Hearts || s == Spades }

var honorValue = map[byte]int{'A': 4, 'K': 3, 'Q': 2, 'J': 1}

// Hand holds one player's thirteen cards, ranks in descending order per suit.
type Hand struct {
	Suits [4]string // indexed by Suit
}

func (h *Hand) Len(s Suit) int { return len(h.Suits[s]) }

// SuitH returns the honor points held in one suit.
func (h *Hand) SuitH(s Suit) int {
	pts := 0
	for i := 0; i < len(h.Suits[s]); i++ {
		pts += honorValue[h.Suits[s][i]]
	}
	return pts
}

// H returns honor points (A=4 K=3 Q=2 J=1).
func (h *Hand) H() int {
	pts := 0
	for s := Clubs; s <= Spades; s++ {
		pts += h.SuitH(s)
	}
	return pts
}

// L returns length points: one per card beyond the fourth in each suit.
func (h *Hand) L() int {
	pts := 0
	for s := Clubs; s <= Spades; s++ {
		if n := h.Len(s); n > 4 {
			pts += n - 4
		}
	}
	return pts
}

func (h *Hand) HL() int { return h.H() + h.L() }

// HLD returns HL plus distribution points for shortness in side suits,
// to be used once a fit in trump has been found (void=3, singleton=2, doubleton=1).
func (h *Hand) HLD(trump Suit) int {
	pts := h.HL()
	for s := Clubs; s <= Spades; s++ {
		if s == trump {
			continue
		}
		switch h.Len(s) {
		case 0:
			pts += 3
		case 1:
			pts += 2
		case 2:
			pts += 1
		}
	}
	return pts
}

func (h *Hand) Aces() int {
	n := 0
	for s := Clubs; s <= Spades; s++ {
		if len(h.Suits[s]) > 0 && h.Suits[s][0] == 'A' {
			n++
		}
	}
	return n
}

// Keycards counts the four aces plus the king of trump.
func (h *Hand) Keycards(trump Suit) int {
	n := h.Aces()
	for i := 0; i < len(h.Suits[trump]); i++ {
		if h.Suits[trump][i] == 'K' {
			n++
		}
	}
	return n
}

// Kings counts the four kings.
func (h *Hand) Kings() int {
	n := 0
	for s := Clubs; s <= Spades; s++ {
		if h.HasCard(s, 'K') {
			n++
		}
	}
	return n
}

// KingKeys counts the four kings plus the trump queen -- the "keys" of the
// king-ask Blackwood used after a 2D opening, where the trump queen replaces
// the trump king as the fifth key.
func (h *Hand) KingKeys(trump Suit) int {
	n := h.Kings()
	if h.HasCard(trump, 'Q') {
		n++
	}
	return n
}

func (h *Hand) HasCard(s Suit, rank byte) bool {
	for i := 0; i < len(h.Suits[s]); i++ {
		if h.Suits[s][i] == rank {
			return true
		}
	}
	return false
}

// Longest returns the longest suit; ties go to the higher-ranking suit.
func (h *Hand) Longest() Suit {
	best := Clubs
	for s := Diamonds; s <= Spades; s++ {
		if h.Len(s) >= h.Len(best) {
			best = s
		}
	}
	return best
}

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

// IsRegular reports 4-3-3-3, 4-4-3-2 or 5-3-3-2 shapes.
func (h *Hand) IsRegular() bool {
	l := h.sortedLens()
	if l[3] <= 1 {
		return false
	}
	doubletons := 0
	for _, n := range l {
		if n == 2 {
			doubletons++
		}
	}
	return doubletons <= 1
}

// IsSemiRegular reports 5-4-2-2 or 6-3-2-2 shapes: the two doubletons are
// what makes the hand irregular, and the longest suit is what keeps it
// playable in notrump. Without that second bound the two doubletons alone
// also admitted 7-2-2-2 -- the only longer shape three doubletons leave room
// for, a unicolore no notrump decision should treat as balanced, and one the
// six-card-major promotions, gated on "not regular and not semi-regular",
// could never reach.
func (h *Hand) IsSemiRegular() bool {
	l := h.sortedLens()
	return l[0] <= 6 && l[2] == 2 && l[3] == 2
}

// notrumpSlamShape reports whether the hand can carry a notrump slam try
// [S-13b]. Twelve tricks at notrump need every suit to bring something: a
// void hands the defence a free suit to cash. A lone singleton is survivable --
// the other three suits still hold the twelve tricks -- which is the whole
// difference with the balanced shapes the notrump *openings* promise; but
// beside a six-card suit it is not a notrump hand at all, and the slam it
// wants belongs in that suit (measured: the old shape gate was the only
// thing keeping AKQJT87 out of a quantitative 4SA).
func (h *Hand) notrumpSlamShape() bool {
	l := h.sortedLens()
	if l[3] == 0 {
		return false // a void: a whole suit handed to the defence
	}
	if l[3] == 1 && l[0] > 5 {
		return false // a singleton beside a long suit: the slam is in the suit
	}
	return l[0] <= 6
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

// GoodSuit reports a suit worth overcalling or preempting in.
func (h *Hand) GoodSuit(s Suit) bool {
	if h.Len(s) < 5 {
		return false
	}
	top := 0
	for _, r := range []byte{'A', 'K', 'Q', 'J', 'T'} {
		if h.HasCard(s, r) {
			top++
		}
	}
	return top >= 2 || h.SuitH(s) >= 5
}

// Stopper reports a likely stopper: A, Kx, Qxx or Jxxx.
func (h *Hand) Stopper(s Suit) bool {
	n := h.Len(s)
	switch {
	case h.HasCard(s, 'A'):
		return true
	case h.HasCard(s, 'K') && n >= 2:
		return true
	case h.HasCard(s, 'Q') && n >= 3:
		return true
	case h.HasCard(s, 'J') && n >= 4:
		return true
	}
	return false
}

// DoubleStopper reports two independent stops in s -- what it takes to survive a
// suit the opponents have bid and raised (a long, running holding) rather than a
// lone hold they knock out before cashing the rest. A bare doubleton ace (Ax)
// stops the suit only once and fails here.
func (h *Hand) DoubleStopper(s Suit) bool {
	n := h.Len(s)
	hasA := h.HasCard(s, 'A')
	hasK := h.HasCard(s, 'K')
	hasQ := h.HasCard(s, 'Q')
	switch {
	case hasA && hasK: // AK(x): two top winners
		return true
	case hasA && hasQ && n >= 3: // AQx
		return true
	case hasK && hasQ && n >= 4: // KQxx
		return true
	case hasA && n >= 5: // Axxxx: length itself buys a second stop via hold-up
		return true
	}
	return false
}

// StopperAndHalf reports the "arrêt et demi" a 2NT answer to the takeout double
// promises: a stopper reinforced by a second honour or by a fourth card -- AJx,
// KJx, Kxxx -- without reaching the two independent stops of DoubleStopper.
func (h *Hand) StopperAndHalf(s Suit) bool {
	if !h.Stopper(s) {
		return false
	}
	if h.DoubleStopper(s) {
		return true
	}
	honours := 0
	for _, r := range []byte{'A', 'K', 'Q', 'J', 'T'} {
		if h.HasCard(s, r) {
			honours++
		}
	}
	return honours >= 2 || h.Len(s) >= 4
}
