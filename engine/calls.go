package engine

import "fmt"

type CallKind int

const (
	KindPass CallKind = iota
	KindBid
	KindDouble
	KindRedouble
)

// Call is one action in the auction.
type Call struct {
	Kind   CallKind
	Level  int
	Strain Strain
}

var passCall = Call{Kind: KindPass}
var doubleCall = Call{Kind: KindDouble}

func bid(level int, s Strain) Call { return Call{Kind: KindBid, Level: level, Strain: s} }

func bidSuit(level int, s Suit) Call { return bid(level, s.Strain()) }

func (c Call) IsBid() bool { return c.Kind == KindBid }

// steps ranks bids on the auction ladder (1C=0 ... 7NT=34).
func (c Call) steps() int { return (c.Level-1)*5 + int(c.Strain) }

func (c Call) higherThan(o Call) bool { return c.steps() > o.steps() }

var strainEN = [5]string{"C", "D", "H", "S", "NT"}
var strainFR = [5]string{"T", "K", "C", "P", "SA"}

// suitNameFR and suitNameEN spell out a suit's full name (Clubs..Spades),
// for comments that name an arbitrary suit rather than a fixed one.
var suitNameFR = [4]string{"Trèfle", "Carreau", "Cœur", "Pique"}
var suitNameEN = [4]string{"clubs", "diamonds", "hearts", "spades"}

// Format renders a call in the requested language ("en" or "fr").
func (c Call) Format(lang string) string {
	fr := lang == "fr"
	switch c.Kind {
	case KindPass:
		if fr {
			return "Passe"
		}
		return "Pass"
	case KindDouble:
		if fr {
			return "Contre"
		}
		return "X"
	case KindRedouble:
		if fr {
			return "Surcontre"
		}
		return "XX"
	}
	names := strainEN
	if fr {
		names = strainFR
	}
	return fmt.Sprintf("%d%s", c.Level, names[c.Strain])
}

// meaning carries what a call shows; it is shared with partner (both follow
// the same system) and drives the rest of the auction.
type meaning struct {
	minPts, maxPts         int // HL-ish points shown; -1 = leave previous knowledge untouched
	lens                   [4]int
	short                  [4]bool // suits this bid discloses as a singleton or void
	forcing                bool
	invite                 bool    // invitation to game
	slamInvite             bool    // quantitative slam try
	secondSuitTry          bool    // second five-card suit named at the four level over partner's 3NT, a slam try [S-13c]
	blackwood              bool    // 4NT keycard ask
	keyResp                bool    // answer to Blackwood
	relay                  bool    // artificial relay (Stayman, 2C/2D relays...)
	cuebid                 bool    // bid of the opponents' suit in answer to a takeout double
	overcallAsk            bool    // cue-bid of the opener's suit asking the overcaller whether his intervention holds opening values
	askMax                 bool    // the overcaller's "opening values" answer to that cue-bid: the side is bound for game [A-6]
	wideDouble             bool    // answer to a takeout double made without a jump available (2H over 1S): the zone spans the weak and the middle one
	wideDoubleSuit         Suit    // the major named by that wide answer
	checkbackOffer         bool    // opener's jump 2NT rebid (18-19), 3C checkback available
	checkback              bool    // responder's 3C checkback over the jump 2NT rebid
	roudiOffer             bool    // opener's plain 1NT rebid after 1m - 1M, 2C Roudi available
	roudi                  bool    // responder's 2C Roudi over the 1NT rebid
	fourthSuit             bool    // « quatrième couleur forcing » : artificial bid of the one suit our side has left unnamed, asking partner to describe his hand
	fourthSuitSuit         Suit    // the suit that ask names
	thirdSuit              bool    // « troisième couleur forcing » : after 1m - 1M - 2m, artificial bid of the suit above the opening minor, asking opener for the third card in the major, the stopper, or his choice of contract
	thirdSuitSuit          Suit    // the suit that ask names
	thirdSuitDenied        bool    // opener's answer to the third suit forcing denied both the support and the stopper
	hasTexas               bool    // transfer bid
	texas                  Suit    // suit shown by the transfer
	michaels               bool    // specified Michaels two-suited overcall
	mSuits                 [2]Suit // the two suits shown by a Michaels overcall
	landy                  bool    // 2C Landy: both majors, at least 5-4, over an opposing 1NT opening
	bothMajors             bool    // 4D over partner's 1NT: 5-5 in the majors, opener picks his better one
	reopen                 bool    // balancing bid (réveil): the opening came back after two passes
	reopenAsk              bool    // advancer's cue-bid over a suit réveil: game hope, opening values
	controlBid             bool    // cue-bid showing a control during slam exploration
	controlSuit            Suit    // the suit in which the control is shown
	ctrlRelay              bool    // « relais contrôle » : conventional ask for the control in ctrlRelaySuit
	ctrlRelaySuit          Suit    // the control the relay asks for (clubs for 3SA, spades for 3S)
	ctrlSlamDoubt          bool    // 3SA « oui mais » : interested in slam, but a contra-indication
	deniedCtrl             [4]bool // suits denied (skipped over) by this control bid
	keycardShown           bool    // this control bid specifically revealed a keycard (an ace, or the trump king) rather than a void/singleton/side king
	partialSlamAccept      bool    // quantitative 4NT answered at the intermediate step (5-level)
	splinter               bool    // double-jump splinter, singleton or void in the fit
	helpSuitTry            bool    // natural help-suit game try: partner must show real help, not just points
	helpSuit               Suit    // the suit named by a help-suit game try
	rubensohlDouble        bool    // Rubensohl: positive double of the intervention, tendency Stayman
	rubensohlAsk           bool    // Rubensohl: "impossible Texas", singleton in the intervention suit, asks for the other major(s)
	rubensohlStopperAsk    bool    // Rubensohl: 3S, generic stopper ask for 3NT regardless of the intervention suit (SEF 2018 p.29)
	staymanBothMajorsRelay bool    // Stayman confirmed both majors: game-proposing transfer, opener rectifies to 3 or 4 by strength
	openerMinorRebid       bool    // opener's plain or jump repeat of the opening minor over our 1NT response
	minorStopperAsk        bool    // responder's artificial major-suit bid asking opener for a stopper there, to play notrump
	minorStopperDenied     bool    // opener's denial of the stopper ask, back to the opening minor
	misereDoreeAsk         bool    // "golden misery" Stayman: reversed 2SA game proposal when our major ranks below opener's Stayman answer
	misereDoreeSuit        Suit    // the five-card major shown by the misère dorée
	misereDoreeFit         bool    // opener's forcing reply confirming three-card support for the misère dorée major
	affranchieCorrection   bool    // opener's rectification to the real suit over the 3NT solid-minor opening
	drury                  bool    // 2C Drury: passed hand, three-card (or singleton-less four-card) fit and 11+ HLD facing a third/fourth-seat major opening
	druryShort             bool    // 2NT Drury: the same, with four trumps and a singleton
	druryGameTry           bool    // opener's artificial 2D over the Drury: game ambition, asks responder to show his support length
	druryShortAsk          bool    // opener's 3C over the 2NT Drury: which singleton?
	reopenNTInvite         bool    // the 2SA answer to our réveil: 13-15 H with a stopper, an invitation the reopener answers
	spoutnik               bool    // contre Spoutnik: exactly four cards in the unbid major(s), after an opposing overcall
	spoutnikSuits          [4]bool // the major(s) that Spoutnik double promises
	lawBid                 bool    // competitive bid bought on the law of total tricks alone -- a sacrifice, or the raise the fit covers -- rather than on points
	lawSuit                Suit    // the fit that law bid spent
	reverse                bool    // « bicolore cher » : opener's second suit named above the first, 18 HL and up, auto-forcing
	reverseBrake           bool    // « 2SA modérateur » : responder's brake over that reverse, 5-7 H, forcing one round
	stops                  [4]bool // suits in which this bid guarantees a stopper (e.g. a notrump bid "avec arrêt")
	fr, en                 string
}

func noInfo() meaning { return meaning{minPts: -1, maxPts: -1} }

func m(min, max int, fr, en string) meaning {
	return meaning{minPts: min, maxPts: max, fr: fr, en: en}
}

func (mn meaning) withLen(s Suit, n int) meaning {
	if n > mn.lens[s] {
		mn.lens[s] = n
	}
	return mn
}

// withShort records a disclosed singleton or void in s: the length floor, as
// withLen, plus a flag the slam-zone code reads so it never asks this hand to
// cue-bid a shortness partner already knows about (docs/regles_moteur.md
// [S-2b], [S-3b]).
func (mn meaning) withShort(s Suit, n int) meaning {
	mn = mn.withLen(s, n)
	mn.short[s] = true
	return mn
}

func (mn meaning) asForcing() meaning { mn.forcing = true; return mn }
func (mn meaning) asInvite() meaning  { mn.invite = true; return mn }

// withStopper records that the bid guarantees a stopper in s, so partner may
// count the enemy suit as held when deciding a notrump game.
func (mn meaning) withStopper(s Suit) meaning { mn.stops[s] = true; return mn }

func (mn meaning) asMichaels(s1, s2 Suit) meaning {
	mn.michaels = true
	mn.mSuits = [2]Suit{s1, s2}
	return mn
}

// asReopen marks a balancing call (réveil). The zones of the balancing seat
// are not those of the direct seat, so the advancer must know which table to
// read before answering (docs/regles_moteur.md §9.3 and §9.5).
func (mn meaning) asReopen() meaning { mn.reopen = true; return mn }

// asLawBid marks a bid bought on the law of total tricks: the trump length,
// not the points, paid for the level. The fit is recorded because the law
// gives one total per deal -- once a fit has bought a level, those same
// trumps cannot buy another.
func (mn meaning) asLawBid(fit Suit) meaning {
	mn.lawBid, mn.lawSuit = true, fit
	return mn
}

// asLandy marks the 2C overcall of an opposing 1NT opening: a major
// two-suiter, at least 5-4, described in a single bid.
func (mn meaning) asLandy() meaning {
	mn.landy = true
	mn.mSuits = [2]Suit{Hearts, Spades}
	return mn
}

// SeatCall is a call as recorded in the auction.
type SeatCall struct {
	Seat  int
	Call  Call
	M     meaning
	Trace []traceStep // the decision's path, when that situation is traced (see trace.go)
}
