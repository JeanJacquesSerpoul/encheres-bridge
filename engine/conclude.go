package engine

import (
	"fmt"
	"slices"
)

// ---------- conclude: priority-ordered handler table ----------
//
// Once the descriptive phase of an auction is over, conclude decides the
// next call. Most of what it does is routing: does this bid answer a
// specific convention partner just started (Blackwood, Stayman, a
// checkback, a splinter continuation...)? Each convention owns one
// self-contained rule, tried in a fixed priority order -- the first one
// that both applies and actually produces a legal call wins. concludeHandlers
// makes that order an explicit, ordered table instead of a long implicit
// chain of "if" statements: a new convention is one more table entry, and
// the priority between conventions is visible at a glance instead of buried
// in nesting.
//
// A handler's condition (when) can hold while its own attempt still finds no
// legal call -- interference stole the room the natural continuation needed,
// or a shape test inside the handler fails once the hand is examined more
// closely. run signals this with its bool return: false means "did not
// handle it after all," so the dispatcher moves on to the next handler in
// the table, exactly as the original sequential code fell through to the
// next "if" block below it. When nothing in the table claims the call, the
// generic value-based endgame in concludeGameDecision takes over: game,
// invitation, staying forced, or the last-resort game-forcing net.

// concludeCtx bundles the read-only context every handler needs, computed
// once per call to conclude so individual handlers don't each recompute the
// agreed fit, the combined point ranges, or whose turn just came back.
type concludeCtx struct {
	p                *playerState
	partner          *playerState
	pm               *meaning
	last             Call
	lastSeat         int
	hasBid           bool
	ours             bool // the last bid belongs to our side
	partnerJustActed bool
	side             int
	fit              Suit
	hasFit           bool
	own              int // this hand's value: H, or HLD(fit) once a fit is agreed
	cMin, cMax       int // combined point range: own plus partner's shown range
	cMinNT, cMaxNT   int // cMin/cMax valued for notrump (honours only, no fit distribution)
	cMinSlam         int // cMin with shortness facing partner's shown length written off: what a slam may lean on
	cMaxSlam         int // the same discount applied to cMax: the best a slam could count on
	ntOK             bool
	tr               *tracer // the decision trace, when conclude decides the call itself (see concludeFrom)
}

// concludeContext computes the shared state threaded through every handler
// in concludeHandlers, and through concludeGameDecision.
func (e *Engine) concludeContext(p *playerState) *concludeCtx {
	partner := e.ps[partnerOf(p.seat)]
	pm := partner.lastM
	last, lastSeat, hasBid := e.lastBid()
	ours := hasBid && sideOf(lastSeat) == sideOf(p.seat)
	partnerJustActed := len(e.calls) > 0 && e.calls[len(e.calls)-1].Seat == partnerOf(p.seat) ||
		(len(e.calls) > 1 && e.calls[len(e.calls)-2].Seat == partnerOf(p.seat) && e.calls[len(e.calls)-1].Call.Kind == KindPass)

	side := sideOf(p.seat)
	fit, hasFit := e.fitSuit(p)
	if !hasFit && e.bw[side].asked && !e.bw[side].noTrump {
		// Once the side has launched Blackwood in a suit, that trump is
		// agreed for both partners, even if the shorter hand never had
		// enough cards there to declare the fit via shownLens itself.
		fit, hasFit = e.bw[side].trump, true
	}
	own := p.hand.H()
	if hasFit {
		own = p.hand.HLD(fit)
	}
	cMin := own + partner.shownMin
	cMax := own + partner.shownMax
	// Notrump-valued combined minimum: count honours only, dropping the
	// trump-fit distribution points that HLD adds for a minor fit. They do
	// not translate into notrump tricks, so a 3NT decision must not lean on
	// them.
	cMinNT := p.hand.H() + partner.shownMin
	cMaxNT := p.hand.H() + partner.shownMax
	// Slam-valued minimum. A game can be made on a ruff the count only half
	// believes in; twelve tricks cannot, and the points that vanish first are
	// the ones HLD grants for shortness facing partner's own long suit. They
	// stay in cMin -- the game decisions are calibrated on them -- and come
	// out here, where every point has to be a trick.
	cMinSlam, cMaxSlam := cMin, cMax
	if hasFit {
		hldFacing := e.hldFacingPartner(p, fit)
		cMinSlam, cMaxSlam = hldFacing+partner.shownMin, hldFacing+partner.shownMax
	}
	ntOK := e.ntSafe(p)

	return &concludeCtx{
		p: p, partner: partner, pm: pm,
		last: last, lastSeat: lastSeat, hasBid: hasBid,
		ours: ours, partnerJustActed: partnerJustActed, side: side,
		fit: fit, hasFit: hasFit, own: own,
		cMin: cMin, cMax: cMax, cMinNT: cMinNT, cMaxNT: cMaxNT,
		cMinSlam: cMinSlam, cMaxSlam: cMaxSlam, ntOK: ntOK,
	}
}

// concludeHandler is one self-contained rule tried by conclude, in the
// table's order. when reports whether the rule applies to the current
// auction. run makes the actual attempt; its bool return is false when the
// rule applied but produced no legal call of its own (interference took the
// room, or an inner shape test failed), in which case the dispatcher falls
// through to the next handler exactly as the original nested "if" once fell
// through to the next block below it.
type concludeHandler struct {
	name string
	when func(e *Engine, ctx *concludeCtx) bool
	run  func(e *Engine, ctx *concludeCtx) (Call, meaning, bool)
	// fr, en name the situation for the decision trace, on the handlers whose
	// run hands the call to an instrumented decision. The others are not
	// instrumented yet: a call they make carries no trace.
	fr, en string
}

// conclude drives the auction once the descriptive phase is over: it tries
// every convention-specific rule in concludeHandlers, in priority order, then
// falls back to the generic value-based endgame.
func (e *Engine) conclude(p *playerState) (Call, meaning) {
	return e.concludeFrom(p, nil)
}

// concludeHandoff hands a decision to conclude with the trace kept: the tests
// already recorded are the route that led here, and conclude's own decision
// follows them.
func (e *Engine) concludeHandoff(p *playerState) (Call, meaning) {
	e.tr.note("→ décision de la suite de l'enchère", "→ later-auction decision")
	e.tr.in()
	return e.concludeFrom(p, e.tr)
}

// concludeFrom is conclude with its own decision trace: decide passes the
// engine's tracer when conclude is the decision itself. Reached from another
// decision (a response that lets the fit be valued directly...), conclude
// records nothing of its own, since the caller may still discard its answer.
func (e *Engine) concludeFrom(p *playerState, tr *tracer) (Call, meaning) {
	ctx := e.concludeContext(p)
	ctx.tr = tr
	for _, h := range concludeHandlers {
		if !h.when(e, ctx) {
			continue
		}
		mark := tr.mark()
		if h.fr != "" {
			tr.note(h.fr, h.en)
		}
		if c, mn, ok := h.run(e, ctx); ok {
			if tr != nil && h.fr == "" {
				return e.untraced(c, mn)
			}
			return c, mn
		}
		tr.rewind(mark)
	}
	return e.concludeGameDecision(ctx)
}

// concludeHandlers is populated by an init func rather than a plain var
// initializer: several handlers close over decisions.go functions (e.g.
// afterStayman) that themselves call back into conclude, which reads
// concludeHandlers. A var initializer expression referencing that chain
// would be a compile-time initialization cycle; assigning inside init()
// sidesteps it, since ordinary function bodies (unlike initializer
// expressions) are allowed to reference each other circularly.
var concludeHandlers []concludeHandler

func init() {
	concludeHandlers = []concludeHandler{
		{
			// A forcing double from partner (an opener's reopening double
			// after interference, docs/addon_1.md: "toutes distributions")
			// is takeout-shaped and demands an answer -- conclude otherwise
			// has no notion of it and falls straight to the generic
			// value-based endgame, which can pass a hand with real combined
			// values simply for lack of a known fit or suit. Route it
			// through the same zone logic advance() uses for the defending
			// side's takeout doubles. The forcing check excludes the one
			// other double conclude produces on this side of the auction,
			// the non-forcing penalty double of a sacrifice
			// (concludeHandlers' "penalty-double-of-sacrifice"), which
			// partner must be left to pass.
			name: "answer-partner-double",
			fr:   "le partenaire a contré (contre forcing) : répondre au contre",
			en:   "partner doubled (forcing double): answer the double",
			when: func(e *Engine, ctx *concludeCtx) bool {
				psc, ok := e.lastCallBy(partnerOf(ctx.p.seat))
				return ok && psc.Call.Kind == KindDouble && psc.M.forcing
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				oppSuit, _ := e.doubledSuit(partnerOf(p.seat))
				freePosition := false
				if n := len(e.calls); n > 0 {
					rho := e.calls[n-1]
					freePosition = rho.Call.IsBid() && sideOf(rho.Seat) != sideOf(p.seat)
				}
				c, mn := e.answerDouble(p, oppSuit, !freePosition)
				return c, mn, true
			},
		},
		{
			name: "answer-blackwood",
			fr:   "le partenaire a demandé les clefs (Blackwood) : répondre",
			en:   "partner asked for keycards (Blackwood): answer",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.blackwood && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				switch {
				case e.bw[ctx.side].kingAsk && e.bw[ctx.side].noTrump:
					c, mn := e.kingAnswerNT(p, ctx.tr)
					return c, mn, true
				case e.bw[ctx.side].kingAsk:
					c, mn := e.kingAnswer(p, e.bw[ctx.side].trump, ctx.tr)
					return c, mn, true
				case e.bw[ctx.side].noTrump:
					c, mn := e.keycardAnswerNT(p, ctx.tr)
					return c, mn, true
				default:
					c, mn := e.keycardAnswer(p, e.bw[ctx.side].trump, ctx.tr)
					return c, mn, true
				}
			},
		},
		{
			name: "continue-after-my-blackwood",
			fr:   "ma demande de clefs a reçu sa réponse : fixer le contrat",
			en:   "my keycard ask has been answered: place the contract",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return e.bw[ctx.side].asked && e.bw[ctx.side].asker == ctx.p.seat && ctx.pm != nil && ctx.pm.keyResp
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				switch {
				case e.bw[ctx.side].kingAsk && e.bw[ctx.side].noTrump:
					c, mn := e.afterKingsNT(p, ctx.tr)
					return c, mn, true
				case e.bw[ctx.side].kingAsk:
					c, mn := e.afterKings(p, ctx.tr)
					return c, mn, true
				case e.bw[ctx.side].noTrump:
					c, mn := e.afterKeycardsNT(p, ctx.tr)
					return c, mn, true
				default:
					c, mn := e.afterKeycards(p, ctx.tr)
					return c, mn, true
				}
			},
		},
		{
			// Answer a Stayman relay that arrives after our strong 2C/2D
			// opening's balanced 2NT rebid: the ask is opener's third call, so
			// it lands here rather than in openerRebid. Show the four-card
			// major(s) exactly as over a 2NT opening (with none, 3D denies
			// both).
			name: "stayman-relay-after-strong-2nt",
			fr:   "Stayman du partenaire après la redemande 2SA de l'ouverture forte",
			en:   "partner's Stayman after the strong opening's 2NT rebid",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.relay && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat) &&
					(e.openCall == bid(2, SClubs) || e.openCall == bid(2, SDiamonds)) && ctx.last == bid(3, SClubs)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.staymanAnswer(ctx.p, 2)
				return c, mn, true
			},
		},
		{
			// Answer a minor-suit Texas that arrives after our strong 2C/2D
			// opening's balanced 2NT rebid: like the Stayman case, opener's
			// answer is its third call, so it lands here rather than in
			// openerRebid. minorTransferAnswer's "good fit" decline (2NT) is
			// naturally skipped as illegal at this level, leaving the plain
			// rectification.
			name: "minor-texas-after-strong-2nt",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.hasTexas && !ctx.pm.texas.IsMajor() && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat) &&
					(e.openCall == bid(2, SClubs) || e.openCall == bid(2, SDiamonds))
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.minorTransferAnswer(ctx.p, ctx.pm.texas)
				return c, mn, true
			},
		},
		{
			// Answer a major-suit Texas that arrives after opener's third call
			// or later -- e.g. the game-forcing minor relay of
			// afterStaymanBothMajors (docs/bidings.md, "Les transferts") or the
			// misère dorée's own transfer to a known fit (docs/addon_11.md):
			// openerRebid only wires up transferAnswer for opener's immediate
			// second call, so a Texas that surfaces later needs the same
			// rectification here.
			name: "major-texas-late",
			fr:   "Texas majeur du partenaire : rectifier",
			en:   "partner's major transfer: complete it",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.hasTexas && ctx.pm.texas.IsMajor() && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.transferAnswer(ctx.p, ctx.pm.texas)
				return c, mn, true
			},
		},
		{
			// Over our strong 2D opening, opener's default balanced rebid (2NT,
			// 24HL+) is treated like a 2NT opening: holding a four-card major,
			// look for a 4-4 major fit through Stayman before settling for a
			// flat notrump conclusion.
			name: "strong-2d-stayman-for-majors",
			when: func(e *Engine, ctx *concludeCtx) bool {
				p := ctx.p
				return p.answeredAces && ctx.hasBid && ctx.lastSeat == partnerOf(p.seat) && ctx.last == bid(2, SNoTrump) &&
					e.openCall == bid(2, SDiamonds) && (p.hand.Len(Hearts) >= 4 || p.hand.Len(Spades) >= 4)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				c := bid(3, SClubs)
				if !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				mn := m(-1, -1, "Stayman, demande les majeures quatrièmes", "Stayman, asking for four-card majors").asForcing()
				mn.relay = true
				oMin := ctx.partner.shownMin
				p.planned = func() (Call, meaning) { return e.afterStayman(p, oMin) }
				return c, mn, true
			},
		},
		{
			// A responder who has so far only used the ace-step convention over
			// a strong opening (docs/bidings.md, "l'As de Carreau") may hold a
			// genuine long, strong suit that was never described. Naturally
			// agreeing it as trump here lets the control-bid/Blackwood machinery
			// take over instead of settling for a flat Sans-Atout conclusion
			// from raw point count alone.
			name: "ace-step-long-suit-as-trump",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.p.answeredAces && !ctx.hasFit && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				long := p.hand.Longest()
				if p.hand.Len(long) < 6 || !p.hand.GoodSuit(long) || p.shownLens[long] != 0 {
					return Call{}, meaning{}, false
				}
				c := e.cheapestCall(long.Strain())
				if !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				mn := m(p.hand.HLD(long), 40, "couleur longue et solide, propose l'atout", "long strong suit, proposes trump").withLen(long, p.hand.Len(long)).asForcing()
				return c, mn, true
			},
		},
		{
			// Answer partner's cue-bid asking how strong the overcall was: the
			// overcall spans 9-18 HL, and the reply splits it in two. Without
			// opening values, sign off by returning to the overcall suit at
			// the cheapest level -- nothing else may be read as extra values.
			// With them, fall through to the descriptive machinery below.
			name: "answer-overcall-strength-ask",
			fr:   "le partenaire demande la force de mon intervention par un cue-bid",
			en:   "partner asks the strength of my overcall with a cue-bid",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && (ctx.pm.overcallAsk || ctx.pm.reopenAsk) && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				// The same coded reply serves both cue-bids, on the zone the
				// first call announced: a direct overcall splits its 9-18 HL
				// at the opening, a réveil [§9.5] its 8-13 HL one rung lower,
				// since it has already denied one.
				floor, minLo, minHi, maxLo, maxHi := 12, 9, 11, 12, 18
				fr, en := "retour à ma couleur : intervention sans l'ouverture", "back to my suit: the overcall is short of opening values"
				topFR, topEN := "l'ouverture", "opening values"
				if ctx.pm.reopenAsk {
					floor, minLo, minHi, maxLo, maxHi = 11, 8, 10, 11, 13
					fr, en = "retour à ma couleur : réveil minimum", "back to my suit: minimum reopening"
					topFR, topEN = "le maximum du réveil", "the top of the reopening"
				}
				var own Suit
				ownLen := 0
				for s := Clubs; s <= Spades; s++ {
					if p.shownLens[s] >= 5 && p.hand.Len(s) > ownLen {
						own, ownLen = s, p.hand.Len(s)
					}
				}
				if ownLen == 0 {
					return Call{}, meaning{}, false
				}
				// The overcall announced a range in HL (§9.2: one level =
				// 9-18 HL), so the "opening values" cut splits it in HL too.
				// HLD would let a wild two-suiter -- 7 H, two singletons --
				// vault a minimum overcall into the describing branch and
				// bid a phantom notrump [A-6].
				if ctx.tr.check(p.hand.HL() < floor, fmt.Sprintf("moins de %d HL : minimum → retour à ma couleur au plus bas", floor),
					fmt.Sprintf("under %d HL: minimum → back to my suit at the lowest level", floor), pts(p.hand.HL(), "HL")) {
					c := e.cheapestCall(own.Strain())
					if c.Level > 3 || !e.legal(p.seat, c) {
						return Call{}, meaning{}, false
					}
					return c, m(minLo, minHi, fr, en).withLen(own, ownLen), true
				}
				// Opening values: describe, and never by the plain return to
				// the overcall suit, which is reserved for the minimum. A
				// second four-card suit first, then notrump with the
				// opponents' suits held, and failing both a jump in the
				// overcall suit.
				var oppNamed [4]bool
				for _, os := range e.opponentSuits(p) {
					oppNamed[os] = true
				}
				var second Suit
				secondLen := 0
				for _, s := range []Suit{Clubs, Diamonds, Hearts, Spades} {
					if s == own || oppNamed[s] {
						continue
					}
					if p.hand.Len(s) >= 4 && p.hand.Len(s) > secondLen {
						second, secondLen = s, p.hand.Len(s)
					}
				}
				if ctx.tr.check(secondLen >= 4, "maximum : une deuxième couleur de 4 cartes → la nommer",
					"maximum: a second four-card suit → bid it", shape(p.hand)) {
					c := e.cheapestCall(second.Strain())
					if c.Level <= 3 && e.legal(p.seat, c) {
						return c, m(maxLo, maxHi, topFR+", deuxième couleur", topEN+", second suit").withLen(second, secondLen), true
					}
				}
				stopped := true
				for _, os := range e.opponentSuits(p) {
					if !p.hand.Stopper(os) {
						stopped = false
					}
				}
				if ctx.tr.check(stopped, "maximum : leurs couleurs arrêtées → Sans-Atout", "maximum: their suits stopped → notrump", "") {
					c := e.cheapestCall(SNoTrump)
					if c.Level <= 3 && e.legal(p.seat, c) {
						return c, m(maxLo, maxHi, topFR+", arrêt dans leur couleur", topEN+", their suit held"), true
					}
				}
				c := e.cheapestCall(own.Strain())
				c = bid(c.Level+1, own.Strain())
				if ctx.tr.check(c.Level <= 4 && e.legal(p.seat, c), "maximum, sinon → saut dans ma couleur",
					"maximum, otherwise → jump in my suit", "") {
					return c, m(maxLo, maxHi, topFR+", saut dans ma couleur", topEN+", jump in my suit").withLen(own, ownLen), true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Answer partner's cue-bid (takeout-double sequences): name the
			// cheapest four-card major, else show a stopper in the opponents'
			// suit.
			name: "answer-cuebid",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.cuebid && !ctx.pm.overcallAsk && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				for _, s := range []Suit{Hearts, Spades} {
					if p.hand.Len(s) >= 4 {
						c := e.cheapestCall(s.Strain())
						if c.Level <= 3 && e.legal(p.seat, c) {
							return c, m(-1, -1, "majeure quatrième la moins chère", "cheapest four-card major").withLen(s, 4), true
						}
					}
				}
				if e.openCall.Strain <= SSpades && p.hand.Stopper(Suit(e.openCall.Strain)) {
					c := e.cheapestCall(SNoTrump)
					if c.Level <= 3 && e.legal(p.seat, c) {
						return c, m(-1, -1, "arrêt dans la couleur adverse, sans majeure quatrième", "stopper in the opponents' suit, no four-card major"), true
					}
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Checkback Stayman: over opener's jump 2NT rebid (1m - 1M - 2NT,
			// 18-19), responder's 3C asks for three-card support and the other
			// major.
			name: "checkback-offer",
			fr:   "l'ouvreur a redemandé 2SA (18-19) : Checkback 3♣ pour chercher le fit majeur",
			en:   "opener rebid 2NT (18-19): 3♣ checkback to look for the major fit",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.checkbackOffer && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat) && ctx.last == bid(2, SNoTrump)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner := ctx.p, ctx.partner
				if c, mn, ok := e.checkbackAsk(p, ctx.tr); ok {
					return c, mn, true
				}
				// No major-suit shape for the checkback: a balanced hand around
				// 14 HL may still try a quantitative 4SA rather than settle for
				// game (docs/addon_4.md, "4SA quantitatif").
				if ctx.tr.check(p.hand.HL() >= 14 && p.hand.HL()+partner.shownMin < 33 && (p.hand.IsRegular() || p.hand.IsSemiRegular()),
					"main régulière, 14 HL et plus sous la zone de chelem → 4SA quantitatif",
					"balanced, 14+ HL below the slam zone → quantitative 4NT", pts(p.hand.HL(), "HL")+", "+shape(p.hand)) {
					c := bid(4, SNoTrump)
					if e.legal(p.seat, c) {
						mn := m(14, 19, "4SA quantitatif, propose le petit chelem", "quantitative 4NT, small slam try")
						mn.slamInvite = true
						return c, mn, true
					}
				}
				return Call{}, meaning{}, false
			},
		},
		{
			name: "answer-checkback",
			fr:   "le partenaire demande par Checkback 3♣ : 3 cartes dans sa majeure ? 4 dans l'autre ?",
			en:   "partner asks with the 3♣ checkback: three cards in partner's major? four in the other?",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.checkback && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.checkbackAnswer(ctx.p, ctx.tr)
				return c, mn, true
			},
		},
		{
			// Roudi: over opener's plain 1NT rebid (1m - 1M - 1SA, 12-14),
			// responder's 2C asks for the 5-3 major fit and opener's exact zone
			// (docs/addon_9.md).
			name: "roudi-offer",
			fr:   "l'ouvreur a redemandé 1SA (12-14) : Roudi 2♣ pour chercher le fit 5-3 et sa zone",
			en:   "opener rebid 1NT (12-14): 2♣ Roudi to look for the 5-3 fit and opener's range",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.roudiOffer && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat) && ctx.last == bid(1, SNoTrump)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				if c, mn, ok := e.roudiAsk(ctx.p, ctx.tr); ok {
					return c, mn, true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Opener's rectification to the real minor over the 3NT
			// solid-minor opening (docs/bidings.md, "L'OUVERTURE DE 3SA"):
			// responder already said everything worth saying with the
			// anchor call, and now knows the real suit -- pass.
			name: "answer-affranchie-correction",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.affranchieCorrection && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				return passCall, m(-1, -1, "passe, la mineure réelle est connue", "pass, the real minor is now known"), true
			},
		},
		{
			// Drury [RM-2d]: opener's artificial 2D asked how long
			// the fit really is, his 3C which singleton the 2NT showed. Both
			// answers are the responder's, and both must come before any
			// generic valuation -- 2D is not a diamond suit to raise.
			name: "answer-drury-game-try",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.druryGameTry && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.druryGameTryAnswer(ctx.p, Suit(e.openCall.Strain))
				return c, mn, true
			},
		},
		{
			name: "answer-drury-singleton-ask",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.druryShortAsk && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.druryShortAnswer(ctx.p, Suit(e.openCall.Strain))
				return c, mn, true
			},
		},
		{
			name: "answer-roudi",
			fr:   "le partenaire demande par Roudi 2♣ : 3 cartes dans sa majeure ? minimum ou maximum ?",
			en:   "partner asks with the 2♣ Roudi: three cards in partner's major? minimum or maximum?",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.roudi && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.roudiAnswer(ctx.p, ctx.tr)
				return c, mn, true
			},
		},
		{
			// Troisième couleur forcing [§8.9]: over our plain repeat of the
			// opening minor, partner named the suit just above it,
			// artificially, to ask the three questions of the convention --
			// the third card in his major, the stopper that lets notrump be
			// played, and failing both the contract we would rather play.
			// The answer comes before any generic valuation: the third suit
			// is a question, not a suit to raise.
			name: "answer-third-suit-forcing",
			fr:   "le partenaire a nommé la troisième couleur (forcing de manche) : 3 cartes dans sa majeure ? arrêt ?",
			en:   "partner bid the third suit (game forcing): three cards in partner's major? a stopper?",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.thirdSuit && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.thirdSuitAnswer(ctx.p, ctx.pm.thirdSuitSuit, ctx.tr)
				if c.Kind == KindPass {
					return Call{}, meaning{}, false
				}
				return c, mn, true
			},
		},
		{
			// Quatrième couleur forcing [§8.8]: partner named the one suit
			// our side had left unbid, artificially, to ask for a
			// description -- three-card support for his major, a stopper to
			// play notrump, or the shape we have left. The answer owes him
			// the full extent of the hand, and must come before any generic
			// valuation: the fourth suit is not a suit to raise.
			name: "answer-fourth-suit-forcing",
			fr:   "le partenaire a nommé la quatrième couleur (forcing) : se décrire",
			en:   "partner bid the fourth suit (forcing): describe the hand",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.fourthSuit && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.fourthSuitAnswer(ctx.p, ctx.pm.fourthSuitSuit, ctx.tr)
				if c.Kind == KindPass {
					return Call{}, meaning{}, false
				}
				return c, mn, true
			},
		},
		{
			// Answering opener's reverse [RO-19b]. The "bicolore cher" is
			// auto-forcing and shows 18 HL and up, so every answer is game
			// forcing -- except the two that let the side stop below it, and
			// both must be found before the generic count, which reads
			// opener's 18 as a floor and drives to game on a hand that has
			// nothing.
			//
			// First, and by priority, the repeat of responder's own major
			// when it is five cards long: it guarantees the fifth card and
			// names a trump suit before anything else is discussed.
			//
			// Otherwise the brake, 2SA "modérateur" or "coup de frein": a
			// hand of 5-7 H warns opener that the game is his to bid, not
			// ours. Both are forcing for one round -- opener answers by
			// clarifying the strength of his reverse, and the side then knows
			// whether to settle in a partscore or bid the game.
			name: "answer-reverse",
			fr:   "l'ouvreur a fait un bicolore cher (18 HL et plus, forcing)",
			en:   "opener reversed (18+ HL, forcing)",
			when: func(e *Engine, ctx *concludeCtx) bool {
				p := ctx.p
				return ctx.pm != nil && ctx.pm.reverse && ctx.ours &&
					ctx.lastSeat == partnerOf(p.seat) &&
					p.seat != e.opener && sideOf(p.seat) == sideOf(e.opener)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				for _, s := range []Suit{Spades, Hearts} {
					if p.shownLens[s] < 4 || p.hand.Len(s) < 5 {
						continue
					}
					c := e.cheapestCall(s.Strain())
					if ctx.tr.check(c.Level <= 3 && e.legal(p.seat, c), "majeure cinquième déjà nommée → la répéter, forcing un tour",
						"five-card major already bid → repeat it, forcing one round", cards(p.hand, s)) {
						return c, m(-1, -1,
							"répétition de la majeure cinquième sur le bicolore cher, forcing un tour",
							"repeats the five-card major over the reverse, forcing one round").
							withLen(s, 5).asForcing(), true
					}
				}
				if ctx.tr.check(p.hand.H() <= 7, "5-7 H → 2SA modérateur (coup de frein)", "5-7 H → 2NT brake", pts(p.hand.H(), "H")) {
					c := bid(2, SNoTrump)
					if c.higherThan(ctx.last) && e.legal(p.seat, c) {
						mn := m(5, 7,
							"2SA modérateur (coup de frein) : main faible, l'ouvreur choisit la partielle ou la manche",
							"2NT brake: a weak hand, opener picks the partscore or the game").asForcing()
						mn.reverseBrake = true
						return c, mn, true
					}
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Opener answers the brake [RO-19b]. His reverse announced 18 as a
			// floor, and the generic count reads a floor as licence to bid the
			// game -- which is exactly what the brake exists to stop. When the
			// combined maximum, partner now capped at 7, does not reach the
			// threshold, opener retreats into his longer suit at the cheapest
			// level: a partscore responder is free to pass. With more than
			// that, the generic path below bids the game, and the brake has
			// cost nothing.
			name: "answer-reverse-brake",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.reverseBrake && ctx.ours &&
					ctx.lastSeat == partnerOf(ctx.p.seat) && ctx.p.seat == e.opener
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				if ctx.cMax >= gameThresholdFor(ctx.fit, ctx.hasFit, ctx.ntOK) {
					return Call{}, meaning{}, false
				}
				best, n := Clubs, 0
				for _, s := range []Suit{Clubs, Diamonds, Hearts, Spades} {
					if p.shownLens[s] >= 4 && p.hand.Len(s) > n {
						best, n = s, p.hand.Len(s)
					}
				}
				if n == 0 {
					return Call{}, meaning{}, false
				}
				c := e.cheapestCall(best.Strain())
				if !c.higherThan(ctx.last) || !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				// The retreat is where opener clarifies the strength of his
				// reverse: it is the minimum of the band, and saying so is
				// what lets responder pass. Left unlimited, the 18 floor
				// alone still adds up to a game facing the top of the brake.
				return c, m(18, 19,
					"bicolore cher minimum : repli dans la plus longue, le partenaire peut passer",
					"minimum reverse: retreat into the longer suit, partner may pass").withLen(best, n), true
			},
		},
		{
			// Stopper ask after opener's plain or jump repeat of the opening
			// minor over our 1NT response (1m - 1SA - 2m/3m, docs/addon_10.md).
			// A new major here cannot be natural: with real length there we
			// would have shown it directly over the opening instead of routing
			// through 1NT first. So, provided we are not a bare minimum, ask
			// for a stopper in a major we cannot guard ourselves, at the same
			// level as the rebid and never as a jump. With both majors covered,
			// bid notrump directly.
			name: "opener-minor-rebid-stopper-ask",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.openerMinorRebid && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, last := ctx.p, ctx.last
				if p.hand.H() < p.shownMin+2 {
					return Call{}, meaning{}, false
				}
				for _, M := range []Suit{Hearts, Spades} {
					if !p.hand.Stopper(M) {
						c := bid(last.Level, M.Strain())
						if e.legal(p.seat, c) {
							mn := m(-1, -1, "demande d'arrêt à "+suitNameFR[M]+" pour jouer Sans-Atout", "asks for a stopper in "+suitNameEN[M]+" to play notrump").asForcing()
							mn.minorStopperAsk = true
							return c, mn, true
						}
					}
				}
				c := bid(last.Level, SNoTrump)
				if e.legal(p.seat, c) {
					return c, m(-1, -1, "Sans-Atout, arrêts dans les deux majeures", "notrump, both majors stopped"), true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Answer the stopper ask: with the stopper, bid notrump at the same
			// level; without it, deny by returning to the opening minor at the
			// cheapest level rather than guess notrump unguarded.
			name: "answer-stopper-ask",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.minorStopperAsk && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, last := ctx.p, ctx.last
				asked := Suit(last.Strain)
				if p.hand.Stopper(asked) {
					c := bid(last.Level, SNoTrump)
					if e.legal(p.seat, c) {
						return c, m(-1, -1, "arrêt à "+suitNameFR[asked]+", Sans-Atout", "stopper in "+suitNameEN[asked]+", notrump").withStopper(asked), true
					}
				}
				os := Suit(e.openCall.Strain)
				c := e.cheapestCall(os.Strain())
				if e.legal(p.seat, c) {
					mn := m(-1, -1, "pas d'arrêt à "+suitNameFR[asked]+", retour à "+suitNameFR[os], "no stopper in "+suitNameEN[asked]+", back to "+suitNameEN[os])
					mn.minorStopperDenied = true
					return c, mn, true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// The stopper ask was denied: notrump is off, so judge the final
			// contract in the minor from the combined count alone
			// (docs/addon_10.md) -- pass, bid the game, or try the small slam.
			name: "stopper-ask-denied",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.minorStopperDenied && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, last, own, cMin := ctx.p, ctx.last, ctx.own, ctx.cMin
				os := Suit(last.Strain)
				if cMin >= 33 {
					c := bidSuit(6, os)
					if e.legal(p.seat, c) {
						return c, m(own, -1, "conclusion au petit chelem", "small slam on combined values"), true
					}
				}
				if cMin >= gameThreshold(os, true) {
					c := bidSuit(5, os)
					if e.legal(p.seat, c) {
						return c, m(own, -1, "conclusion à la manche", "bidding game"), true
					}
				}
				return passCall, m(-1, -1, "pas assez pour la manche, arrêt", "not enough for game, staying"), true
			},
		},
		{
			// The third suit forcing came back denied [§8.9]: no third card
			// in our major, and no stopper in the suit we named. Notrump
			// then rests on our own guard there -- with it, the count bids
			// the 3SA it always would; without it, nine tricks are a
			// fiction, and the game left to a side committed to one is the
			// eleven-trick game in the minor opener has now named three
			// times. Only the strain is decided here: a hand with no fit to
			// fall back on falls through to the generic endgame.
			name: "third-suit-forcing-denied",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.thirdSuitDenied && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				if p.hand.Stopper(ctx.pm.thirdSuitSuit) || !ctx.hasFit || ctx.fit.IsMajor() {
					return Call{}, meaning{}, false
				}
				c := bidSuit(5, ctx.fit)
				if !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				return c, m(-1, -1,
					"personne n'a l'arrêt à "+suitNameFR[ctx.pm.thirdSuitSuit]+" : manche dans le fit mineur",
					"neither hand guards "+suitNameEN[ctx.pm.thirdSuitSuit]+": game in the minor fit").withLen(ctx.fit, p.hand.Len(ctx.fit)), true
			},
		},
		{
			// Answer partner's "relais contrôle": the immediately higher step
			// with the control asked for, otherwise the cheapest control above
			// it -- skipping the relay step denies what it asked for.
			name: "control-relay-answer",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.ctrlRelay && ctx.hasFit && ctx.hasBid &&
					ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.controlRelayAnswer(ctx.p, ctx.fit, ctx.pm.ctrlRelaySuit)
				if c.Kind == KindPass {
					return Call{}, meaning{}, false
				}
				return c, mn, true
			},
		},
		{
			// Continue a control-bid slam exploration started by partner
			// (docs/addon_4.md).
			name: "continue-control-bid",
			fr:   "exploration du chelem par les contrôles",
			en:   "slam exploration through control bids",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.controlBid && ctx.hasFit && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.continueControlBid(ctx.p, ctx.fit, ctx.tr)
				return c, mn, true
			},
		},
		{
			// Rectify partner's game-proposing transfer after Stayman confirmed
			// both majors (docs/bidings.md, "Les transferts").
			name: "rectify-stayman-both-majors",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.staymanBothMajorsRelay && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn := e.staymanBothMajorsAnswer(ctx.p, ctx.pm.texas)
				return c, mn, true
			},
		},
		{
			// Answer the misère dorée's reversed 2SA (docs/addon_11.md): our
			// Stayman answer ranked above partner's five-card major, so 2SA
			// stood in for the direct major-suit bid. With extra values, a
			// forcing 3-of-the-major shows three-card support; a maximum
			// without the fit settles for 3NT instead; a bare minimum just
			// stays at 2SA.
			name: "misere-doree-ask",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.misereDoreeAsk && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				M := ctx.pm.misereDoreeSuit
				if p.hand.H() >= p.shownMin+2 {
					if p.hand.Len(M) >= 3 {
						c := bid(3, M.Strain())
						if e.legal(p.seat, c) {
							mn := m(-1, -1, "maximum, trois cartes à "+suitNameFR[M], "maximum, three-card "+suitNameEN[M]).asForcing()
							mn.misereDoreeFit = true
							return c, mn, true
						}
					}
					c := bid(3, SNoTrump)
					if e.legal(p.seat, c) {
						return c, m(-1, -1, "maximum, pas trois cartes à "+suitNameFR[M], "maximum, no three-card "+suitNameEN[M]), true
					}
				}
				return passCall, m(-1, -1, "minimum, arrêt à Sans-Atout", "minimum, staying at notrump"), true
			},
		},
		{
			// The misère dorée's fit was confirmed (forcing three-card
			// support): the nine-card fit plus the known singleton settle the
			// strain, conclude game directly (docs/addon_11.md).
			name: "misere-doree-fit-confirmed",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.misereDoreeFit && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c := bid(4, ctx.last.Strain)
				if e.legal(ctx.p.seat, c) {
					return c, m(-1, -1, "conclusion à la manche, fit connu", "bidding game, known fit"), true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Slam invitation from partner (quantitative 4SA): the answer
			// follows the asker's promised range, at the intermediate step
			// (5-level) when neither extreme is certain (docs/addon_4.md, "4SA
			// quantitatif").
			name: "answer-slam-invite",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.slamInvite && ctx.partnerJustActed && ctx.ours
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner, own := ctx.p, ctx.partner, ctx.own
				strain := SNoTrump
				if ctx.hasFit {
					strain = ctx.fit.Strain()
				}
				// The answer is arithmetic, not a position inside our own
				// announced band. Reading it as a position is what lets a
				// wide band lie: sitting in the middle of a 4-11 answer says
				// nothing about whether 33 [E-9] is there, and the pair ends
				// up in six on twenty-nine points. What decides is our own
				// count added to what the asker has promised.
				switch {
				case own+partner.shownMin >= 33:
					c := bid(6, strain)
					if e.legal(p.seat, c) {
						return c, m(own, own, "accepte la proposition de chelem, le compte y est", "accepts the slam try, the count is there"), true
					}
				case own+partner.shownMax >= 33:
					c := bid(5, strain)
					if e.legal(p.seat, c) {
						mn := m(own, own, "accepte partiellement, laisse la décision au partenaire", "partial acceptance, leaves the final decision to partner")
						mn.partialSlamAccept = true
						return c, mn, true
					}
				}
				return passCall, m(own, own, "refuse la proposition de chelem, le compte n'y est pas", "declines the slam try, the count falls short"), true
			},
		},
		{
			// No fit for the second suit partner named over our 3SA [S-13c].
			// The question he asked was the quantitative 4SA's own -- is the
			// slam there? -- in a form that also offered a trump, so without
			// the trump the answer is the same arithmetic in notrump: 33
			// facing his floor bids six, 33 facing only his top accepts
			// partially at 5SA when his band is narrow enough for the answer
			// to mean something, and anything less goes back to 4SA. Left to
			// the generic endgame, the 4SA came back every time, and the slam
			// the quantitative try used to find was lost to the natural bid
			// that replaced it.
			name: "answer-second-suit-no-fit",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.secondSuitTry && ctx.partnerJustActed && ctx.ours && !ctx.hasFit
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner := ctx.p, ctx.partner
				own := p.hand.H()
				switch {
				case own+partner.shownMin >= 33:
					c := bid(6, SNoTrump)
					if e.legal(p.seat, c) {
						return c, m(own, own, "pas de fit, le compte y est : petit chelem à Sans-Atout", "no fit, the count is there: small slam in notrump"), true
					}
				case own+partner.shownMax >= 33 && partner.shownMax-partner.shownMin <= 7:
					c := bid(5, SNoTrump)
					if e.legal(p.seat, c) {
						mn := m(own, own, "pas de fit, accepte partiellement, laisse la décision au partenaire", "no fit, partial acceptance, leaves the final decision to partner")
						mn.partialSlamAccept = true
						return c, mn, true
					}
				}
				c := bid(4, SNoTrump)
				if e.legal(p.seat, c) {
					return c, m(own, own, "pas de fit, retour à Sans-Atout, le compte n'atteint pas le chelem", "no fit, back to notrump, the count falls short of slam"), true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Final word after a partial quantitative acceptance (5SA /
			// 5-of-the-fit): the original asker decides from their own
			// promised range.
			name: "after-partial-slam-accept",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.partialSlamAccept && ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner, own, last := ctx.p, ctx.partner, ctx.own, ctx.last
				// The partial acceptance pinned partner's count exactly, so
				// the last word is a subtraction: the slam needs 33 combined
				// [E-9], whatever end of our own band we sit at.
				if own+partner.shownMin < 33 {
					return passCall, m(-1, -1, "le compte combiné n'atteint pas le chelem, arrêt", "the combined count falls short of the slam, staying"), true
				}
				if own <= p.shownMin {
					return passCall, m(-1, -1, "minimum de la proposition, arrêt", "minimum of the range shown, staying"), true
				}
				c := bid(6, last.Strain)
				if e.legal(p.seat, c) {
					return c, m(-1, -1, "pas minimum, conclusion au petit chelem", "not minimum, bids the small slam"), true
				}
				return passCall, noInfo(), true
			},
		},
		{
			// "Oui mais": facing the FORCING three-level major raise, 3SA shows
			// a hand worth exploring a slam with -- never minimum -- that
			// nonetheless carries a contra-indication. It comes before the
			// control handlers below, which are the plain "oui"; the plain
			// "non" is the game the value-based endgame bids on its own.
			name: "slam-doubt-notrump",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.hasFit && ctx.fit.IsMajor() && ctx.cMin >= 29 &&
					ctx.pm != nil && ctx.pm.forcing && !ctx.pm.invite &&
					ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat) &&
					ctx.last == bidSuit(3, ctx.fit) &&
					!e.sideHasCued(ctx.p) && !e.bw[ctx.side].asked &&
					e.fitExpressed(ctx.p, ctx.fit)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				return e.slamDoubtNT(ctx.p, ctx.fit)
			},
		},
		{
			// "L'atout avant les contrôles" [S-0]. The control machinery below
			// needs a fit both hands have named. A hand that knows the fit on
			// its own -- partner promised five, we hold four -- knows something
			// partner does not: he never heard the suit from us, so he cannot
			// tell a control from a natural bid, and his answer would be to a
			// question he has not been asked.
			//
			// The knowledge is still worth having; it is the dialogue it cannot
			// open, not the contract it cannot choose. So name the trump first,
			// forcing, under the same slam-in-view gate that arms the controls,
			// and let the exchange start one round later on a suit both
			// partners have shown.
			name: "express-fit-before-controls",
			fr:   "chelem en vue (29-32 HLD combinés) mais le partenaire n'a jamais entendu l'atout de ma part : le soutenir d'abord, forcing",
			en:   "slam in view (29-32 HLD combined) but partner never heard the trump from me: raise it first, forcing",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return e.slamProbeArmed(ctx) && !e.trumpNamedByBoth(ctx.p, ctx.fit)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				return e.expressFit(ctx)
			},
		},
		{
			// Control-bid slam try: fit established, promising but not certain
			// yet (docs/addon_4.md): show the cheapest control below asking
			// Blackwood.
			//
			// The exchange only opens with a slam actually in view -- "ne
			// démarrer les enchères de contrôle que si on envisage un chelem".
			// That means either the combined *maximum* reaches the 33-HLD slam
			// zone (a wide-range partner -- a takeout double, an overcall --
			// may still hold slam values his floor hides), or the combined
			// *minimum* is already within a king of it. A pair whose ceiling
			// is a dead 29, facing a fully limited raise, has no slam to
			// explore and the probe only muddies a plain game.
			//
			// With a minor fit, though, any control past the 3SA level
			// condemns the pair to five of the minor (eleven tricks) when the
			// same count already affords 3NT (nine): the try must clear a
			// higher bar -- the minimum within two points of the slam zone --
			// unless notrump is not a playable game anyway, in which case 5m
			// is the game and the probe costs nothing.
			name: "control-bid-slam-try",
			fr:   "fit nommé par les deux mains, chelem en vue (29-32 HLD combinés) : ouvrir les enchères de contrôle",
			en:   "fit named by both hands, slam in view (29-32 HLD combined): open the control bids",
			when: func(e *Engine, ctx *concludeCtx) bool {
				// The fit must have been *expressed*, not merely computed from
				// partner's promise plus this hand's own length [S-0]: a suit
				// partner has never heard from us cannot be the subject of a
				// control, and he would read the bid as a natural one or a game
				// try. The handler above says the trump first; this one may
				// then open the dialogue.
				return e.slamProbeArmed(ctx) && e.trumpNamedByBoth(ctx.p, ctx.fit)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				if c, mn, ok := e.initiateControls(ctx.p, ctx.fit, ctx.tr); ok {
					return c, mn, true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// King ask over partner's game conclusion [S-10]. After a 2D
			// opening the ace step has already located every ace, so once
			// opener sees the four of them he owns a tool the point count
			// cannot reach: 4SA asks for kings, the trump queen standing in
			// as the fifth key [S-11]. [RO-24] applies it when opener's own
			// default notrump rebid would already be a game partner can pass;
			// the same burial happens one round later, when the fit is found
			// under the game and partner -- forced to answer whatever his
			// count, so 0 as a floor -- signs off in four of it. The control
			// road above cannot reopen the auction there: its ceiling is the
			// trump game, which is exactly where the bidding already sits.
			// Asking is still only worth the five level it costs when a
			// single king from partner would carry the pair to the 33-honour
			// slam zone [E-9]; a disappointing answer stops in the small slam
			// or, keys missing, is never reached at all (afterKings).
			name: "strong-2d-king-ask-over-game",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.hasFit && ctx.hasBid && ctx.ours &&
					ctx.lastSeat == partnerOf(ctx.p.seat) && isGame(ctx.last) &&
					!e.bw[ctx.side].asked && e.isKingAsk(ctx.p) &&
					ctx.own+ctx.partner.shownMin+3 >= 33
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, fit := ctx.p, ctx.fit
				if !e.trumpAgreed(p, fit) {
					return Call{}, meaning{}, false
				}
				c := bid(4, SNoTrump)
				if !c.higherThan(ctx.last) || !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				st, mn := e.blackwoodAsk(p, fit, ctx.own)
				e.bw[ctx.side] = st
				return c, mn, true
			},
		},
		{
			// Petit chelem sur les As déjà localisés [S-7b]. After a 2D opening
			// the ace step told opener exactly how many aces partner holds, so
			// the keycard count is already on the table before any ask: his own
			// keys plus partner's aces. Once the fit is found above 4SA -- a
			// minor raised to game, 2K - 2C - 2P - 3SA - 4T - 5T -- neither
			// Blackwood nor the king ask has room left, and the control
			// exchange stops at the trump game it already sits on, so the slam
			// the count reaches is simply passed out. Four keys known and the
			// slam count reached is [S-7]'s own conclusion: bid it.
			name: "strong-2d-slam-on-located-aces",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.hasFit && ctx.hasBid && ctx.cMinSlam >= 33 &&
					ctx.partner.acesExact && !ctx.p.answeredAces && !e.bw[ctx.side].asked &&
					!bid(4, SNoTrump).higherThan(ctx.last) &&
					ctx.p.hand.Keycards(ctx.fit)+ctx.partner.acesShown >= 4
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, fit := ctx.p, ctx.fit
				if !e.trumpAgreed(p, fit) {
					return Call{}, meaning{}, false
				}
				c := bidSuit(6, fit)
				if !c.higherThan(ctx.last) || !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				return c, m(ctx.own, 40, "petit chelem : As du partenaire connus, une seule clef manque au plus", "small slam: partner's aces known, one keycard missing at most"), true
			},
		},
		{
			// Help-suit game try from partner: unlike a generic invitation,
			// accepting requires real help in the named suit (an honor or
			// shortness), not just being in the upper half of the shown point
			// range (docs/bidings.md, "EN FACE D'UN SOUTIEN MAJEUR SIMPLE").
			name: "help-suit-try-answer",
			fr:   "le partenaire fait un essai de manche dans une couleur d'aide : accepter avec de l'aide dans cette couleur",
			en:   "partner makes a help-suit game try: accept with help in that suit",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.helpSuitTry && ctx.partnerJustActed && ctx.ours
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, fit, hasFit, last := ctx.p, ctx.fit, ctx.hasFit, ctx.last
				tr := ctx.tr
				if hasFit && tr.check(e.hasHelp(p, ctx.pm.helpSuit),
					"aide dans la couleur d'essai : As ou Roi, Dame troisième, ou deux cartes au plus",
					"help in the try suit: ace or king, queen third, or two cards at most", cards(p.hand, ctx.pm.helpSuit)) {
					gc := bidSuit(4, fit)
					gcFR, gcEN := callSym(gc)
					if tr.check(e.legal(p.seat, gc), "→ la manche : "+gcFR, "→ game: "+gcEN, "") {
						return gc, m(p.shownMin, p.shownMax, "accepte l'essai, aide dans la couleur demandée", "accepts the try, help in the asked suit"), true
					}
				}
				if hasFit && last.Strain != fit.Strain() {
					c := bid(last.Level, fit.Strain())
					if !c.higherThan(last) {
						c = bid(last.Level+1, fit.Strain())
					}
					cFR, cEN := callSym(c)
					if tr.check(e.legal(p.seat, c), "refuser : revenir dans le fit au plus bas → "+cFR,
						"decline: back to the fit at the lowest level → "+cEN, "") {
						if c.Level <= 3 {
							return c, m(-1, -1, "refuse l'essai, pas d'aide dans la couleur", "declines the try, no help in the suit"), true
						}
						tr.note("l'essai ne laissait pas la place de refuser sous la manche", "the try left no room to decline below game")
						// The try was named above three of the trump, so there
						// is no room left to decline: the return is forced.
						// Passing would leave the pair playing the try itself,
						// in a suit nobody has agreed -- always worse than the
						// game in the fit, however unwanted.
						return c, m(-1, -1, "pas d'aide, mais l'essai ne laissait pas la place de refuser : retour forcé dans notre couleur", "no help, but the try left no room to decline: forced back to our suit"), true
					}
				}
				tr.note("refuser → Passe", "decline → Pass")
				return passCall, m(-1, -1, "refuse l'essai, pas d'aide dans la couleur", "declines the try, no help in the suit"), true
			},
		},
		{
			// Partner answered our réveil with 2SA: 13-15 H and a stopper,
			// an invitation [R-1]. The réveilleur announced 8-13 HL, so the
			// answer is where he sits inside that bracket rather than an
			// absolute count: from 11 the pair is facing an average 14 and
			// the game is there -- and the five-card suit the réveil promised
			// is the source of tricks 3SA plays for. Below that he passes.
			name: "answer-reopening-2nt",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.reopenNTInvite && ctx.partnerJustActed && ctx.ours &&
					ctx.p.lastM != nil && ctx.p.lastM.reopen
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				if p.hand.HL() < 11 {
					return passCall, m(-1, -1,
						"minimum du réveil, la manche n'y est pas",
						"minimum for the réveil, the game is not there"), true
				}
				c := bid(3, SNoTrump)
				if !c.higherThan(ctx.last) || !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				return c, m(11, 13,
					"3SA : le haut du réveil, 11HL et plus",
					"3NT: the top of the réveil, 11+ HL"), true
			},
		},
		{
			// Game invitation from partner.
			name: "game-invitation-answer",
			fr:   "le partenaire propose la manche : accepter ou refuser",
			en:   "partner invites game: accept or decline",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.pm != nil && ctx.pm.invite && ctx.partnerJustActed && ctx.ours
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner, fit, hasFit, last, ntOK := ctx.p, ctx.partner, ctx.fit, ctx.hasFit, ctx.last, ctx.ntOK
				// A non-forcing three-level raise of a major is a narrow,
				// trustworthy zone, and the hand that hears it is the only one
				// who can decide to look further: "La décision d'explorer le
				// chelem est uniquement le fait du joueur qui vient de recevoir
				// l'information de ce soutien limite." Once the combined
				// minimum reaches the exploration window, accepting the game
				// here would throw that moment away — hand over to the control
				// machinery further down the table. It still bids the game on
				// its own if nothing there claims the call.
				// The raise must be a genuine limit raise -- "ces soutiens sont
				// toujours limités et donc très précis" -- not the generic
				// invitation the value-based endgame produces, whose range is
				// inferred and often several points wide.
				narrowRaise := ctx.pm != nil && ctx.pm.minPts >= 0 && ctx.pm.maxPts >= 0 &&
					ctx.pm.maxPts-ctx.pm.minPts <= 4
				// Hand over to the control machinery only when a slam is in view
				// ("ne démarrer les enchères de contrôle que si on envisage un
				// chelem"): the combined maximum reaches the slam zone, or the
				// minimum is within a king of it. Otherwise no cue partner can
				// answer leads anywhere but the game this handler bids anyway.
				if hasFit && fit.IsMajor() && ctx.cMin >= 29 && (ctx.cMax >= 33 || ctx.cMin >= 31) &&
					!e.bw[ctx.side].asked &&
					last == bidSuit(3, fit) && narrowRaise && e.fitExpressed(p, fit) {
					return Call{}, meaning{}, false
				}
				// Accept on genuine extra values. A doubleton adds little
				// playing strength opposite a limited invitation, so only real
				// shortness (a singleton or void) counts toward straining to
				// game; a balanced maximum must bring the honour strength
				// itself. This keeps flat hands (e.g. 4-2-2-5, whose two
				// doubletons inflate HLD) from overbidding a minimum into a
				// failing game.
				acceptVal := p.hand.HL()
				if hasFit {
					for s := Clubs; s <= Spades; s++ {
						if s == fit {
							continue
						}
						switch p.hand.Len(s) {
						case 0:
							acceptVal += 3
						case 1:
							acceptVal += 2
						}
					}
				} else {
					// No suit fit: the invitation is toward notrump, where shape
					// does not add playing tricks the way it does in a suit
					// contract -- a long side suit may not run, and a void is a
					// liability rather than an asset. Value by high cards alone
					// so a distributional minimum (e.g. 4-5-0-4, 13 HCP but 14
					// HL) cannot masquerade as a maximum and strain to a failing
					// 3NT.
					//
					// A six-card suit headed by two of the three top honours is
					// the exception when partner invites in notrump: it is the
					// source of tricks 3SA plays for, not a suit that may not
					// run. Its length points stay, as the HL bracket the hand
					// announced counted them (1K - 1P - 2K - 2SA with KT A94
					// AK7643 87: 16 HL, the top of 13-17, accepts). A suit
					// invitation is left alone -- accepting it in 3SA opposite
					// a void in partner's suit is the stranded game the retreat
					// below avoids.
					acceptVal = p.hand.H()
					for s := Clubs; s <= Spades && last.Strain == SNoTrump; s++ {
						if n := p.hand.Len(s); n >= 6 && topHonours(p.hand, s) >= 2 {
							acceptVal += n - 4
						}
					}
				}
				// The invitation asks where the hand sits *inside the range it
				// has already shown*: a 12-14 1SA rebid declines on 12 and
				// accepts on 13-14 -- partner has done the addition, minimum and
				// maximum are relative to the announced bracket, not to an
				// absolute count. Only a hand still unlimited (a wide-open
				// 1-of-a-suit opening) falls back on the "two points above the
				// promised minimum" heuristic.
				threshold := p.shownMin + 2
				if mid := p.shownMin + (p.shownMax-p.shownMin+1)/2; mid < threshold {
					threshold = mid
				}
				accept := acceptVal >= threshold
				tr := ctx.tr
				acceptV := pts(acceptVal, "H")
				if bonus := acceptVal - p.hand.HL(); hasFit {
					acceptV = fmt.Sprintf("%d HL + %d", p.hand.HL(), bonus)
				} else if bonus := acceptVal - p.hand.H(); bonus > 0 {
					acceptV = fmt.Sprintf("%d H + %d", p.hand.H(), bonus)
				}
				acceptFR := fmt.Sprintf("haut de la fourchette annoncée (%d-%d) : %d et plus → accepter",
					p.shownMin, p.shownMax, threshold)
				acceptEN := fmt.Sprintf("top of the announced range (%d-%d): %d or more → accept",
					p.shownMin, p.shownMax, threshold)
				if len(e.opponentSuits(p)) > 0 {
					// In a contested auction the invitation is a competitive
					// raise with a wide but honest floor, so accept only when
					// the partnership's minimum combined count actually reaches
					// game. The "+2 over my own minimum" heuristic overbids
					// here: an unclarified wide opening (1M, 12-23) has a low
					// shownMin, so a bare maximum strains to a failing game
					// opposite a raise that may be minimum.
					v := acceptVal
					if hasFit && p.hand.Len(fit) < partner.shownLens[fit] {
						// The threshold is an HLD figure [E-9] -- 27 in a major
						// fit -- while acceptVal is HL plus shortness, which is
						// HLD minus the doubleton. Against an absolute count
						// the gap is a unit error worth exactly one point, and
						// games are decided by one point.
						//
						// It is only worth closing in the hand that will be
						// dummy. Distribution pays for ruffs, so the doubleton
						// is worth its point to the *supporting* hand, whose
						// trumps are spare; in the hand holding the longer
						// trumps the same doubleton buys nothing -- those
						// trumps are spent drawing the opponents' -- and
						// counting it is how a flat 3-5-3-2 opening talks
						// itself into a failing game.
						v = e.hldAgainstTheirBidding(p, fit)
					}
					accept = partner.shownMin+v >= gameThresholdFor(fit, hasFit, ntOK)
					g := gameThresholdFor(fit, hasFit, ntOK)
					acceptFR = fmt.Sprintf("enchères disputées : minimum combiné au seuil de la manche (%d) → accepter", g)
					acceptEN = fmt.Sprintf("contested auction: combined minimum at the game threshold (%d) → accept", g)
					acceptV = fmt.Sprintf("%d + %d = %d", v, partner.shownMin, partner.shownMin+v)
				}
				// The answer to a takeout double with no jump available (2H
				// over 1S) covers the weak and the middle zone at once, and
				// the doubler's try -- which he only makes on a real 16 HLD
				// with nothing wasted in their suit -- asks precisely which
				// half. The middle zone accepts: the combined count computed
				// here understates the try, whose floor is derived from that
				// same wide zone.
				if p.wideDouble && hasFit && fit == p.wideDoubleFit && p.hand.H() >= 8 {
					accept = true
					acceptFR = "réponse au contre qui couvrait deux zones, 8 H et plus, fit → accepter"
					acceptEN = "answer to the double covering two zones, 8+ H, fit → accept"
					acceptV = pts(p.hand.H(), "H")
				}
				if tr.check(accept, acceptFR, acceptEN, acceptV) {
					gc, ok := gameCall(fit, hasFit, 99, 99, ntOK) // force game choice
					if hasFit && fit.IsMajor() {
						gc, ok = bidSuit(4, fit), true
					}
					gcFR, gcEN := callSym(gc)
					if tr.check(ok && e.legal(p.seat, gc), "→ la manche : "+gcFR, "→ game: "+gcEN, "") {
						// In a contested auction the acceptance rests on the
						// combined count, not on clearing the threshold: never
						// promise more than the value that actually accepted.
						lo := threshold
						if acceptVal < lo {
							lo = acceptVal
						}
						return gc, m(lo, p.shownMax, "accepte la proposition de manche, maximum", "accepts the game try, maximum"), true
					}
				}
				// Decline: correct to the fit at the lowest level if needed,
				// else pass. A genuine decline caps the hand below the
				// acceptance threshold, so partner reads "minimum" against the
				// announced bracket.
				declineMax := -1
				if !accept {
					declineMax = threshold - 1
				}
				if hasFit && last.Strain != fit.Strain() {
					c := bid(last.Level, fit.Strain())
					if !c.higherThan(last) {
						c = bid(last.Level+1, fit.Strain())
					}
					if tr.check(c.Level <= 3 && e.legal(p.seat, c),
						"refuser : revenir dans la couleur du fit, au plus bas", "decline: back to the fit, at the lowest level", fitSym(fit)) {
						return c, m(-1, declineMax, "refuse, retour dans la couleur fittée", "declines, signs off in the fit"), true
					}
				}
				// Passing a suit invitation with a void in that suit strands
				// the side in a trump suit the hand cannot support at all (a
				// 6-0 "fit"): with no fit elsewhere, retreat to the long suit
				// already shown instead.
				if !hasFit && last.Strain != SNoTrump && p.hand.Len(Suit(last.Strain)) == 0 {
					long, longLen := Clubs, 0
					for s := Clubs; s <= Spades; s++ {
						if s != Suit(last.Strain) && p.shownLens[s] >= 5 && p.hand.Len(s) > longLen {
							long, longLen = s, p.hand.Len(s)
						}
					}
					if longLen >= 6 {
						c := bid(last.Level, long.Strain())
						if !c.higherThan(last) {
							c = bid(last.Level+1, long.Strain())
						}
						if tr.check(c.Level <= 3 && e.legal(p.seat, c),
							"chicane dans sa couleur : refuser en revenant à sa propre sixième",
							"void in partner's suit: decline by going back to the own six-card suit", cards(p.hand, long)) {
							return c, m(-1, declineMax, "refuse la proposition, chicane dans la couleur du partenaire, retour à sa couleur", "declines the invitation, void in partner's suit, retreats to own suit").withLen(long, p.hand.Len(long)), true
						}
					}
				}
				tr.note("refuser → Passe", "decline → Pass")
				return passCall, m(-1, declineMax, "refuse la proposition, minimum", "declines the invitation, minimum"), true
			},
		},
		{
			// Slam exploration with a fit. Blackwood counts keycards, not
			// second-round control: launching it with two fast losers in an
			// uncontrolled side suit risks a slam off the first two tricks
			// there. When the point count reaches slam but a side suit is
			// unguarded and partner is not known to cover it, probe with
			// control bids first -- partner may show the missing control and
			// reopen the road to Blackwood; only if the probe finds nothing
			// does the auction stop in game (docs/addon_4.md, "mécanisme des
			// contrôles").
			name: "slam-explore-with-fit",
			fr:   "fit et zone de chelem : 33 HLD combinés (sans l'appoint des courtes face aux longues du partenaire), ou 32 et les 5 cartes clefs",
			en:   "fit and slam zone: 33 HLD combined (without shortness facing partner's length), or 32 and all 5 keycards",
			when: func(e *Engine, ctx *concludeCtx) bool {
				// p.bids > 0: see the identical guard on control-bid-slam-try --
				// the fit must be agreed through bidding, not just computed from
				// the opening's promise plus this hand's own length.
				//
				// The count is not the only road in. When a big hand can see
				// that no keycard is missing -- its own aces and trump king,
				// plus whatever partner's cue-bids have revealed, already make
				// five -- the slam no longer turns on the total: it turns on
				// the trump queen and the kings, and the ask cannot come back
				// short. That is the case [S-4] describes, and the count is
				// exactly what fails to see it: a king shown by a cue-bid
				// raises partner's floor by three, so a pair holding every ace
				// and three kings still counts 32 and stops in game. Partner
				// signing off there says he has no more controls to show, not
				// that there is no slam.
				//
				// The shortcut only makes up for that one missing point, though.
				// Five keycards say every ace is home, not that twelve tricks
				// are: a partner who has shown nothing past a forced sign-off
				// (2K - 2P - Passe - 3C - 4C, cMin 30) brings no king, no trump
				// queen, no ruff the slam could lean on. Holding the five
				// himself, the asker also learns nothing from the answer --
				// "0 or 3" can only be 0 -- so the ask would only buy the five
				// level on the way to a slam the count never came near.
				noKeycardMissing := ctx.hasFit && ctx.own >= 20 && ctx.cMinSlam >= 32 &&
					ctx.p.hand.Keycards(ctx.fit)+ctx.partner.keycardsShown >= 5
				return ctx.hasFit && (ctx.cMinSlam >= 33 || noKeycardMissing) &&
					ctx.p.bids > 0 && ctx.partner.bids > 0 &&
					!e.bw[ctx.side].asked && !ctx.p.answeredAces
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, fit, own, cMin, last, pm := ctx.p, ctx.fit, ctx.own, ctx.cMin, ctx.last, ctx.pm
				tr := ctx.tr
				tr.check(true, "zone de chelem", "slam zone", pts(ctx.cMinSlam, "HLD"))
				// Keycard Blackwood counts the trump king and the trump queen:
				// it can only be asked once the trump suit is one partner can
				// name too. A fit merely computed from the opening's promise
				// plus this hand's own length is not that -- answering "two
				// keycards and the trump queen" would be answering about a suit
				// nobody agreed. Show the suit first; the ask comes later.
				if !tr.check(e.trumpAgreed(p, fit), "atout convenu par la séquence ("+fitSym(fit)+")",
					"trump agreed by the auction ("+fitSym(fit)+")", "") {
					// Name the suit instead: a jump when there is room, so the
					// bid carries the values that reached the slam zone rather
					// than reading as a mere preference.
					if p.shownLens[fit] == 0 && p.hand.Len(fit) >= 5 {
						c := e.cheapestCall(fit.Strain())
						if up := bid(c.Level+1, fit.Strain()); up.Level <= 4 && e.legal(p.seat, up) {
							c = up
						}
						cFR, cEN := callSym(c)
						if tr.check(c.Level <= 4 && e.legal(p.seat, c),
							"couleur de 5 cartes pas encore nommée → la nommer (avec saut si possible) : "+cFR,
							"five-card suit not yet bid → bid it (jumping if possible): "+cEN, cards(p.hand, fit)) {
							mn := m(own, 40, "couleur longue, propose l'atout avant toute demande", "long suit, proposes the trump before any ask").withLen(fit, p.hand.Len(fit)).asForcing()
							return c, mn, true
						}
					}
					// The trump lives in partner's suit and this hand is the
					// short one: it has no five-card suit to name, but the
					// raise puts the very same trump on the table [S-4b]. Kept
					// below the game level so partner still has room to answer,
					// and forcing: at the slam count this is a slam try, not
					// the conclusion the same call would be one rung higher.
					if partner := ctx.partner; partner.shownLens[fit] > 0 && p.hand.Len(fit) >= 3 {
						c := e.cheapestCall(fit.Strain())
						cFR, cEN := callSym(c)
						if tr.check(gameOfTrump(fit).higherThan(c) && e.legal(p.seat, c),
							"soutenir la couleur du partenaire sous la manche, forcing : "+cFR,
							"raise partner's suit below game, forcing: "+cEN, cards(p.hand, fit)) {
							mn := m(own, 40, "soutien forcing, propose l'atout avant toute demande", "forcing raise, proposes the trump before any ask").withLen(fit, p.hand.Len(fit)).asForcing()
							return c, mn, true
						}
					}
					return Call{}, meaning{}, false
				}
				if !tr.check(e.uncontrolledSideSuit(p, fit),
					"une couleur annexe sans contrôle connu (deux perdantes rapides possibles)",
					"a side suit with no known control (two fast losers possible)", "") {
					c := bid(4, SNoTrump)
					if tr.check(e.legal(p.seat, c), "toutes les couleurs couvertes → Blackwood 4SA",
						"every suit covered → 4NT Blackwood", "") {
						st, mn := e.blackwoodAsk(p, fit, own)
						e.bw[ctx.side] = st
						return c, mn, true
					}
					// Every side suit covered but 4SA already gone (the auction
					// is past it): keep describing with control bids instead of
					// giving up -- the exchange finds the slam or signs off on
					// its own.
					if c, mn, ok := e.controlBid(p, fit); tr.check(ok,
						"4SA déjà dépassé → continuer par les contrôles", "4NT already passed → carry on with control bids", callOrEmpty(c, ok)) {
						return c, mn, true
					}
				} else {
					// A side suit is unguarded, so Blackwood is out: probe with
					// a control bid instead. Below the trump game the probe is
					// free -- partner can always sign off in four of the fit --
					// so ask the question there without further conditions.
					// Above game it buys the five level if the missing control
					// never shows, so it additionally needs a real cushion
					// beyond the bare slam zone and a partner whose last bid
					// set a narrow codified zone (e.g. a 20-23 jump raise) the
					// count can trust -- not a heuristic conclusion whose cap
					// is an illusion inherited from an earlier bid.
					belowGame := gameOfTrump(fit).higherThan(last)
					cushion := cMin >= 36 && pm != nil && pm.minPts >= 0 && pm.maxPts >= 0 && pm.maxPts-pm.minPts <= 4
					// The trump is agreed [S-4b] -- the branch above returned
					// otherwise -- so the cue-bid names a suit partner can read
					// even when he alone has shown its length: a raise is not
					// the only way to agree a trump, and demanding one here
					// left the slam hand with nothing but the game to bid.
					if tr.check(belowGame || cushion,
						"sous la manche (ou 36 HLD et un partenaire limité) → sonder par les contrôles plutôt que Blackwood",
						"below game (or 36 HLD and a limited partner) → probe with control bids rather than Blackwood", "") {
						// [S-0] again, at the other door into the control
						// machinery. trumpAgreed above was enough to ask
						// Blackwood -- the answers name a suit our side has
						// bid -- but a control is a question, and partner
						// cannot answer one about a trump he has never heard
						// from us. Say the fit first; the exchange opens next
						// round on a suit both hands have named.
						// When there is no room to raise under the game, the
						// raise would be the conclusion and the cue is all
						// that is left: the trump is then agreed in substance
						// -- it is the only suit our side has on the table --
						// so the exchange goes ahead rather than dying here.
						if !e.trumpNamedByBoth(p, fit) {
							if c, mn, ok := e.expressFit(ctx); ok {
								return c, mn, true
							}
						}
						if c, mn, ok := e.initiateControls(p, fit, ctx.tr); ok {
							return c, mn, true
						}
					}
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Slam exploration with no fit: ask keycards first rather than jump
			// blind to 6NT. With no trump suit agreed, this can only count
			// plain aces, but confirming none are missing (or that all four are
			// held, worth a grand slam try) beats trusting the raw point count
			// alone.
			//
			// A hand that answered the ace steps is not the captain and does
			// not ask: its aces are already on the table and the strong hand
			// runs the auction. But once that hand has signed off in game, the
			// description is over, and the ace ladder has told it almost
			// nothing about points -- a floor of 0 for the 2C step, 4 for a
			// single ace, 8 for the rest, and no ceiling at all. Counting from
			// that floor the captain rests in 3SA on 35 real points, and the
			// only hand that can see the 33 is the one holding the points the
			// floor left out. Over a sign-off, and only there, it corrects
			// [S-10b].
			name: "slam-explore-no-fit-notrump",
			fr:   "sans fit, 33 H combinés : zone de chelem à Sans-Atout",
			en:   "no fit, 33 H combined: notrump slam zone",
			when: func(e *Engine, ctx *concludeCtx) bool {
				if ctx.p.answeredAces {
					return !ctx.hasFit && ctx.cMin >= 33 &&
						ctx.lastSeat == partnerOf(ctx.p.seat) && isGame(ctx.last)
				}
				return !ctx.hasFit && ctx.cMin >= 33
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, own, last, ours := ctx.p, ctx.own, ctx.last, ctx.ours
				tr := ctx.tr
				tr.check(true, "zone de chelem", "slam zone", fmt.Sprintf("%d H + %d = %d", own, ctx.partner.shownMin, ctx.cMin))
				if !e.bw[ctx.side].asked && !p.answeredAces {
					c := bid(4, SNoTrump)
					if tr.check(e.legal(p.seat, c) && (!ours || last.steps() < c.steps()),
						"4SA encore disponible → Blackwood (demande des As)", "4NT still available → Blackwood (ace ask)", "") {
						st := bwState{asked: true, asker: p.seat, noTrump: true}
						var mn meaning
						if e.isKingAsk(p) {
							// After a 2D opening every ace is already located,
							// so this 4NT asks for kings instead (no trump, so
							// a plain king count).
							st.kingAsk = true
							mn = m(own-1, -1, "Blackwood 4SA, appel aux Rois (As déjà localisés)", "4NT king Blackwood (aces already located)")
						} else {
							mn = m(own-1, -1, "Blackwood 4SA sans-atout, demande des As", "4NT Blackwood, notrump ace ask")
						}
						mn.blackwood = true
						mn.forcing = true
						e.bw[ctx.side] = st
						return c, mn, true
					}
				}
				c := bid(6, SNoTrump)
				if tr.check(e.legal(p.seat, c) && (!ours || last.steps() < c.steps()),
					"→ petit chelem à Sans-Atout : 6SA", "→ small slam in notrump: 6NT", "") {
					return c, m(own, -1, "conclusion au petit chelem à Sans-Atout (33HL+)", "small slam in notrump on combined values (33+)"), true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Deuxième couleur par-dessus le 3SA du partenaire [S-13c]. A hand
			// that has shown one five-card suit and still holds a second one,
			// unnamed, has not finished describing itself when partner signs
			// off in 3SA -- he chose notrump without knowing the second suit
			// exists. Where the count leaves a slam in reach, the 4SA
			// quantitative try [S-13] would bury the double fit exactly as the
			// flat 3SA does: twelve notrump tricks on a 5-5 with a singleton
			// beside it, when four cards opposite the second suit make it a
			// trump suit worth the distribution points notrump never counts.
			// Naming the suit at the four level is natural and forcing, it
			// costs no more than the quantitative 4SA, and it names the strain
			// from the long hand, which then declares.
			name: "second-suit-over-partner-3nt",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return !ctx.hasFit && ctx.cMin >= 28 && ctx.cMax >= 33 &&
					ctx.lastSeat == partnerOf(ctx.p.seat) &&
					ctx.last.Kind == KindBid && ctx.last.Strain == SNoTrump && isGame(ctx.last)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner := ctx.p, ctx.partner
				first, hasFirst := Clubs, false
				for s := Spades; s >= Clubs; s-- {
					if p.shownLens[s] >= 5 {
						first, hasFirst = s, true
						break
					}
				}
				if !hasFirst {
					return Call{}, meaning{}, false
				}
				opp := e.opponentSuits(p)
				for s := Spades; s >= Clubs; s-- {
					if s == first || p.hand.Len(s) < 5 || p.shownLens[s] > 0 || partner.shownLens[s] > 0 ||
						slices.Contains(opp, s) {
						continue
					}
					c := bidSuit(4, s)
					if !c.higherThan(ctx.last) || !e.legal(p.seat, c) {
						continue
					}
					mn := m(-1, -1, "bicolore : deuxième couleur cinquième, essai de chelem", "two-suiter: second five-card suit, slam try").
						withLen(s, 5).withLen(first, p.shownLens[first]).asForcing()
					mn.secondSuitTry = true
					return c, mn, true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Quantitative 4SA: opener's plain notrump rebid only promises the
			// bottom of its range, so cMin alone misses the slam zone reached
			// opposite its top (docs/addon_4.md, "4SA quantitatif").
			//
			// The floor matters as much as the ceiling. The question is bought
			// with the fourth level of notrump -- a game that must now take ten
			// tricks instead of nine -- so it is only worth asking when partner
			// needs a few points above his floor, not the whole width of his
			// band. Over 40 000 deals the break is clean: from 28 combined the
			// try produced 129 invitations for 55 slams, below it 98 for 6,
			// with 85 of those coming straight back to rest in 4SA.
			//
			// The strict balanced shape was the wrong guard [S-13b]: it was
			// protecting a notrump contract the side is going to play in any
			// case -- no fit anywhere -- when what twelve tricks at notrump
			// really cannot afford is a void. A lone singleton beside three
			// real suits does not stop 6SA from being right, and a hand like
			// AQJ87 AK9 AQ63 4 opposite a 12-14 notrump rebid was left to bid
			// a flat 3SA on 32 combined points. What the shape must still
			// exclude is the singleton sitting beside a six-card suit, whose
			// slam belongs in the suit; and since the proposal names a
			// notrump slam, ntOK now has to hold as well.
			//
			// Two conditions that look equally arbitrary were measured and
			// kept. Partner's last call must be a notrump bid: without it the
			// rule fires over a live suit auction and buries it -- on 5 000
			// neutral deals, one slam lost outright and a grand cut to 6SA,
			// because a modest 3SA leaves partner room to bid on where 4SA
			// ends the auction. And the band no wider than seven keeps the
			// rule out of the strong 2K sequences: facing a partner left
			// uncapped by a forcing bid the invitation is not mine to make --
			// he knows my floor and his own count, I only know mine, and
			// jumping in front of him buys the small slam where his own king
			// ask found the grand.
			name: "quantitative-4nt",
			when: func(e *Engine, ctx *concludeCtx) bool {
				p, partner := ctx.p, ctx.partner
				return !ctx.hasFit && ctx.ntOK && ctx.cMin >= 28 && ctx.cMin < 33 && ctx.cMax >= 33 &&
					partner.shownMax-partner.shownMin <= 7 && p.hand.notrumpSlamShape() &&
					ctx.pm != nil && ctx.lastSeat == partnerOf(p.seat) &&
					ctx.last.Kind == KindBid && ctx.last.Strain == SNoTrump
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				partner := ctx.partner
				c := bid(4, SNoTrump)
				if e.legal(ctx.p.seat, c) {
					mn := m(max(0, 33-partner.shownMax), 32-partner.shownMin, "4SA quantitatif, propose le petit chelem", "quantitative 4NT, small slam try")
					mn.slamInvite = true
					return c, mn, true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Troisième couleur forcing [§8.9]. Opener has repeated his
			// opening minor at the two level, which denies at once the fit
			// for our major, a second suit and any extra values; we hold the
			// five-card major and the values for game, and the only two
			// cards that decide the strain -- his third trump, or his
			// stopper in the suit above his minor -- are cards no count can
			// guess. Naming that suit asks for them.
			name: "third-suit-forcing",
			fr:   "l'ouvreur a répété sa mineure : chercher le fit 5-3 ou l'arrêt",
			en:   "opener repeated the minor: look for the 5-3 fit or the stopper",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.partnerJustActed && sideOf(ctx.p.seat) == sideOf(e.opener)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn, ok := e.thirdSuitAsk(ctx.p)
				ctx.tr.check(ok, "majeure cinquième exactement, 11 H, pas de fit majeur → troisième couleur forcing (forcing de manche)",
					"exactly five cards in my major, 11 H, no major fit → third suit forcing (game forcing)", callOrEmpty(c, ok))
				return c, mn, ok
			},
		},
		{
			// Quatrième couleur forcing [§8.8]. Our side has named three
			// suits, none of them a fit, and the hand is worth game if
			// partner has the right card. Bidding the fourth suit -- the one
			// nobody can want to play -- asks him which card that is,
			// instead of guessing at notrump without a stopper or at a game
			// in a fit that may not exist.
			name: "fourth-suit-forcing",
			fr:   "trois couleurs nommées sans fit : la quatrième peut servir de question",
			en:   "three suits bid without a fit: the fourth can be used as a question",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return !ctx.hasFit && ctx.partnerJustActed && sideOf(ctx.p.seat) == sideOf(e.opener)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				c, mn, ok := e.fourthSuitAsk(ctx.p)
				ctx.tr.check(ok, "10 H, pas de fit, majeure cinquième ou pas d'arrêt dans la 4e couleur → quatrième couleur forcing",
					"10 H, no fit, a five-card major or no stopper in the fourth suit → fourth suit forcing", callOrEmpty(c, ok))
				return c, mn, ok
			},
		},
		{
			// Responder's second major: rather than guess the final strain, a
			// responder holding an unbid second major alongside the suit
			// already shown names it (docs/bidings.md, "le changement de
			// couleur par le répondant") instead of settling for a vaguer
			// quantitative call (a raise or a notrump proposal) -- naming the
			// suit costs nothing extra when it fits at the same level the
			// auction is already at, and it may uncover a far superior fit a
			// generic call would hide (e.g. a 5-5 major two-suiter opposite a
			// 4-card fit in the second suit: notrump plays for eight tricks
			// where the major makes eleven). Only in an uncontested auction
			// with no fit yet agreed, and only for a second major (a minor is
			// not worth exploring once a good major is in hand).
			//
			// A suit reachable at the same level as the auction's last bid is
			// shown freely, since it is no more committal than the generic
			// call it replaces. A suit that can only be reached by jumping a
			// level (a genuine reverse) still needs the combined count to be
			// in the game zone (cMin >= 25) to justify forcing the auction up
			// -- the "bicolore cher" case.
			name: "responder-second-major",
			when: func(e *Engine, ctx *concludeCtx) bool {
				p := ctx.p
				return !ctx.hasFit && p.seat != e.opener && sideOf(p.seat) == sideOf(e.opener) &&
					len(e.opponentSuits(p)) == 0 && !p.hand.IsRegular() && !p.hand.IsSemiRegular()
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner, last, cMin := ctx.p, ctx.partner, ctx.last, ctx.cMin
				// Once our side has named three suits, the fourth one is the
				// convention's [§8.8], not a natural second suit: a hand too
				// weak to ask must not borrow its call and be answered as if
				// it had.
				_, fourth, isFourthSuitAuction := e.fourthSuitCall(p)
				firstSuit, hasFirst := Clubs, false
				for s := Spades; s >= Clubs; s-- {
					if p.shownLens[s] >= 4 {
						firstSuit, hasFirst = s, true
						break
					}
				}
				if !hasFirst {
					return Call{}, meaning{}, false
				}
				for _, s := range []Suit{Spades, Hearts} {
					// A genuine two-suiter has both suits within a card of
					// each other (5-5, 5-4, 6-5...): the strain genuinely
					// matters, so naming the second suit is worth it. A much
					// longer first suit (e.g. 6-4) is already a good trump
					// source on its own -- rebidding/inviting in it describes
					// the hand better than introducing a shorter side suit.
					if s == firstSuit || p.hand.Len(s) < 4 || p.hand.Len(s)+1 < p.hand.Len(firstSuit) ||
						p.shownLens[s] > 0 || partner.shownLens[s] > 0 ||
						(isFourthSuitAuction && s == fourth) {
						continue
					}
					c := e.cheapestCall(s.Strain())
					if c.Level > 3 || isGame(c) || !c.higherThan(last) || !e.legal(p.seat, c) {
						continue
					}
					lo := p.shownMin
					if c.Level > last.Level {
						// A genuine reverse: forcing the auction up a level
						// needs real values behind it, not just shape. The
						// hand's own guaranteed floor is what remains once
						// partner's minimum is subtracted from the game
						// threshold that justifies it -- capped at the
						// classic 11 of a two-over-one, never above what is
						// actually held.
						reverseLo := 25 - partner.shownMin
						if reverseLo > 11 {
							reverseLo = 11
						}
						if reverseLo < 0 {
							reverseLo = 0
						}
						if cMin < 25 {
							continue
						}
						if reverseLo > lo {
							lo = reverseLo
						}
					}
					mn := m(lo, 40, "bicolore du répondant, forcing", "responder's second suit, forcing").asForcing()
					mn = mn.withLen(s, p.hand.Len(s)).withLen(firstSuit, p.hand.Len(firstSuit))
					return c, mn, true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Nine tricks off the top. Partner has repeated his suit at the
			// three level in a contested auction, promising six cards, and we
			// hold two of them: an eight-card fit, a suit that will run once
			// we are in. With a stopper in every suit the opponents named,
			// 3NT beats both the partial and the eleven tricks five of a minor
			// would take. The plain count never finds it -- partner's sixth
			// and seventh cards are tricks the honour scale does not see --
			// which is why the floor sits about five points under the 25 the
			// 3NT threshold asks for elsewhere [E-9].
			//
			// A single stopper is enough here, where [E-7] would demand a
			// double one of a suit named twice: with partner's suit running
			// there is no hold-up to play, and no lead to hand back.
			name: "notrump-over-partner-long-suit",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.hasBid && ctx.lastSeat == partnerOf(ctx.p.seat) &&
					ctx.last.Strain <= SSpades && ctx.last.Level >= 3 &&
					len(e.opponentSuits(ctx.p)) > 0
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner := ctx.p, ctx.partner
				s := Suit(ctx.last.Strain)
				if partner.shownLens[s] < 6 || p.hand.Len(s) < 2 {
					return Call{}, meaning{}, false
				}
				if ctx.hasFit && ctx.fit.IsMajor() {
					return Call{}, meaning{}, false // a major game is the better shot
				}
				if p.hand.H()+partner.shownMin < 20 {
					return Call{}, meaning{}, false
				}
				for _, os := range e.opponentSuits(p) {
					if !p.hand.Stopper(os) {
						return Call{}, meaning{}, false
					}
				}
				c := bid(3, SNoTrump)
				if e.cheapestCall(SNoTrump) != c || !e.legal(p.seat, c) {
					return Call{}, meaning{}, false
				}
				mn := m(p.hand.H(), 40, "3SA : arrêt dans leur couleur et fit dans la couleur longue du partenaire", "3NT: their suit stopped and a fit for partner's long suit")
				for _, os := range e.opponentSuits(p) {
					mn.stops[os] = true
				}
				return c, mn, true
			},
		},
		{
			// Competitive sacrifice (law of total tricks + the scoring
			// table). Only in a contested auction, when the opponents' last
			// bid is a game their shown strength makes them favourites to
			// bring home. Each side runs its own evaluation. Two options,
			// tried in order:
			//  - bid on to make ("surenchère") when the combined count
			//    affords the extra level(s): the game threshold plus about
			//    three points per level above game;
			//  - otherwise sacrifice in the side's nine-card-plus fit when
			//    the doubled penalty (expected tricks = combined trumps,
			//    law of total tricks) costs less than what the opponents
			//    would score for their contract, vulnerability of each side
			//    taken into account.
			name: "competitive-sacrifice",
			fr:   "les adversaires ont demandé la manche et nous avons un fit : surenchérir, sacrifier ou les laisser jouer",
			en:   "the opponents bid game and we hold a fit: outbid them, sacrifice or let them play",
			when: func(e *Engine, ctx *concludeCtx) bool {
				// Their bid must be a contract they mean to play. The slam
				// machinery lands at the game level too -- control cue-bids,
				// the 4NT ask, its answer, the conventional relays -- and a
				// cue-bid often names our own suit: saving over any of them
				// buys the five level against a contract nobody has named.
				if sc, ok := e.lastCallBy(ctx.lastSeat); ok &&
					(sc.M.controlBid || sc.M.blackwood || sc.M.keyResp || sc.M.ctrlRelay || sc.M.relay) {
					return false
				}
				return ctx.hasBid && !ctx.ours && ctx.hasFit && isGame(ctx.last) &&
					(ctx.p.bids > 0 || ctx.partner.bids > 0) && !e.bw[ctx.side].asked
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				return e.competitiveSacrifice(ctx)
			},
		},
		{
			// Penalty double of a sacrifice: the opponents have outbid the
			// game our side had reached in a contested auction, on strength
			// too thin to make their call -- a save. When we cannot
			// profitably bid on (the sacrifice/surenchère handler above has
			// already declined), the double is the only way to collect what
			// the scoring table promises: undoubled, their two-down save
			// costs them 100 where our vulnerable game was worth 620.
			name: "penalty-double-of-sacrifice",
			fr:   "notre manche a été dépassée par les adversaires : sacrifice ou contrat sérieux ?",
			en:   "the opponents outbid our game: a sacrifice or a real contract?",
			when: func(e *Engine, ctx *concludeCtx) bool {
				if !ctx.hasBid || ctx.ours || !e.legal(ctx.p.seat, doubleCall) {
					return false
				}
				// Our side must have bid a game of its own -- on real
				// values, not as a save of ours (the combined floor holds
				// the game zone).
				ourGame := false
				for _, sc := range e.calls {
					if sideOf(sc.Seat) == ctx.side && isGame(sc.Call) {
						ourGame = true
						break
					}
				}
				return ourGame && ctx.cMin >= gameThresholdFor(ctx.fit, ctx.hasFit, ctx.ntOK)
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p := ctx.p
				// Only when the opponents' shown strength marks their call
				// as a sacrifice rather than a constructive auction of
				// their own: a genuinely strong side may still make it, and
				// the double would only inflate their score.
				oppMin := e.ps[(p.seat+1)%4].shownMin + e.ps[(p.seat+3)%4].shownMin
				if !ctx.tr.check(oppMin < 23, "moins de 23 H montrés par les deux adversaires : c'est un sacrifice → contre punitif",
					"under 23 H shown by the two opponents: it is a sacrifice → penalty double", pts(oppMin, "H")) {
					return Call{}, meaning{}, false
				}
				return doubleCall, m(-1, -1, "contre punitif du sacrifice", "penalty double of the sacrifice"), true
			},
		},
		{
			// Misfit retreat: partner's last call is a natural, non-forcing
			// suit bid that finds this hand with at most one card there and
			// no seven-card combined holding. Passing would freeze a 6-1 or
			// worse "fit"; with a six-card suit of its own already shown, the
			// hand corrects to it at the lowest level (never beyond its game).
			name: "misfit-retreat",
			fr:   "misfit : un singleton ou une chicane dans la couleur du partenaire, pas de fit septième",
			en:   "misfit: a singleton or void in partner's suit, no seven-card fit",
			when: func(e *Engine, ctx *concludeCtx) bool {
				if !ctx.hasBid || !ctx.ours || ctx.lastSeat != partnerOf(ctx.p.seat) ||
					ctx.hasFit || ctx.pm == nil || ctx.pm.forcing || ctx.pm.relay || ctx.pm.blackwood ||
					!ctx.last.IsBid() || ctx.last.Strain > SSpades {
					return false
				}
				s := Suit(ctx.last.Strain)
				return ctx.p.hand.Len(s) <= 1 && ctx.p.hand.Len(s)+ctx.partner.shownLens[s] < 7
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, last := ctx.p, ctx.last
				for s := Spades; s >= Clubs; s-- {
					if p.shownLens[s] < 5 || p.hand.Len(s) < 6 {
						continue
					}
					c := e.cheapestCall(s.Strain())
					if !c.higherThan(last) || c.steps() > gameOfTrump(s).steps() || !e.legal(p.seat, c) {
						continue
					}
					ctx.tr.check(true, "couleur sixième déjà nommée → y revenir au plus bas", "six-card suit already bid → back to it at the lowest level", cards(p.hand, s))
					return c, m(-1, -1, "misfit : retour dans la couleur sixième", "misfit: corrects to the six-card suit").withLen(s, p.hand.Len(s)), true
				}
				return Call{}, meaning{}, false
			},
		},
		{
			// Facing partner's limited preempt (weak two, three- or four-level
			// barrage): the raw combined count grossly understates seven
			// playing tricks headed by a long suit, so a strong hand bids the
			// obvious game itself instead of adding up to a partscore.
			name: "raise-partner-preempt-to-game",
			when: func(e *Engine, ctx *concludeCtx) bool {
				return ctx.partner.shownMax <= 10 && ctx.p.hand.H() >= 16 && !e.bw[ctx.side].asked
			},
			run: func(e *Engine, ctx *concludeCtx) (Call, meaning, bool) {
				p, partner, last := ctx.p, ctx.partner, ctx.last
				pre, preLen := Clubs, 0
				for s := Clubs; s <= Spades; s++ {
					if partner.shownLens[s] >= 6 && partner.shownLens[s] > preLen {
						pre, preLen = s, partner.shownLens[s]
					}
				}
				if preLen < 6 {
					return Call{}, meaning{}, false
				}
				if pre.IsMajor() && p.hand.Len(pre) >= 2 {
					c := bidSuit(4, pre)
					if c.higherThan(last) && e.legal(p.seat, c) {
						return c, m(-1, -1, "conclusion à la manche sur le barrage, main forte", "raises the preempt to game, strong hand").withLen(pre, p.hand.Len(pre)), true
					}
				}
				if !pre.IsMajor() {
					stopped := true
					for s := Clubs; s <= Spades; s++ {
						if s != pre && !p.hand.Stopper(s) {
							stopped = false
							break
						}
					}
					if stopped {
						c := bid(3, SNoTrump)
						if c.higherThan(last) && e.legal(p.seat, c) {
							return c, m(-1, -1, "3SA sur le barrage, main forte avec tenues", "3NT over the preempt, strong hand with stoppers"), true
						}
					}
				}
				return Call{}, meaning{}, false
			},
		},
	}
}

// concludeGameDecision is the shared value-based endgame conclude falls back
// to once nothing in concludeHandlers claims the call: add up the
// partnership's points and steer to the right partscore, game or slam.
func (e *Engine) concludeGameDecision(ctx *concludeCtx) (Call, meaning) {
	p, partner, pm := ctx.p, ctx.partner, ctx.pm
	last, ours, partnerJustActed := ctx.last, ctx.ours, ctx.partnerJustActed
	side, own := ctx.side, ctx.own
	cMin, cMax, cMinNT, cMaxNT, ntOK := ctx.cMin, ctx.cMax, ctx.cMinNT, ctx.cMaxNT, ctx.ntOK
	fit, hasFit := ctx.fit, ctx.hasFit
	tr := ctx.tr

	// The count the decision rests on, as the trace shows it.
	unit := "H"
	fitVal := ""
	if hasFit {
		unit = "HLD"
		fitVal = fmt.Sprintf("%d + %d %s", p.hand.Len(fit), partner.shownLens[fit], suitSymbol[fit])
	}
	tr.note("aucune convention en cours : décision sur la force combinée (la main + le minimum-maximum montré par le partenaire)",
		"no convention under way: decision on combined strength (your hand + the minimum-maximum partner has shown)")
	tr.check(hasFit, "fit de 8 cartes et plus connu", "known fit of 8+ cards", fitVal)
	if !(hasFit && fit.IsMajor()) {
		tr.check(ntOK, "Sans-Atout jouable : arrêt dans chaque couleur adverse",
			"notrump playable: a stopper in every opposing suit", "")
	}

	// Game decision.
	gc, wantGame := gameCall(fit, hasFit, cMin, cMinNT, ntOK)
	gameVal := fmt.Sprintf("%d %s + %d = %d", own, unit, partner.shownMin, cMin)
	if hasFit && !fit.IsMajor() {
		gameVal += fmt.Sprintf(" (SA : %d H + %d = %d)", p.hand.H(), partner.shownMin, cMinNT)
	}
	tr.check(wantGame,
		"minimum combiné suffisant pour une manche : 27 HLD en majeure, 25 H à Sans-Atout, 30 HLD en mineure",
		"combined minimum enough for game: 27 HLD in a major, 25 H in notrump, 30 HLD in a minor", gameVal)
	if wantGame {
		tr.in()
		defer tr.out()
	}
	if wantGame && !(hasFit && fit.IsMajor()) &&
		!p.hand.IsRegular() && !p.hand.IsSemiRegular() {
		// An unbalanced hand with a six-card major it has already shown
		// belongs in four of that major, not 3NT (or a minor game):
		// opposite partner's balanced rebid the major is at worst a 6-2
		// fit, while a singleton or void makes notrump run off the top.
		for _, s := range []Suit{Spades, Hearts} {
			if p.shownLens[s] >= 4 && p.hand.Len(s) >= 6 {
				tr.check(true, "main irrégulière avec une majeure sixième déjà nommée : la manche dans cette majeure",
					"unbalanced hand with a six-card major already bid: game in that major", cards(p.hand, s))
				// Partner's balanced notrump rebid guarantees a doubleton,
				// so the sixth trump certifies an eight-card fit: revalue
				// the hand with the trump-fit distribution points. When the
				// revalued minimum reaches the slam zone, explore with
				// control bids instead of closing in game (docs/addon_4.md,
				// "mécanisme des contrôles").
				if psc, ok := e.lastCallBy(partner.seat); ok && psc.Call.IsBid() && psc.Call.Strain == SNoTrump {
					reval := p.hand.HLD(s)
					for os := Clubs; os <= Spades; os++ {
						if os == s || p.hand.Len(os) != 1 {
							continue
						}
						switch {
						case p.hand.HasCard(os, 'Q'):
							reval -= 2
						case p.hand.HasCard(os, 'J'):
							reval -= 1
						}
					}
					if tr.check(reval+partner.shownMin >= 31 && !e.bw[side].asked,
						"majeure sixième revalorisée : zone de chelem (31) → enchères de contrôle",
						"six-card major revalued: slam zone (31) → control bids",
						fmt.Sprintf("%d HLD + %d = %d", reval, partner.shownMin, reval+partner.shownMin)) {
						c, mn, ok := e.controlBid(p, s)
						// The cheapest control can fall on the very call
						// partner's rebid has reserved for his convention
						// (3C Checkback, 2C Roudi): he would answer the
						// question, not hear the control. Repeat the major
						// instead -- forcing, and the suit the controls are
						// about, which the exchange had never named -- and
						// start the cue-bids next round, above the call the
						// convention owns.
						switch {
						case ok && e.reservedByOffer(p, c):
							if rc := e.cheapestCall(s.Strain()); rc.Level <= 3 && e.legal(p.seat, rc) {
								// The revalued count travels with the bid: it
								// is what the control it replaces would have
								// implied, and without it partner reads a mere
								// repeat and signs off in game.
								return rc, m(reval, -1,
									"majeure sixième répétée, forcing : l'atout avant les contrôles",
									"repeats the six-card major, forcing: trumps agreed before the controls").
									withLen(s, p.hand.Len(s)).asForcing()
							}
						case ok:
							return c, mn
						}
					}
				}
				if mc := bidSuit(4, s); e.legal(p.seat, mc) {
					gc, fit, hasFit = mc, s, true
				}
				break
			}
		}
	}
	if wantGame {
		// Five of a minor is two tricks dearer than 3SA for the same bonus:
		// pulling partner's notrump game there is only right when notrump is
		// unsafe, and partner -- who bid it -- has just said it is not. Slam
		// belongs to the handlers above; here the count only reaches game, so
		// settle rather than climb.
		pulls3NT := ours && last == bid(3, SNoTrump) && gc.Level >= 5 && hasFit && !fit.IsMajor()
		if tr.check(ours && (last.steps() >= gc.steps() || pulls3NT),
			"notre camp a déjà atteint ce palier (ou le partenaire a choisi 3SA)",
			"our side already reached that level (or partner chose 3NT)", "") {
			return e.settleAboveGameCall(p, fit, hasFit, last, pm, partnerJustActed, tr)
		}
		gcFR, gcEN := callSym(gc)
		if tr.check(e.legal(p.seat, gc), "→ la manche : "+gcFR, "→ game: "+gcEN, "") {
			mn := m(own-1, -1, "conclusion à la manche sur la force combinée", "bids game on combined strength")
			if hasFit && gc.Kind == KindBid && gc.Strain != SNoTrump {
				// Record the real trump length so partner can still read
				// the fit afterwards (e.g. to launch a slam try) rather
				// than lose track of it and fall back to notrump.
				mn = mn.withLen(fit, p.hand.Len(fit))
			}
			return gc, mn
		}
		tr.note("la manche n'est plus possible → Passe", "game is no longer available → Pass")
		return passCall, noInfo()
	}

	// Invitation zone.
	threshold := gameThresholdFor(fit, hasFit, ntOK)
	// A responder who passed partner's opening has denied the values to
	// respond (< 6 HCP): it must not now initiate a game invitation.
	//
	// The ceiling that pass advertised is not always low enough to say so on
	// its own. Over an overcall the same pass only denies 11 HL, and is read
	// as capped at 7 or 10 [RC-4], so a hand that had nothing whatever to say
	// slips through the shownMax test and then invites on the arithmetic of a
	// 12-23 opening's maximum. Every response has a floor of 6 [RC-3]; a hand
	// that never cleared it does not clear it later either.
	weakPassedResponder := p.seat != e.opener && sideOf(p.seat) == sideOf(e.opener) &&
		p.bids == 0 && ((p.shownMax >= 0 && p.shownMax < 6) || own < 6)
	// An invitation promises at least threshold minus partner's maximum. A
	// hand that has already limited itself below that must not invite.
	capped := p.shownMax >= 0 && p.shownMax < threshold-partner.shownMax
	// An invitation must be worth hearing. Partner's shown maximum is often
	// the 23 of an opening that has limited nothing -- 1D - 1H - 1S says
	// nothing of opener's force [RO-19] -- and the arithmetic then clears the
	// threshold on any hand whatever: a 9-count raised to 3S "game
	// invitation" where the response scheme gives 6-10 the simple raise and
	// reserves the jump for 11-12 HLD [RM-4], [Rm-7]. So the hand must hold
	// the invitational zone in its own right, unless partner has already
	// limited himself high enough that the points still missing are really
	// his to hold.
	belowInviteZone := own < inviteFloor && cMin < threshold-2
	inviteZone := tr.check(cMax >= threshold,
		fmt.Sprintf("maximum combiné au seuil de la manche (%d) → proposition de manche possible", threshold),
		fmt.Sprintf("combined maximum reaches the game threshold (%d) → game invitation possible", threshold),
		fmt.Sprintf("%d %s + %d = %d", own, unit, partner.shownMax, cMax))
	if inviteZone {
		tr.in()
		switch {
		case p.invited:
			tr.check(false, "pas encore de proposition faite", "no invitation made yet", "")
		case partner.bids == 0:
			tr.check(false, "le partenaire a déjà enchéri", "partner has already bid", "")
		case weakPassedResponder:
			tr.check(false, "pas un répondant qui a passé faute de 6 H", "not a responder who passed for lack of 6 H", pts(own, unit))
		case capped:
			tr.check(false, "main pas encore limitée sous la proposition", "hand not already limited below an invitation", "")
		}
	}
	if cMax >= threshold && !p.invited && partner.bids > 0 && !weakPassedResponder && !capped {
		var c Call
		canInvite := true
		// The two-level rebid of our own long major below is not really an
		// invitation but a description partner is free to pass, so the floor
		// above does not apply to it.
		raisesTheLevel := true
		// A six-card major we have already bid describes the hand far
		// better than a notrump proposal when the hand is unbalanced.
		longMajor, hasLongMajor := Clubs, false
		for _, s := range []Suit{Spades, Hearts} {
			if p.shownLens[s] >= 4 && p.hand.Len(s) >= 6 {
				longMajor, hasLongMajor = s, true
				break
			}
		}
		unbalanced := !p.hand.IsRegular() && !p.hand.IsSemiRegular()
		switch {
		case tr.check(hasFit && fit.IsMajor(), "fit majeur → proposer au palier de 3", "major fit → invite at the three level", fitVal):
			c = bidSuit(3, fit)
		case tr.check(hasLongMajor && unbalanced && bidSuit(2, longMajor).higherThan(last),
			"main irrégulière, majeure sixième déjà nommée → la répéter à 2",
			"unbalanced, six-card major already bid → repeat it at the two level", shape(p.hand)):
			c, raisesTheLevel = bidSuit(2, longMajor), false
		case tr.check(ntOK, "Sans-Atout jouable → 2SA", "notrump playable → 2NT", ""):
			c = bid(2, SNoTrump)
		case tr.check(hasFit, "fit mineur sans arrêt → proposer au palier de 3", "minor fit without stoppers → invite at the three level", fitVal):
			// Minor fit with notrump unavailable (no stopper in the
			// opponents' suit): raise the minor to invite rather than pass
			// out a nine-card fit at partscore.
			c = bidSuit(3, fit)
		default:
			canInvite = false
		}
		// A notrump invitation must reach the threshold on high cards alone.
		// cMax values the hand by HLD as soon as a fit is agreed, and a minor
		// fit thus lends the hand shortness points that buy no trick at all in
		// notrump: without this guard a 4-1-3-5 eight-count facing a 12-14
		// 1NT rebid invited game on a 22-point maximum (same reasoning as
		// cMinNT for the game decision above).
		if canInvite && c.IsBid() && c.Strain == SNoTrump && cMaxNT < threshold {
			tr.check(false, "le seuil est atteint en points d'honneur seuls (Sans-Atout)",
				"the threshold is reached on high cards alone (notrump)",
				fmt.Sprintf("%d H + %d = %d", p.hand.H(), partner.shownMax, cMaxNT))
			canInvite = false
		}
		if canInvite && raisesTheLevel && belowInviteZone {
			tr.check(false, fmt.Sprintf("la main vaut elle-même une proposition (%d et plus)", inviteFloor),
				fmt.Sprintf("the hand is itself worth an invitation (%d+)", inviteFloor), pts(own, unit))
			canInvite = false
		}
		cFR, cEN := callSym(c)
		if canInvite && tr.check(c.higherThan(last) && e.legal(p.seat, c) && (ours || pm != nil),
			"→ proposition de manche : "+cFR+" (encore disponible)", "→ game invitation: "+cEN+" (still available)", "") {
			p.invited = true
			mn := m(threshold-partner.shownMax, threshold-1-partner.shownMin, "proposition de manche", "game invitation").asInvite()
			if mn.minPts < 0 {
				mn.minPts = 0
			}
			if mn.maxPts < mn.minPts {
				mn.maxPts = mn.minPts
			}
			// In a game-forcing auction (splinter, two-over-one...) the same
			// low bid stays useful as a descriptive step, but it must not
			// be an invitation partner could decline below game: mark it
			// forcing.
			if e.gameForce[side] {
				mn.invite = false
				mn.forcing = true
				mn.fr, mn.en = "enchère descriptive, forcing de manche", "descriptive bid, game forcing"
			}
			if hasFit && c.Strain == fit.Strain() {
				mn = mn.withLen(fit, p.hand.Len(fit))
			} else if c.IsBid() && c.Strain != SNoTrump {
				mn = mn.withLen(longMajor, p.hand.Len(longMajor))
			}
			return c, mn
		}
	}
	if inviteZone {
		tr.out()
	}

	// Forced to bid?
	if tr.check(pm != nil && pm.forcing && partnerJustActed && ours,
		"le partenaire vient de faire une enchère forcing → enchère au plus bas palier",
		"partner has just made a forcing bid → cheapest constructive call", "") {
		return e.cheapestConstructive(p, fit, hasFit)
	}

	// Last resort: the side committed to game earlier (a splinter, a
	// two-over-one response) but nothing above already got there or kept
	// the auction alive -- bid game now rather than pass it out below game.
	if e.gameForce[side] && !isGame(last) {
		gc, ok := gameCall(fit, hasFit, 99, 99, ntOK)
		if hasFit && fit.IsMajor() {
			gc, ok = bidSuit(4, fit), true
		}
		if tr.check(ok && gc.higherThan(last) && e.legal(p.seat, gc),
			"séquence forcing de manche pas encore conclue → la manche",
			"game-forcing auction not yet at game → bid game", "") {
			return gc, m(-1, -1, "conclusion à la manche, forcing de manche engagé", "bids game, the auction is game forcing")
		}
	}

	// Nothing left to bid for value -- but a six-card major of our own is a
	// playable strain the arithmetic never sees.
	if c, mn, ok := e.ownLongMajorSignoff(ctx); tr.check(ok,
		"majeure sixième à nous, sous la zone de proposition → s'y retirer au palier de 2",
		"a six-card major of our own, below the invitation zone → retreat into it at the two level", callOrEmpty(c, ok)) {
		return c, mn
	}
	// Below the invitational zone the fit still has to be shown: the simple
	// raise is the bid that hand was always meant to make.
	if c, mn, ok := e.supportRaise(ctx); tr.check(ok,
		fmt.Sprintf("fit pas encore montré, 6-%d %s → soutien simple", inviteFloor-1, "HLD"),
		fmt.Sprintf("fit not shown yet, 6-%d %s → simple raise", inviteFloor-1, "HLD"), callOrEmpty(c, ok)) {
		return c, mn
	}
	// Nothing left to bid for value -- but the fit may still have something
	// to say in a partscore battle.
	if c, mn, ok := e.competitivePartscore(ctx); tr.check(ok,
		"loi des levées totales : 9 atouts valent le palier de 3, 10 le palier de 4 → ne pas leur laisser la partielle",
		"law of total tricks: 9 trumps are worth the three level, 10 the four level → don't leave them the partscore",
		callOrEmpty(c, ok)) {
		return c, mn
	}
	// Before passing out partner's non-forcing two-suiter, take back his
	// first suit when it is the better strain.
	if c, mn, ok := e.preferenceBack(ctx); tr.check(ok,
		"bicolore non forcing du partenaire, sa première couleur est le meilleur fit → préférence",
		"partner's non-forcing two-suiter, whose first suit is the better fit → preference", callOrEmpty(c, ok)) {
		return c, mn
	}
	tr.note("rien à ajouter → Passe", "nothing more to say → Pass")
	return passCall, noInfo()
}

// callOrEmpty shows the call a rule produced, for the trace: nothing when the
// rule did not apply.
func callOrEmpty(c Call, ok bool) string {
	if !ok {
		return ""
	}
	fr, _ := callSym(c)
	return fr
}

// supportRaise shows the fit when the count has just declined to invite.
// Everywhere in the scheme the same split applies: 6-10 HLD makes the simple,
// non-forcing raise and 11 opens the invitational zone [RM-4], [Rm-7]. Under
// that floor the raise is not a consolation prize, it is the natural bid --
// passing partner's freshly named suit with the fit buries eight trumps at
// the level he happened to reach. Nothing more is promised: the meaning
// carries the 6-10 ceiling so partner reads it as the limit it is.
func (e *Engine) supportRaise(ctx *concludeCtx) (Call, meaning, bool) {
	p, partner := ctx.p, ctx.partner
	if !ctx.hasFit || !ctx.ours || !ctx.partnerJustActed || ctx.lastSeat != partner.seat {
		return Call{}, meaning{}, false
	}
	if !ctx.last.IsBid() || ctx.last.Strain != ctx.fit.Strain() {
		return Call{}, meaning{}, false
	}
	if p.invited || ctx.own < 6 || ctx.own >= inviteFloor {
		return Call{}, meaning{}, false
	}
	// Only the hand that has not yet shown the fit: once the support is on
	// the table it has been heard, and re-raising it is not a second piece of
	// news -- it is the two hands pushing each other up a suit neither has
	// the values for.
	if p.shownLens[ctx.fit] > 0 {
		return Call{}, meaning{}, false
	}
	// The cheapest call in the fit must be the simple raise itself: a jump
	// would say more than 6-10, and anything higher means the auction has
	// already passed this bid by. The two level is the ceiling -- a hand under
	// the invitational floor buys nothing higher, and past it partner's own
	// bid has already told a different story.
	c := e.cheapestCall(ctx.fit.Strain())
	if c.Level != ctx.last.Level+1 || c.Level > 2 || !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	return c, m(6, inviteFloor-1, "soutien simple", "simple raise").withLen(ctx.fit, p.hand.Len(ctx.fit)), true
}

// competitivePartscore applies the law of total tricks [L-1] to the partscore
// battle the point count has just declined to enter. The opponents own the
// contract, our side holds a known nine-card fit, and the law says that fit
// is worth the three level whatever the honour count -- ten trumps, the four
// level. Letting them play two of their suit while nine trumps sit unused is
// the one thing the law forbids, and no count-based rule above will ever say
// so: they all measure the road to game, which this deal has already left.
//
// The level the fit affords is also the ceiling: once our side has bid it,
// the cheapest call in the fit lands above the law and the rule goes quiet,
// so the auction cannot ping-pong. Nothing is promised in points -- the bid
// is made on trump length alone and must not read as values partner could
// raise.
func (e *Engine) competitivePartscore(ctx *concludeCtx) (Call, meaning, bool) {
	p, partner := ctx.p, ctx.partner
	// Only against a live opposing partscore: their bid must be the one on
	// the table (ours needs no rescuing), below the game the sacrifice rules
	// already weigh, and low enough that the law can still speak over it.
	if !ctx.hasFit || !ctx.hasBid || ctx.ours || isGame(ctx.last) || ctx.last.Level > 3 {
		return Call{}, meaning{}, false
	}
	trumps := p.hand.Len(ctx.fit) + partner.shownLens[ctx.fit]
	lawLevel := 0
	switch {
	case trumps >= 10:
		lawLevel = 4
	case trumps >= 9:
		lawLevel = 3
	}
	if lawLevel == 0 {
		return Call{}, meaning{}, false
	}
	c := e.cheapestCall(ctx.fit.Strain())
	if c.Level > lawLevel || !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	return c, m(-1, -1,
		"loi des levées totales : le fit vaut le palier, on ne leur laisse pas la partielle",
		"law of total tricks: the fit is worth the level, not leaving them the partscore").
		withLen(ctx.fit, p.hand.Len(ctx.fit)), true
}

// preferenceBack gives partner's first suit the preference over the cheap
// second suit he has just named, when the auction is about to die there.
// The bicolore économique is non-forcing and leaves the choice of trump to
// responder (docs/bidings.md, "EN FACE D'UN BICOLORE ÉCONOMIQUE", whose
// criteria the 1SA response reuses: "il est toujours possible de passer ou
// de donner une préférence pour la première couleur de l'ouvreur selon les
// mêmes critères que précédemment"). He passes only with *two* cards fewer
// in the first suit than in the second, and otherwise returns to it at the
// two level holding at least two cards there. Against a 5-4 opener that is
// the same thing as comparing the two fits, which is how it is expressed
// here: on 1♠ - 1SA - 2♦, five plus two spades and four plus three diamonds
// are both seven-card holdings, so the major wins -- it plays no worse on
// partner's longer trump suit and its partscore is worth half as much again.
func (e *Engine) preferenceBack(ctx *concludeCtx) (Call, meaning, bool) {
	p, partner, last := ctx.p, ctx.partner, ctx.last
	if ctx.hasFit || ctx.lastSeat != partner.seat || !ctx.partnerJustActed {
		return Call{}, meaning{}, false
	}
	if ctx.pm != nil && (ctx.pm.forcing || ctx.pm.invite) {
		return Call{}, meaning{}, false
	}
	if !last.IsBid() || last.Strain == SNoTrump {
		return Call{}, meaning{}, false
	}
	fb, ok := e.firstBidBy(partner.seat)
	if !ok || !fb.IsBid() || fb.Strain == SNoTrump || fb.Strain == last.Strain {
		return Call{}, meaning{}, false
	}
	first, second := Suit(fb.Strain), Suit(last.Strain)
	// Only a first suit known to be the longer one is worth going back to:
	// the opening major promises five cards, the second suit four. A minor
	// opening, promising three, tells us nothing to prefer.
	if partner.shownLens[first] < 5 || partner.shownLens[second] < 4 ||
		partner.shownLens[first] <= partner.shownLens[second] {
		return Call{}, meaning{}, false
	}
	if p.hand.Len(first) < 2 ||
		p.hand.Len(first)+partner.shownLens[first] < p.hand.Len(second)+partner.shownLens[second] {
		return Call{}, meaning{}, false
	}
	c := bidSuit(last.Level, first)
	if !c.higherThan(last) || !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	return c, m(-1, -1, "préférence pour la première couleur", "preference back to the first suit").withLen(first, p.hand.Len(first)), true
}

// ownLongMajorSignoff retreats the responder into a six-card major of his own
// when the count has nothing left to aim at.
//
// The game and invitation zones above already promote such a major, but only
// once their own threshold is reached; below it, and whenever no game contract
// exists at all -- no fit, and no stopper for notrump -- the arithmetic finds
// nothing to bid and passes out a hand that owns a trump suit. Opener's
// notrump rebid is balanced and promises at least a doubleton, so the sixth
// card certifies an eight-card fit: two of the major is a better partscore
// than his notrump, which a singleton in responder's hand runs off the top.
//
// The bid promises nothing. It names a strain, and opener is free to pass --
// which is why it stops at the two level: past it the retreat would be buying
// a level instead of choosing a denomination.
func (e *Engine) ownLongMajorSignoff(ctx *concludeCtx) (Call, meaning, bool) {
	p, partner := ctx.p, ctx.partner
	// Responder only: opener's partner may have answered 1NT "poubelle",
	// which denies a fit without promising a balanced hand, and a retreat
	// there could land on a singleton.
	if p.seat == e.opener || sideOf(p.seat) != sideOf(e.opener) {
		return Call{}, meaning{}, false
	}
	if ctx.hasFit || p.invited || e.gameForce[ctx.side] {
		return Call{}, meaning{}, false
	}
	// Partner's notrump rebid must be the standing contract: it is what
	// promises the doubleton, and what we are retreating from.
	if !ctx.ours || ctx.lastSeat != partner.seat || ctx.last.Strain != SNoTrump {
		return Call{}, meaning{}, false
	}
	// Only for the hand the invitation zone left behind. A hand whose count
	// reaches the invitation belongs there, where the same major is already
	// promoted: retreating it would sell a game the side can afford.
	threshold := gameThresholdFor(ctx.fit, ctx.hasFit, ctx.ntOK)
	if ctx.own+partner.shownMax >= threshold {
		return Call{}, meaning{}, false
	}
	// The retreat says what not inviting said: the hand cannot reach the
	// threshold even facing partner's maximum. Recording that cap matters --
	// the bid names a six-card suit, and without a ceiling partner reads the
	// length as news and raises a hand that has none.
	ceiling := max(threshold-1-partner.shownMax, 0)
	for _, s := range []Suit{Spades, Hearts} {
		if p.shownLens[s] < 4 || p.hand.Len(s) < 6 {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if c.Level > 2 || !c.higherThan(ctx.last) || !e.legal(p.seat, c) {
			continue
		}
		return c, m(-1, ceiling,
			"retrait dans la majeure sixième : la redemande à Sans-Atout promet le doubleton, le fit est huitième",
			"retreat into the six-card major: the balanced notrump rebid promises a doubleton, so the fit is eight cards").withLen(s, 6), true
	}
	return Call{}, meaning{}, false
}

// topHonours counts the ace, king and queen h holds in s.
func topHonours(h *Hand, s Suit) int {
	n := 0
	for _, r := range []byte{'A', 'K', 'Q'} {
		if h.HasCard(s, r) {
			n++
		}
	}
	return n
}

// fitSym shows a trump suit in the trace: "♠".
func fitSym(s Suit) string { return suitSymbol[s] }
