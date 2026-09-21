package main

// plannedFn is a deferred decision evaluated when the player's turn comes back.
type plannedFn func() (Call, meaning)

type playerState struct {
	seat          int
	hand          *Hand
	shownMin      int // points promised so far
	shownMax      int
	shownLens     [4]int  // minimum suit lengths promised
	shownStop     [4]bool // suits in which this player has guaranteed a stopper
	bids          int     // non-pass calls made
	responded     bool    // first response to partner's opening already given
	invited       bool
	ctrlShown     [4]bool // suits where this player has shown a control
	ctrlDenied    [4]bool // suits where this player has denied any control
	shortShown    [4]bool // suits this player has disclosed as a singleton or void (splinter, Drury singleton answer)
	planned       plannedFn
	lastM         *meaning
	answeredAces  bool // responded to a strong opening with the ace-step convention, no shape shown yet
	acesShown     int  // aces disclosed by the ace-step response (a floor: the 3SA step covers "two mixed" as well as "three or more")
	acesExact     bool // the ace-step response pinned the exact ace count
	keycardsShown int  // disclosed keycards (trump ace/king, or a side ace) already shown by a control bid before any Blackwood ask
	wideDouble    bool // answered partner's takeout double with the 0-10 zone (no jump available: 2H over 1S)
	wideDoubleFit Suit // the major of that wide answer
	passedOpening bool // passed at its first turn before anyone had opened: capped below the opening threshold
	lightOpen     bool // opened light in third or fourth seat [O-7b]: below the [O-6] threshold, and never bids again of its own accord
}

type bwState struct {
	asked   bool
	asker   int
	trump   Suit
	noTrump bool // keycard ask with no agreed trump: plain ace count, no trump-king/queen step
	kingAsk bool // 4NT after a 2D opening: king ask (aces already located), trump queen is the fifth key
	keyStep int  // which rung of the answer ladder was taken (0 = lowest): disambiguates the paired counts of a trump ask, and holds the exact count for the notrump asks (rung = count there)
}

// Engine simulates the whole auction.
type Engine struct {
	hands     [4]*Hand
	dealer    int
	ps        [4]*playerState
	calls     []SeatCall
	opener    int // seat of the opening bid, -1 before
	openCall  Call
	bw        [2]bwState // per side (seat%2)
	gameForce [2]bool
	vul       [2]bool // vulnerability per side (seat%2), from the PBN Vulnerable tag
}

func NewEngine(d *Deal) *Engine {
	e := &Engine{hands: d.Hands, dealer: d.Dealer, opener: -1, vul: d.Vul}
	for i := range 4 {
		e.ps[i] = &playerState{seat: i, hand: d.Hands[i], shownMax: 40}
	}
	return e
}

func partnerOf(seat int) int { return (seat + 2) % 4 }
func sideOf(seat int) int    { return seat % 2 }

// Run plays out the auction and returns the recorded calls.
func (e *Engine) Run() []SeatCall {
	seat := e.dealer
	for !e.over() {
		var c Call
		var mn meaning
		if len(e.calls) >= 40 {
			c, mn = passCall, noInfo() // safety net: a stuck auction ends in passes
		} else {
			c, mn = e.decide(seat)
			c, mn = e.sanitize(seat, c, mn)
			// Safety net: a forcing bid by partner, with the right-hand
			// opponent silent, must never be passed. Whatever hole in the
			// decision tree produced the pass, keep the auction alive at the
			// lowest constructive level instead.
			if c.Kind == KindPass && len(e.calls) >= 2 {
				rho, pc := e.calls[len(e.calls)-1], e.calls[len(e.calls)-2]
				if rho.Call.Kind == KindPass && pc.Seat == partnerOf(seat) &&
					pc.Call.Kind == KindBid && pc.M.forcing {
					p := e.ps[seat]
					fit, hasFit := e.fitSuit(p)
					if c2, mn2 := e.cheapestConstructive(p, fit, hasFit); c2.Kind != KindPass {
						c, mn = c2, mn2
					}
				}
			}
			// Safety net: a side committed to game (splinter, two-over-one,
			// strong opening) must not let the auction die below it.
			if c.Kind == KindPass {
				if c2, mn2, ok := e.gameForceNet(seat); ok {
					c, mn = c2, mn2
				}
			}
		}
		e.record(seat, c, mn)
		seat = (seat + 1) % 4
	}
	return e.calls
}

func (e *Engine) over() bool {
	n := len(e.calls)
	if n < 4 {
		return false
	}
	hasBid := false
	for _, sc := range e.calls {
		if sc.Call.Kind != KindPass {
			hasBid = true
			break
		}
	}
	if !hasBid {
		return n >= 4
	}
	for i := n - 3; i < n; i++ {
		if e.calls[i].Call.Kind != KindPass {
			return false
		}
	}
	return true
}

// lastBid returns the highest (latest) bid in the auction and its seat.
func (e *Engine) lastBid() (Call, int, bool) {
	for i := len(e.calls) - 1; i >= 0; i-- {
		if e.calls[i].Call.IsBid() {
			return e.calls[i].Call, e.calls[i].Seat, true
		}
	}
	return Call{}, -1, false
}

func (e *Engine) legal(seat int, c Call) bool {
	switch c.Kind {
	case KindPass:
		return true
	case KindBid:
		last, _, ok := e.lastBid()
		return !ok || c.higherThan(last)
	case KindDouble:
		last, lastSeat, ok := e.lastBid()
		_ = last
		if !ok || sideOf(lastSeat) == sideOf(seat) {
			return false
		}
		// not already doubled
		for i := len(e.calls) - 1; i >= 0; i-- {
			if e.calls[i].Call.IsBid() {
				break
			}
			if e.calls[i].Call.Kind != KindPass {
				return false
			}
		}
		return true
	case KindRedouble:
		doubled := false
		for i := len(e.calls) - 1; i >= 0; i-- {
			k := e.calls[i].Call.Kind
			if k == KindBid {
				break
			}
			if k == KindDouble {
				doubled = true
				break
			}
			if k == KindRedouble {
				return false
			}
		}
		return doubled
	}
	return false
}

// sanitize repairs an intended call that interference made illegal.
func (e *Engine) sanitize(seat int, c Call, mn meaning) (Call, meaning) {
	if e.legal(seat, c) {
		return c, mn
	}
	if c.Kind != KindBid {
		return passCall, noInfo()
	}
	// Try the same strain one level up, once, if the hand is worth it.
	last, _, _ := e.lastBid()
	up := bid(last.Level, c.Strain)
	if !up.higherThan(last) {
		up = bid(last.Level+1, c.Strain)
	}
	p := e.ps[seat]
	worth := p.hand.HL() >= 11 || (c.Strain != SNoTrump && p.hand.Len(Suit(c.Strain))+e.ps[partnerOf(seat)].shownLens[Suit(c.Strain)] >= 8)
	if up.Level <= 4 && up.Level <= c.Level+1 && worth && e.legal(seat, up) {
		return up, mn
	}
	return passCall, noInfo()
}

// passReason puts words on a pass the decision tree left silent. Every pass
// the engine *means* carries its own text; these are the ones it reaches by
// running out of rules, and in the annotated auction a bare "Passe" reads as a
// hole -- worse, filtered out of the commented list it used to make the
// sequence skip a turn and show the players out of order.
//
// The text says only what the position itself guarantees: who owns the
// auction, and what this pass declines to do about it. It never invents a
// reason the code did not have -- in particular it never claims a hand is weak
// when it is not. A defender who holds an opening and still passes is told
// exactly that: the values are there, no bid describes the hand.
func (e *Engine) passReason(seat int) (fr, en string) {
	p := e.ps[seat]

	// Nobody has bid: the pass is a statement about the hand alone, and it is
	// the hardest one the hand will ever make [E-1].
	if e.opener < 0 {
		return "pas les conditions d'ouverture", "does not meet the opening requirements"
	}

	_, lastSeat, _ := e.lastBid()

	// Our own side holds the contract; the pass leaves it standing.
	if sideOf(lastSeat) == sideOf(seat) {
		if lastSeat == seat {
			return "rien à ajouter à sa propre enchère", "nothing to add to its own bid"
		}
		return "l'enchère du partenaire convient, rien à ajouter",
			"partner's call is the right contract, nothing to add"
	}

	// The opponents hold the contract. If our side has already described
	// something -- this hand or partner's -- the pass is a competitive
	// decision and not a confession of weakness: the level has simply gone
	// past what the side can pay for.
	spoken := p.bids > 0
	for _, sc := range e.calls {
		if sc.Seat == partnerOf(seat) && sc.Call.Kind != KindPass {
			spoken = true
		}
	}
	if spoken {
		return "l'enchère est aux adversaires, pas de quoi pousser plus haut",
			"the opponents own the auction, not enough to compete any higher"
	}

	// Our side has said nothing at all: this is the seat's own decision not
	// to enter.
	if p.hand.H() >= 12 {
		return "l'ouverture, mais aucune enchère ne décrit la main",
			"opening values, but no bid describes the hand"
	}
	return "pas de quoi intervenir", "not enough to act"
}

func (e *Engine) record(seat int, c Call, mn meaning) {
	p := e.ps[seat]
	if c.Kind == KindPass && mn.fr == "" {
		mn.fr, mn.en = e.passReason(seat)
	}
	if c.Kind != KindPass {
		p.bids++
	} else if p.bids == 0 && !p.passedOpening && !e.openCall.IsBid() {
		// A pass at one's first turn, before anyone has opened, caps the hand
		// below the opening threshold [E-1] -- and that cap is the hardest
		// fact the hand will ever tell partner.
		p.passedOpening = true
	}
	if mn.minPts >= 0 && mn.minPts > p.shownMin {
		v := mn.minPts
		if p.passedOpening && p.shownMax >= 0 && v > p.shownMax {
			// The later bid claims a floor the opening pass has already
			// denied. Point ranges attached to bids are conventional
			// descriptions; the pass is an observation about the actual
			// cards. When they contradict each other the pass wins, and the
			// claim is trimmed to the cap rather than the cap raised to the
			// claim -- which is what let a passed hand drive to a slam on
			// fourteen announced points.
			v = p.shownMax
		}
		if v > p.shownMin {
			p.shownMin = v
		}
	}
	if mn.maxPts >= 0 && mn.maxPts < p.shownMax {
		p.shownMax = mn.maxPts
	}
	if p.shownMax < p.shownMin {
		p.shownMax = p.shownMin
	}
	for s := Clubs; s <= Spades; s++ {
		if mn.lens[s] > p.shownLens[s] {
			p.shownLens[s] = mn.lens[s]
		}
		if mn.stops[s] {
			p.shownStop[s] = true
		}
		if mn.short[s] {
			p.shortShown[s] = true
		}
	}
	if mn.wideDouble {
		p.wideDouble, p.wideDoubleFit = true, mn.wideDoubleSuit
	}
	if mn.controlBid {
		p.ctrlShown[mn.controlSuit] = true
		if mn.keycardShown {
			p.keycardsShown++
		}
	}
	for s := Clubs; s <= Spades; s++ {
		if mn.deniedCtrl[s] {
			p.ctrlDenied[s] = true
		}
	}
	cm := mn
	p.lastM = &cm
	if c.IsBid() && e.opener == -1 {
		e.opener = seat
		e.openCall = c
	}
	e.calls = append(e.calls, SeatCall{Seat: seat, Call: c, M: mn})
}

// decide routes to the relevant decision function.
func (e *Engine) decide(seat int) (Call, meaning) {
	p := e.ps[seat]
	if p.planned != nil {
		fn := p.planned
		p.planned = nil
		return fn()
	}
	if e.opener == -1 {
		return e.opening(p)
	}
	if sideOf(seat) == sideOf(e.opener) {
		if seat == e.opener {
			if p.bids == 1 {
				return e.openerRebid(p)
			}
			return e.conclude(p)
		}
		if !p.responded {
			p.responded = true
			return e.respond(p)
		}
		return e.conclude(p)
	}
	// defending side
	partner := e.ps[partnerOf(seat)]
	if p.bids == 0 && partner.bids == 0 {
		return e.overcall(p)
	}
	if p.bids == 0 && partner.bids > 0 {
		return e.advance(p)
	}
	return e.conclude(p)
}

// ---------- generic late-auction logic ----------

// fitSuit finds the best known 8+ card fit (majors first, then longest).
func (e *Engine) fitSuit(p *playerState) (Suit, bool) {
	partner := e.ps[partnerOf(p.seat)]
	best, bestN := Clubs, 0
	found := false
	order := []Suit{Spades, Hearts, Diamonds, Clubs}
	for _, s := range order {
		n := p.hand.Len(s) + partner.shownLens[s]
		if n >= 8 {
			if !found || (s.IsMajor() && !best.IsMajor()) || (s.IsMajor() == best.IsMajor() && n > bestN) {
				best, bestN, found = s, n, true
			}
		}
	}
	return best, found
}

// hasHelp reports whether p's hand can plausibly cover losers in s, as asked
// by a help-suit game try: an ace or king there, a third-round queen, or
// shortness to ruff.
func (e *Engine) hasHelp(p *playerState, s Suit) bool {
	h := p.hand
	if h.HasCard(s, 'A') || h.HasCard(s, 'K') {
		return true
	}
	if h.Len(s) <= 2 {
		return true
	}
	return h.HasCard(s, 'Q') && h.Len(s) >= 3
}

// uncontested reports whether the opponents of the given seat have stayed
// silent (nothing but passes) so far.
func (e *Engine) uncontested(seat int) bool {
	for _, sc := range e.calls {
		if sideOf(sc.Seat) != sideOf(seat) && sc.Call.Kind != KindPass {
			return false
		}
	}
	return true
}

// opponentSuits returns the suits the opponents have bid naturally so far.
func (e *Engine) opponentSuits(p *playerState) []Suit {
	var suits []Suit
	var seen [4]bool
	for _, sc := range e.calls {
		if sideOf(sc.Seat) == sideOf(p.seat) || !sc.Call.IsBid() || sc.Call.Strain > SSpades {
			continue
		}
		s := Suit(sc.Call.Strain)
		if !seen[s] {
			seen[s] = true
			suits = append(suits, s)
		}
	}
	return suits
}

// uncontrolledSideSuit reports a side suit (other than the trump fit) where the
// hand holds two or more cards headed by neither the ace nor the king -- two
// potential fast losers -- in a suit the partnership has never bid and where
// partner has shown no control. Blackwood cannot detect such a hole (it counts
// keycards, not second-round control), so it is unsafe to launch it there.
func (e *Engine) uncontrolledSideSuit(p *playerState, fit Suit) bool {
	partner := e.ps[partnerOf(p.seat)]
	for s := Clubs; s <= Spades; s++ {
		if s == fit {
			continue
		}
		// A singleton or void is itself a first/second-round control; a top
		// honour guards the suit for the second round.
		if p.hand.Len(s) < 2 || p.hand.HasCard(s, 'A') || p.hand.HasCard(s, 'K') {
			continue
		}
		// Partner is known to cover: a shown control, or a genuine suit of
		// their own (opened, rebid or supported -- length promised elsewhere).
		if partner.ctrlShown[s] || partner.shownLens[s] >= 3 {
			continue
		}
		return true
	}
	return false
}

// ntSafe reports whether p can offer notrump: no opponent suit at all, or a
// stopper in every suit the opponents have bid -- and a *double* stopper in any
// suit they have bid and raised, since a lone hold in a known long suit is
// dislodged and the rest cashed against a notrump contract.
func (e *Engine) ntSafe(p *playerState) bool {
	// Count opponent bids per side-suit: a suit bid at least twice (overcall
	// plus raise, or bid and rebid) is a shown long, running holding.
	var oppBids [4]int
	for _, sc := range e.calls {
		if sideOf(sc.Seat) == sideOf(p.seat) || !sc.Call.IsBid() || sc.Call.Strain > SSpades {
			continue
		}
		oppBids[Suit(sc.Call.Strain)]++
	}
	partner := e.ps[partnerOf(p.seat)]
	for _, s := range e.opponentSuits(p) {
		// A stopper partner has guaranteed (e.g. a notrump bid "avec arrêt")
		// covers the suit just as one held in hand.
		if !p.hand.Stopper(s) && !partner.shownStop[s] {
			return false
		}
		if oppBids[s] >= 2 && !p.hand.DoubleStopper(s) && !partner.shownStop[s] {
			return false
		}
	}
	return true
}

// gameCall picks the game contract. cMin is the combined count valued for a
// suit contract (HLD, crediting the trump-fit shortness); cMinNT is the same
// count valued for notrump (honour strength only). The distinction matters when
// a minor fit steers toward 3NT: the doubletons and length that inflate HLD buy
// ruffing tricks in a suit contract, not notrump tricks, so the notrump game
// must clear the threshold on honours alone.
func gameCall(fit Suit, hasFit bool, cMin, cMinNT int, ntOK bool) (Call, bool) {
	if hasFit && fit.IsMajor() && cMin >= 27 {
		return bidSuit(4, fit), true
	}
	if hasFit && !fit.IsMajor() {
		// With a stopper in every opponent suit, 3NT (nine tricks) is a better
		// game than 5-of-the-minor (eleven); only fall back to the minor game
		// when notrump is unsafe.
		if cMinNT >= 25 && ntOK {
			return bid(3, SNoTrump), true
		}
		if cMin >= 30 {
			return bidSuit(5, fit), true
		}
	}
	if (!hasFit || !fit.IsMajor()) && ntOK {
		if cMinNT >= 25 {
			return bid(3, SNoTrump), true
		}
	}
	return Call{}, false
}

// gameThreshold is the combined count a game needs [E-9]: 27 HLD in a major
// fit, 30 HLD in a minor one -- eleven tricks have to be paid for -- and 25
// elsewhere, the notrump figure.
func gameThreshold(fit Suit, hasFit bool) int {
	switch {
	case hasFit && fit.IsMajor():
		return 27
	case hasFit:
		return 30
	}
	return 25
}

// inviteFloor is the bottom of the invitational zone. Every response scheme
// splits the same way: 6-10 HLD makes the simple, non-forcing bid and 11 is
// where a hand earns the right to ask partner for game ([RM-4], [Rm-7],
// [RC-3]). A hand below it has already said everything it holds.
const inviteFloor = 11

// gameThresholdFor weighs that bar against the game actually on offer. With a
// minor fit and notrump safe, the side bids 3NT -- nine tricks -- and 25 is
// the right bar; strip the stoppers away and five of the minor is the only
// game left, so the bar goes back to 30. A major fit always plays 4M.
func gameThresholdFor(fit Suit, hasFit, ntOK bool) int {
	if ntOK && !(hasFit && fit.IsMajor()) {
		return 25
	}
	return gameThreshold(fit, hasFit)
}

// gameForceNet is the last-resort bid of a side committed to game whose
// decision tree came up with a pass below it: reach the game — the major fit
// first, then 3SA behind stoppers, then the minor fit — or at worst keep the
// auction alive at the cheapest constructive level. It stays silent when the
// opponents own the last bid (their sacrifice is not ours to outbid here) or
// when game is already reached.
func (e *Engine) gameForceNet(seat int) (Call, meaning, bool) {
	side := sideOf(seat)
	if !e.gameForce[side] {
		return Call{}, meaning{}, false
	}
	last, lastSeat, has := e.lastBid()
	if !has || isGame(last) || sideOf(lastSeat) != side {
		return Call{}, meaning{}, false
	}
	p := e.ps[seat]
	fit, hasFit := e.fitSuit(p)
	var cands []Call
	if hasFit && fit.IsMajor() {
		cands = append(cands, gameOfTrump(fit))
	}
	if e.ntSafe(p) {
		cands = append(cands, bid(3, SNoTrump))
	}
	if hasFit && !fit.IsMajor() {
		cands = append(cands, gameOfTrump(fit))
	}
	for _, c := range cands {
		if c.higherThan(last) && e.legal(seat, c) {
			mn := m(-1, -1, "conclusion à la manche, forcing de manche engagé", "bids game, the auction is game forcing")
			if hasFit && c.Strain == fit.Strain() {
				mn = mn.withLen(fit, p.hand.Len(fit))
			}
			return c, mn, true
		}
	}
	if c, mn := e.cheapestConstructive(p, fit, hasFit); c.Kind != KindPass {
		return c, mn, true
	}
	return Call{}, meaning{}, false
}

// isGame reports whether a bid already is a game contract.
func isGame(c Call) bool {
	if !c.IsBid() {
		return false
	}
	switch {
	case c.Strain == SNoTrump:
		return c.Level >= 3
	case c.Strain >= SHearts:
		return c.Level >= 4
	default:
		return c.Level >= 5
	}
}

// settleAboveGameCall decides what to do with game values when our side has
// already outbid the natural game call (typically a forcing 4m below 3NT):
// raise a known fit to game, keep a forcing auction alive, else pass.
func (e *Engine) settleAboveGameCall(p *playerState, fit Suit, hasFit bool, last Call, pm *meaning, partnerJustActed bool) (Call, meaning) {
	if isGame(last) {
		return passCall, noInfo()
	}
	if hasFit {
		lvl := 4
		if !fit.IsMajor() {
			lvl = 5
		}
		c := bidSuit(lvl, fit)
		if c.higherThan(last) && e.legal(p.seat, c) {
			return c, m(-1, -1, "conclusion à la manche dans le fit", "raises the fit to game").withLen(fit, p.hand.Len(fit))
		}
	}
	if pm != nil && pm.forcing && partnerJustActed {
		return e.cheapestConstructive(p, fit, hasFit)
	}
	return passCall, noInfo()
}

// cheapestConstructive keeps a forcing sequence alive at the lowest level.
func (e *Engine) cheapestConstructive(p *playerState, fit Suit, hasFit bool) (Call, meaning) {
	last, _, _ := e.lastBid()
	partner := e.ps[partnerOf(p.seat)]
	// Support the partner suit with the best combined holding -- a genuine
	// seven-card holding at least, so a forced preference (which may have to
	// jump a level) never lands the side in a 4-2 "fit"; with less, the
	// waiting notrump below is the safer resting spot.
	var partnerSuit Suit
	bestComb := 0
	for s := Clubs; s <= Spades; s++ {
		if partner.shownLens[s] < 4 || p.hand.Len(s) < 2 {
			continue
		}
		if comb := p.hand.Len(s) + partner.shownLens[s]; comb >= 7 && comb > bestComb {
			bestComb, partnerSuit = comb, s
		}
	}
	if hasFit {
		c := bid(last.Level, fit.Strain())
		if !c.higherThan(last) {
			c = bid(last.Level+1, fit.Strain())
		}
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "soutien au plus bas palier", "lowest-level support").withLen(fit, p.hand.Len(fit))
		}
	}
	// Without a fit, take the cheapest of: the preference for partner's
	// suit, the repeat of a six-card suit already shown, the waiting
	// notrump. A preference that must jump a level while the six-card
	// repeat stays below it inflates the contract for nothing (e.g. 5C on
	// a doubleton when 4D was available).
	raise := func(st Strain) (Call, bool) {
		c := bid(last.Level, st)
		if !c.higherThan(last) {
			c = bid(last.Level+1, st)
		}
		return c, e.legal(p.seat, c)
	}
	best, bestMn, found := Call{}, meaning{}, false
	consider := func(c Call, ok bool, mn meaning) {
		if ok && (!found || c.steps() < best.steps()) {
			best, bestMn, found = c, mn, true
		}
	}
	if bestComb > 0 {
		c, ok := raise(partnerSuit.Strain())
		consider(c, ok, m(-1, -1, "préférence", "preference"))
	}
	for s := Spades; s >= Clubs; s-- {
		if p.shownLens[s] >= 5 && p.hand.Len(s) >= 6 {
			c, ok := raise(s.Strain())
			consider(c, ok, m(-1, -1, "répétition de la couleur sixième au plus bas palier", "cheapest repeat of the six-card suit").withLen(s, p.hand.Len(s)))
			break
		}
	}
	// The waiting notrump competes on price only when it is safe (a stopper
	// in every opponent suit); otherwise it stays the very last resort of a
	// forced hand with nothing else to say.
	ntC, ntOK := raise(SNoTrump)
	if ntOK && e.ntSafe(p) {
		consider(ntC, true, m(-1, -1, "enchère d'attente à Sans-Atout", "waiting notrump bid"))
	}
	if found {
		return best, bestMn
	}
	if ntOK {
		return ntC, m(-1, -1, "enchère d'attente à Sans-Atout", "waiting notrump bid")
	}
	return passCall, noInfo()
}

// ---------- Checkback Stayman (docs/addon_2.md) ----------

// checkbackMajors returns responder's major (his one-level response) and the
// other major, or ok=false when the first bid was not a one-level major.
func (e *Engine) checkbackMajors(seat int) (myM, otherM Suit, ok bool) {
	fb, has := e.firstBidBy(seat)
	if !has || fb.Level != 1 || (fb.Strain != SHearts && fb.Strain != SSpades) {
		return 0, 0, false
	}
	myM = Suit(fb.Strain)
	otherM = Hearts
	if myM == Hearts {
		otherM = Spades
	}
	return myM, otherM, true
}

// checkbackAsk bids 3C over the jump 2NT rebid when responder holds a
// five-card major or four cards in the other major (game is on facing 18-19,
// the ask locates the right fit first).
func (e *Engine) checkbackAsk(p *playerState) (Call, meaning, bool) {
	myM, otherM, ok := e.checkbackMajors(p.seat)
	if !ok {
		return Call{}, meaning{}, false
	}
	partner := e.ps[partnerOf(p.seat)]
	if p.hand.HL()+partner.shownMin >= 33 { // slam zone: keep the generic road
		return Call{}, meaning{}, false
	}
	if p.hand.Len(myM) < 5 && p.hand.Len(otherM) < 4 {
		return Call{}, meaning{}, false
	}
	c := bid(3, SClubs)
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	p.planned = func() (Call, meaning) { return e.afterCheckback(p, myM, otherM) }
	mn := m(-1, -1,
		"3T Checkback : demande 3 cartes dans ma majeure et la quatrième dans l'autre, forcing",
		"3C checkback: asking for three-card support and the other four-card major, forcing").asForcing()
	mn.checkback = true
	return c, mn, true
}

// checkbackAnswer describes opener's majors over the 3C checkback.
func (e *Engine) checkbackAnswer(p *playerState) (Call, meaning) {
	myM, otherM, ok := e.checkbackMajors(partnerOf(p.seat))
	if !ok {
		return passCall, noInfo()
	}
	has3 := p.hand.Len(myM) >= 3
	has4 := p.hand.Len(otherM) >= 4
	var c Call
	var mn meaning
	switch {
	case has3 && has4:
		c = bid(3, SDiamonds)
		mn = m(-1, -1, "3K : 3 cartes dans votre majeure et 4 cartes dans l'autre", "3D: three-card support and four cards in the other major").withLen(myM, 3).withLen(otherM, 4).asForcing()
	case has3:
		c = bidSuit(3, myM)
		mn = m(-1, -1, "3 cartes dans votre majeure, sans 4 cartes dans l'autre", "three-card support, no four cards in the other major").withLen(myM, 3).asForcing()
	case has4:
		c = bidSuit(3, otherM)
		mn = m(-1, -1, "4 cartes dans l'autre majeure, sans 3 cartes dans la vôtre", "four cards in the other major, no three-card support").withLen(otherM, 4).asForcing()
	default:
		c = bid(3, SNoTrump)
		mn = m(-1, -1, "ni 3 cartes dans votre majeure, ni 4 cartes dans l'autre", "neither three-card support nor the other four-card major")
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	return c, mn
}

// afterCheckback places the contract once opener has answered the checkback.
func (e *Engine) afterCheckback(p *playerState, myM, otherM Suit) (Call, meaning) {
	op := e.ps[partnerOf(p.seat)]
	last, _, _ := e.lastBid()
	if op.shownLens[myM] >= 3 && p.hand.Len(myM) >= 5 {
		c := bidSuit(4, myM)
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "conclusion à la manche dans le fit 5-3", "game in the 5-3 major fit").withLen(myM, 5)
		}
	}
	if op.shownLens[otherM] >= 4 && p.hand.Len(otherM) >= 4 {
		c := bidSuit(4, otherM)
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "conclusion à la manche dans le fit 4-4", "game in the 4-4 major fit").withLen(otherM, 4)
		}
	}
	c := bid(3, SNoTrump)
	if last == c || !e.legal(p.seat, c) {
		return passCall, m(-1, -1, "pas de fit majeur", "no major-suit fit")
	}
	return c, m(-1, -1, "pas de fit majeur, conclusion à 3SA", "no major-suit fit, 3NT to play")
}

// ---------- Roudi (docs/addon_9.md) ----------

// roudiAsk bids 2C over opener's plain 1NT rebid when responder holds exactly
// five cards in the major he answered and at least 11H: game (or slam) is in
// sight and the convention locates the 5-3 fit and opener's exact zone before
// choosing between the major and notrump.
func (e *Engine) roudiAsk(p *playerState) (Call, meaning, bool) {
	myM, _, ok := e.checkbackMajors(p.seat)
	if !ok {
		return Call{}, meaning{}, false
	}
	if p.hand.Len(myM) != 5 || p.hand.H() < 11 {
		return Call{}, meaning{}, false
	}
	partner := e.ps[partnerOf(p.seat)]
	if p.hand.HL()+partner.shownMin >= 33 { // slam zone: keep the generic road
		return Call{}, meaning{}, false
	}
	c := bid(2, SClubs)
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	p.planned = func() (Call, meaning) { return e.afterRoudi(p, myM) }
	mn := m(11, 40,
		"2T Roudi : 5 cartes dans ma majeure, 11H et plus, demande le fit et la zone (alerte)",
		"2C Roudi: five cards in my major, 11+, asking for the fit and the exact zone (alertable)").withLen(myM, 5).asForcing()
	mn.roudi = true
	return c, mn, true
}

// roudiAnswer describes opener's hand over the 2C Roudi on the four-step
// scale: 2D = two cards in responder's major and a minimum (12H), 2H = three
// cards and a minimum, 2S = three cards and a maximum (13-14H), 2NT = two
// cards and a maximum.
func (e *Engine) roudiAnswer(p *playerState) (Call, meaning) {
	myM, _, ok := e.checkbackMajors(partnerOf(p.seat))
	if !ok {
		return passCall, noInfo()
	}
	fit3 := p.hand.Len(myM) >= 3
	weak := p.hand.H() <= 12
	var c Call
	var mn meaning
	switch {
	case !fit3:
		// Three steps, not four: without the third card the answer is 2D
		// whatever the strength. Splitting it would spend a step on a
		// distinction that no longer decides anything -- there is no fit to
		// raise, and the opener's zone is still the 12-14 of his 1NT rebid,
		// which the responder can invite into.
		c = bid(2, SDiamonds)
		mn = m(-1, -1, "2K Roudi : 2 cartes dans votre majeure, minimum ou maximum", "2D Roudi: two cards in your major, minimum or maximum")
	case weak:
		c = bid(2, SHearts)
		mn = m(-1, 12, "2C Roudi : 3 cartes dans votre majeure et jeu faible (12H)", "2H Roudi: three-card support, minimum (12)").withLen(myM, 3)
	default:
		c = bid(2, SSpades)
		mn = m(13, -1, "2P Roudi : 3 cartes dans votre majeure et jeu fort (13-14H)", "2S Roudi: three-card support, maximum (13-14)").withLen(myM, 3)
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	return c, mn
}

// afterRoudi places the contract once opener has answered the Roudi: a
// minimum responder (11H) facing the weak answers stops at the two level;
// otherwise game is bid, four of the major on the 5-3 fit, 3NT without it.
func (e *Engine) afterRoudi(p *playerState, myM Suit) (Call, meaning) {
	op := e.ps[partnerOf(p.seat)]
	last, _, _ := e.lastBid()
	fit := op.shownLens[myM] >= 3
	strongOpener := op.shownMin >= 13
	// The 2D answer denies the fit without disclosing the zone: 11 H facing an
	// opener still worth 12 to 14 is an invitation, not a sign-off. 2NT lets
	// him pass on the minimum and bid the game on the maximum.
	if !fit && !strongOpener && op.shownMax >= 13 && p.hand.H() <= 11 {
		c := bid(2, SNoTrump)
		if c.higherThan(last) && e.legal(p.seat, c) {
			return c, m(11, 11, "2SA : proposition, l'ouvreur n'a pas dévoilé sa zone", "2NT: invitation, opener has not disclosed his range").asInvite()
		}
	}
	if p.hand.H() <= 11 && !strongOpener {
		c := bidSuit(2, myM)
		if last == c {
			return passCall, m(11, 11, "ouvreur minimum, arrêt au palier de 2", "minimum opener, stopping at the two level")
		}
		if c.higherThan(last) && e.legal(p.seat, c) {
			return c, m(11, 11, "arrêt : répétition de la majeure au minimum", "stopping: repeats the major at the lowest level").withLen(myM, 5)
		}
		return passCall, noInfo()
	}
	if fit {
		c := bidSuit(4, myM)
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "conclusion à la manche dans le fit majeur 5-3", "game in the 5-3 major fit").withLen(myM, 5)
		}
	}
	c := bid(3, SNoTrump)
	if e.legal(p.seat, c) {
		return c, m(-1, -1, "pas de fit majeur troisième, conclusion à 3SA", "no three-card fit, 3NT to play")
	}
	return passCall, noInfo()
}

// ---------- quatrième couleur forcing (docs/regles_moteur.md §8.8) ----------

// fourthSuitCall locates the auction's fourth suit and the call that names
// it. The convention lives in the quiet auctions where our side has shown
// three suits and neither hand has found a fit: nobody can want to play the
// suit nobody bid, so naming it is free to carry a question instead.
//
// Once our side has bid notrump the auction has its own tools -- Checkback
// [§8.2] and Roudi [§8.3] -- and the fourth suit is not one of them.
//
// A fourth suit still reachable at the one level (spades, and only spades)
// stays natural: holding four of them we would simply bid them. The
// convention then moves up to the "impossible" 2S, which no natural hand
// would ever choose over the cheaper one-level call.
func (e *Engine) fourthSuitCall(p *playerState) (Call, Suit, bool) {
	partner := e.ps[partnerOf(p.seat)]
	if !e.uncontested(p.seat) || p.bids == 0 || partner.bids == 0 {
		return Call{}, 0, false
	}
	// The convention reads three natural suits and deduces the fourth. It
	// therefore has no place in an auction built on artificial calls -- a
	// strong 2C opening and its relays, a Drury, a checkback -- where a
	// named suit is no evidence of length: there the fourth suit is not the
	// one nobody holds, and the deduction is simply false.
	if e.openCall.Level != 1 || e.openCall.Strain > SSpades {
		return Call{}, 0, false
	}
	var named [4]bool
	n := 0
	for _, sc := range e.calls {
		if sideOf(sc.Seat) != sideOf(p.seat) || !sc.Call.IsBid() {
			continue
		}
		if sc.Call.Strain == SNoTrump {
			return Call{}, 0, false
		}
		s := Suit(sc.Call.Strain)
		if sc.M.lens[s] == 0 {
			return Call{}, 0, false
		}
		if !named[s] {
			named[s], n = true, n+1
		}
	}
	if n != 3 {
		return Call{}, 0, false
	}
	f := Clubs
	for s := Clubs; s <= Spades; s++ {
		if !named[s] {
			f = s
		}
	}
	c := e.cheapestCall(f.Strain())
	if c.Level == 1 {
		c = bid(2, f.Strain())
	}
	if c.Level > 3 || !e.legal(p.seat, c) {
		return Call{}, 0, false
	}
	return c, f, true
}

// fourthSuitAsk bids the fourth suit, artificially, as a request for more
// information: three-card support for our own major, a stopper in the named
// suit to play notrump, or failing both a fuller picture of partner's shape.
// It needs 10 H and no fit; it is forcing one round over a cheap two-suiter,
// and forcing to game when partner's own rebid was already forcing (the
// "bicolore cher"), where the side is committed either way.
//
// The ask is only made when there is a real question to put. Five cards in
// our major look for the three-card support partner could not show over a
// one-level response; no stopper in the fourth suit means notrump needs his.
// With neither, the hand describes itself better than a relay would.
func (e *Engine) fourthSuitAsk(p *playerState) (Call, meaning, bool) {
	c, f, ok := e.fourthSuitCall(p)
	if !ok {
		return Call{}, meaning{}, false
	}
	if _, hasFit := e.fitSuit(p); hasFit || p.hand.H() < 10 {
		return Call{}, meaning{}, false
	}
	fb, has := e.firstBidBy(p.seat)
	if !has || fb.Strain > SSpades {
		return Call{}, meaning{}, false
	}
	// Exactly five cards in our major: the length that needs partner's third
	// one, and that a one-level response could not show. A sixth card makes
	// the suit playable on its own -- repeating it says more than a relay.
	my := Suit(fb.Strain)
	if !(my.IsMajor() && p.hand.Len(my) == 5) && p.hand.Stopper(f) {
		return Call{}, meaning{}, false
	}
	fr := "quatrième couleur forcing : 10H et plus, ni fit ni enchère naturelle, demande de renseignement (alerte)"
	en := "fourth suit forcing: 10+ HCP, no fit and no natural bid, asking for a description (alert)"
	partner := e.ps[partnerOf(p.seat)]
	if partner.lastM != nil && partner.lastM.forcing {
		e.gameForce[sideOf(p.seat)] = true
		fr = "quatrième couleur forcing sur bicolore cher : forcing de manche, demande de renseignement (alerte)"
		en = "fourth suit forcing over a reverse: game forcing, asking for a description (alert)"
	}
	mn := m(10, 40, fr, en).asForcing()
	mn.fourthSuit = true
	mn.fourthSuitSuit = f
	return c, mn, true
}

// fourthSuitAnswer describes the hand over partner's fourth suit forcing, in
// the convention's own order: the three-card support for his major first,
// then the one shape that raises the fourth suit itself, then the stopper
// that lets notrump be played, and -- holding none of them -- whatever shape
// is left to show.
func (e *Engine) fourthSuitAnswer(p *playerState, f Suit) (Call, meaning) {
	h := p.hand
	fb, has := e.firstBidBy(partnerOf(p.seat))
	if !has || fb.Strain > SSpades {
		return passCall, noInfo()
	}
	his := Suit(fb.Strain)

	// 1. Three-card support for partner's major, the first thing he asked
	// for. The stronger hand jumps a level, as it would over any response.
	// A minor first suit is not raised here: there the ask is about the
	// stopper, and a return to a suit is reserved for the hands that cannot
	// play notrump at all (rule 4 below).
	if his.IsMajor() && h.Len(his) >= 3 {
		c := e.cheapestCall(his.Strain())
		if h.HLD(his) >= 17 {
			if up := bid(c.Level+1, his.Strain()); !isGame(up) && e.legal(p.seat, up) {
				return up, m(17, 19,
					"3 cartes dans votre majeure, 17-19HLD",
					"three-card support, 17-19 HLD").withLen(his, 3)
			}
		}
		if e.legal(p.seat, c) {
			return c, m(-1, 16,
				"3 cartes dans votre majeure, minimum",
				"three-card support, minimum").withLen(his, 3)
		}
	}

	// 2. The conventional raise of the fourth suit: 5-4-3-1 with the ace
	// third in the suit partner named. The ace is a stopper, but the
	// singleton makes notrump the wrong contract to offer -- so the hand
	// says what it really holds and raises the suit nobody bid, which this
	// one holding makes playable.
	if h.Len(f) == 3 && h.HasCard(f, 'A') && h.sortedLens()[3] == 1 {
		if c := e.cheapestCall(f.Strain()); c.Level <= 3 && e.legal(p.seat, c) {
			// Conventional, not a proposal to play: the ask may have been
			// void of the suit. The raise stays forcing so the auction can
			// never die in a strain neither hand ever claimed.
			return c, m(-1, -1,
				"soutien conventionnel de la quatrième couleur : l'As troisième et un singleton",
				"conventional raise of the fourth suit: ace third and a singleton").withLen(f, 3).asForcing()
		}
	}

	// 3. The stopper that answers the real question of every fourth suit
	// with a minor at its origin: notrump is playable, and at which zone.
	if h.Stopper(f) {
		strong := h.H() >= 15
		c := e.cheapestCall(SNoTrump)
		if strong && c.Level < 3 {
			c = bid(3, SNoTrump)
		}
		if c.Level <= 3 && e.legal(p.seat, c) {
			// The zone is the hand's, never the level's: when the auction
			// has already climbed, the minimum lands on the same 3NT the
			// maximum would have jumped to, and must not claim its points.
			if strong {
				return c, m(15, -1,
					"arrêt à "+suitNameFR[f]+", 15H et plus",
					"stopper in "+suitNameEN[f]+", 15+ HCP").withStopper(f)
			}
			return c, m(-1, 14,
				"arrêt à "+suitNameFR[f]+", minimum",
				"stopper in "+suitNameEN[f]+", minimum").withStopper(f)
		}
	}

	// 4. Neither the support nor the stopper: describe the shape instead --
	// the second suit again when the two are 5-5, the first one again when
	// it is six cards long.
	// The two branches split on the first suit: five cards there and five in
	// the second is the 5-5, six is the long suit. Testing "at least five"
	// made the 6-5 answer the first branch, which announces 5-5 and repeats
	// the second suit -- the sixth card of the opening suit, the one that
	// turns a possible 6-2 into a real trump suit, was neither bid nor
	// recorded, and a nine-card fit went unfound.
	mine := e.suitsBidBy(p.seat)
	if len(mine) >= 2 && h.Len(mine[0]) == 5 && h.Len(mine[1]) >= 5 {
		if c := e.cheapestCall(mine[1].Strain()); c.Level <= 3 && e.legal(p.seat, c) {
			return c, m(-1, -1, "bicolore 5-5, sans arrêt à "+suitNameFR[f],
				"5-5 two-suiter, no stopper in "+suitNameEN[f]).withLen(mine[1], 5).withLen(mine[0], 5)
		}
	}
	if len(mine) >= 1 && h.Len(mine[0]) >= 6 {
		if c := e.cheapestCall(mine[0].Strain()); c.Level <= 3 && e.legal(p.seat, c) {
			return c, m(-1, -1, "couleur sixième, sans arrêt à "+suitNameFR[f],
				"six-card suit, no stopper in "+suitNameEN[f]).withLen(mine[0], 6)
		}
	}
	if len(mine) >= 1 {
		if c := e.cheapestCall(mine[0].Strain()); c.Level <= 3 && e.legal(p.seat, c) {
			return c, m(-1, -1, "répétition de la première couleur, sans arrêt à "+suitNameFR[f],
				"repeats the first suit, no stopper in "+suitNameEN[f])
		}
	}
	return passCall, noInfo()
}

// suitsBidBy lists the suits a seat has named, in bidding order.
func (e *Engine) suitsBidBy(seat int) []Suit {
	var seen [4]bool
	var suits []Suit
	for _, sc := range e.calls {
		if sc.Seat != seat || !sc.Call.IsBid() || sc.Call.Strain > SSpades {
			continue
		}
		if s := Suit(sc.Call.Strain); !seen[s] {
			seen[s] = true
			suits = append(suits, s)
		}
	}
	return suits
}

// reservedByOffer reports whether c is the call partner's rebid has just put
// at the service of a convention: the 3C Checkback over the jump 2NT rebid
// [§8.2], the 2C Roudi over the plain 1NT one [§8.3]. Both rebids announce
// the call themselves -- they are alerted on it -- so partner will read it as
// the question whatever else we meant by it. No other bid may borrow it: a
// control bid, in particular, would start the slam exchange on a call that
// asks about majors, and be answered as such.
func (e *Engine) reservedByOffer(p *playerState, c Call) bool {
	partner := e.ps[partnerOf(p.seat)]
	pm := partner.lastM
	last, lastSeat, ok := e.lastBid()
	if pm == nil || !ok || lastSeat != partner.seat {
		return false
	}
	switch {
	case pm.checkbackOffer && last == bid(2, SNoTrump):
		return c == bid(3, SClubs)
	case pm.roudiOffer && last == bid(1, SNoTrump):
		return c == bid(2, SClubs)
	}
	return false
}

// ---------- troisième couleur forcing (docs/regles_moteur.md §8.9) ----------

// thirdSuitCall locates the auction's third suit and the call that names it.
// The door is far narrower than the fourth suit's [§8.8]: the opening is a
// minor, the response a major at the one level, and opener's rebid the bare
// repeat of that minor at the two level -- a rebid that in one call denies
// three cards in the major, a second suit and any extra values, and leaves
// the responder alone with the two questions the convention puts.
//
// The third suit is always the "collante", the suit immediately above the
// opening minor: diamonds after 1C, hearts after 1D. Spades are never one --
// holding four of them the responder would simply have named them at the one
// level, so a spade bid here keeps its natural meaning.
func (e *Engine) thirdSuitCall(p *playerState) (Call, Suit, bool) {
	if !e.uncontested(p.seat) || p.seat == e.opener || sideOf(p.seat) != sideOf(e.opener) {
		return Call{}, 0, false
	}
	if e.openCall.Level != 1 || e.openCall.Strain > SDiamonds {
		return Call{}, 0, false
	}
	minor := Suit(e.openCall.Strain)
	// Exactly the three bids of 1m - 1M - 2m, each of them promising the
	// suit it names: like the fourth suit, the convention reads natural
	// lengths and deduces from them, and an artificial call anywhere in the
	// sequence would make the deduction false.
	var ours []SeatCall
	for _, sc := range e.calls {
		if sideOf(sc.Seat) == sideOf(p.seat) && sc.Call.IsBid() {
			ours = append(ours, sc)
		}
	}
	if len(ours) != 3 || ours[0].M.lens[minor] == 0 {
		return Call{}, 0, false
	}
	resp, rebid := ours[1], ours[2]
	if resp.Seat != p.seat || resp.Call.Level != 1 || resp.Call.Strain > SSpades {
		return Call{}, 0, false
	}
	my := Suit(resp.Call.Strain)
	if !my.IsMajor() || resp.M.lens[my] == 0 {
		return Call{}, 0, false
	}
	if rebid.Seat != e.opener || rebid.Call != bidSuit(2, minor) || rebid.M.lens[minor] == 0 {
		return Call{}, 0, false
	}
	f := minor + 1
	if f == my {
		// 1D - 1H - 2D: the collante is the very suit already named, and the
		// side has no third suit left below spades.
		return Call{}, 0, false
	}
	c := e.cheapestCall(f.Strain())
	if c.Level != 2 || !e.legal(p.seat, c) {
		return Call{}, 0, false
	}
	return c, f, true
}

// thirdSuitAsk names the third suit, artificially, as the triple question the
// convention puts to opener: three cards in my major -- failing that, a
// stopper in the suit I have just named -- failing both, which contract do
// you want to play? It promises exactly five cards in the major, the length a
// one-level response could not show and the one the ask is made for: a sixth
// card would repeat itself and needs no support.
//
// The ask is forcing to game. Opener's minor rebid has already limited him,
// so the responder who holds the 11 H that face it is committed either way;
// what remains to be found is the strain, not the level.
func (e *Engine) thirdSuitAsk(p *playerState) (Call, meaning, bool) {
	c, f, ok := e.thirdSuitCall(p)
	if !ok {
		return Call{}, meaning{}, false
	}
	// A hand that passed before the opening cannot commit the side to game:
	// its eleven points are the top of a range that starts at nothing, and
	// the convention's whole force -- "we are playing game, only the strain
	// is open" -- would be a claim it is not entitled to make. Such a hand
	// invites instead [G-4].
	if e.passedBeforeOpening(p.seat) {
		return Call{}, meaning{}, false
	}
	// A major fit would have settled the strain and left nothing to ask.
	// The minor fit opener's repeat so often creates is another matter: it
	// is precisely the auction the convention is for -- eleven tricks in a
	// minor is the last contract to settle for while nine at notrump or ten
	// in the major are still to be found.
	if fit, hasFit := e.fitSuit(p); (hasFit && fit.IsMajor()) || p.hand.H() < 11 {
		return Call{}, meaning{}, false
	}
	fb, has := e.firstBidBy(p.seat)
	if !has || fb.Strain > SSpades {
		return Call{}, meaning{}, false
	}
	my := Suit(fb.Strain)
	if p.hand.Len(my) != 5 {
		return Call{}, meaning{}, false
	}
	mn := m(11, 40,
		"troisième couleur forcing : 5 cartes dans ma majeure, 11H et plus, forcing de manche (alerte)",
		"third suit forcing: exactly five cards in my major, 11+ HCP, game forcing (alert)").
		asForcing().withLen(my, 5)
	mn.thirdSuit = true
	mn.thirdSuitSuit = f
	e.gameForce[sideOf(p.seat)] = true
	return c, mn, true
}

// thirdSuitAnswer replies to the ask in the convention's own order: the third
// card in partner's major, which is what the ask is for; then four cards in
// the third suit when it is a major, where a 4-4 fit is still to be found;
// then the stopper that lets notrump be played; then four cards in a minor
// third suit -- "no stopper, but here is the length I hold there"; and
// failing all of it the opening minor, repeated once more.
//
// Every answer is made at the three level. The ask is forcing to game, so
// there is no partscore left below it to protect, and the extra step buys
// nothing.
func (e *Engine) thirdSuitAnswer(p *playerState, f Suit) (Call, meaning) {
	h := p.hand
	fb, has := e.firstBidBy(partnerOf(p.seat))
	if !has || fb.Strain > SSpades {
		return passCall, noInfo()
	}
	his := Suit(fb.Strain)
	answer := func(st Strain) (Call, bool) {
		c := bid(3, st)
		return c, e.legal(p.seat, c)
	}

	// 1. Three cards in the asker's major: the 5-3 fit the whole ask looks
	// for, and the one answer that ends the auction's search on the spot.
	if c, ok := answer(his.Strain()); ok && h.Len(his) >= 3 {
		return c, m(-1, -1,
			"3 cartes dans votre majeure",
			"three-card support for your major").withLen(his, 3)
	}

	// 2. Four cards in a major third suit. The ask promises nothing there,
	// but it denies nothing either: the responder may well hold four hearts
	// alongside his five spades, and the 4-4 fit is worth more than the
	// notrump the stopper would offer.
	if c, ok := answer(f.Strain()); ok && f.IsMajor() && h.Len(f) >= 4 {
		return c, m(-1, -1,
			"pas 3 cartes dans votre majeure, mais 4 cartes à "+suitNameFR[f],
			"no three-card support, but four cards in "+suitNameEN[f]).withLen(f, h.Len(f))
	}

	// 3. The stopper: the second question, and the one that settles the
	// contract the side will most often play.
	if c, ok := answer(SNoTrump); ok && h.Stopper(f) {
		return c, m(-1, -1,
			"pas 3 cartes dans votre majeure, mais l'arrêt à "+suitNameFR[f],
			"no three-card support, but a stopper in "+suitNameEN[f]).withStopper(f)
	}

	// 4. Four cards in a minor third suit, without the stopper: not a fit to
	// play, but the holding the notrump question was about -- partner now
	// knows the guard has to come from his own hand.
	if c, ok := answer(f.Strain()); ok && h.Len(f) >= 4 {
		mn := m(-1, -1,
			"4 cartes à "+suitNameFR[f]+", sans l'arrêt",
			"four cards in "+suitNameEN[f]+", without the stopper").withLen(f, h.Len(f))
		mn.thirdSuitDenied, mn.thirdSuitSuit = true, f
		return c, mn
	}

	// 5. Nothing to offer but the suit already shown twice: no support, no
	// stopper, and the trick source is where it has always been.
	minor := Suit(e.openCall.Strain)
	if c, ok := answer(minor.Strain()); ok {
		mn := m(-1, -1,
			"ni soutien ni arrêt à "+suitNameFR[f]+" : répétition de la couleur d'ouverture",
			"no support and no stopper in "+suitNameEN[f]+": repeats the opening suit").withLen(minor, h.Len(minor))
		mn.thirdSuitDenied, mn.thirdSuitSuit = true, f
		return c, mn
	}
	return passCall, noInfo()
}

// ---------- control bids (docs/addon_4.md) ----------

// gameOfTrump is the game bid in the agreed trump suit (4 major, 5 minor).
func gameOfTrump(trump Suit) Call {
	lvl := 4
	if !trump.IsMajor() {
		lvl = 5
	}
	return bidSuit(lvl, trump)
}

// controlKind describes the control held in s, or ok=false when the hand has
// none there. In the trump suit only the ace or the king qualifies: shortness
// in trump is no control at all. First-round controls are the ace and the
// void; second-round controls are the king and the singleton.
func controlKind(h *Hand, s, trump Suit) (fr, en string, bump int, keycard, ok bool) {
	switch {
	case s == trump && h.HasCard(s, 'A'):
		return "enchère de contrôle, l'As d'atout", "control bid, the trump ace", honorValue['A'], true, true
	case s == trump && h.HasCard(s, 'K'):
		return "enchère de contrôle, le Roi d'atout", "control bid, the trump king", honorValue['K'], true, true
	case s == trump:
		return "", "", 0, false, false
	case h.Len(s) == 0:
		return "enchère de contrôle, chicane", "control bid, void", 0, false, true
	case h.HasCard(s, 'A'):
		return "enchère de contrôle, l'As", "control bid, the ace", honorValue['A'], true, true
	case h.Len(s) == 1:
		return "enchère de contrôle, singleton", "control bid, a singleton", 0, false, true
	case h.HasCard(s, 'K'):
		return "enchère de contrôle, le Roi", "control bid, the king", honorValue['K'], false, true
	}
	return "", "", 0, false, false
}

// fitExpressed reports whether the trump fit was actually agreed in the
// bidding -- both hands have promised length there -- rather than merely
// computed from one side's promise plus the other's hidden cards. The control
// mechanism only starts "après l'expression d'un soutien" : before that, the
// hand holding the support must show it first, and a suit bid below the game
// level is still a natural bid or a game try, never a control.
func (e *Engine) fitExpressed(p *playerState, fit Suit) bool {
	return p.shownLens[fit] > 0 && e.ps[partnerOf(p.seat)].shownLens[fit] > 0
}

// trumpAgreed reports whether fit is a trump suit partner could name too --
// what keycard Blackwood needs before it can be asked, since its answers count
// the trump king and the trump queen. Either both hands have shown length in
// it, or it is the last suit our side actually bid, which is what "the agreed
// suit" means at the table. A fit merely computed from the opening's minimum
// promise plus this hand's own length is neither: partner, who never heard the
// suit from us, cannot know what he is answering about.
func (e *Engine) trumpAgreed(p *playerState, fit Suit) bool {
	if e.fitExpressed(p, fit) && !e.trumpLeftOpen(p, fit) {
		return true
	}
	for i := len(e.calls) - 1; i >= 0; i-- {
		sc := e.calls[i]
		if sideOf(sc.Seat) != sideOf(p.seat) || !sc.Call.IsBid() || sc.Call.Strain > SSpades {
			continue
		}
		return Suit(sc.Call.Strain) == fit
	}
	return false
}

// trumpLeftOpen reports that the fit, though expressed, is not the suit the
// side has settled on: our own bidding has offered a second suit of its own
// and partner has never named the fit himself.
//
// Length alone does not choose. An artificial call can promise it without
// proposing anything -- the cue-bid that asks how strong an overcall is [A-6]
// guarantees three cards in that suit, and its subject is the overcall's
// strength, not where to play -- so it cannot answer a question it never
// asked. While two of our suits are still on the table, keycard answers
// counting "the trump queen" would be about a suit nobody chose; partner
// names his preference first, and the ask follows.
func (e *Engine) trumpLeftOpen(p *playerState, fit Suit) bool {
	var named [4]bool
	partnerNamed := false
	for _, sc := range e.calls {
		if sideOf(sc.Seat) != sideOf(p.seat) || !sc.Call.IsBid() || sc.Call.Strain > SSpades {
			continue
		}
		// Only natural calls put a suit on the table as a trump proposal.
		if sc.M.cuebid || sc.M.relay || sc.M.controlBid || sc.M.blackwood || sc.M.hasTexas {
			continue
		}
		s := Suit(sc.Call.Strain)
		named[s] = true
		if sc.Seat == partnerOf(p.seat) && s == fit {
			partnerNamed = true
		}
	}
	if partnerNamed {
		return false
	}
	n := 0
	for _, ok := range named {
		if ok {
			n++
		}
	}
	return n > 1
}

// sideHasCued reports whether the control-bid exchange has already started for
// our side. It matters because the player who *starts* the process is free to
// choose his step, while every later bid follows economic order.
func (e *Engine) sideHasCued(p *playerState) bool {
	partner := e.ps[partnerOf(p.seat)]
	for s := Clubs; s <= Spades; s++ {
		if p.ctrlShown[s] || partner.ctrlShown[s] {
			return true
		}
	}
	return false
}

// urgentControl returns the control it is most urgent to hear about: the
// cheapest side suit where this hand holds nothing and partner has shown
// nothing either. ok is false once every side suit is accounted for.
func (e *Engine) urgentControl(p *playerState, trump Suit) (Suit, bool) {
	partner := e.ps[partnerOf(p.seat)]
	for s := Clubs; s <= Spades; s++ {
		if s == trump || partner.ctrlShown[s] {
			continue
		}
		if _, _, _, _, ok := controlKind(p.hand, s, trump); ok {
			continue
		}
		return s, true
	}
	return 0, false
}

// controlBid names one control below the trump game level.
//
// The player who opens the exchange starts where he likes: he names the
// control immediately *below* the one he urgently needs to hear about, so that
// partner's cheapest answer is precisely that control ("Le joueur qui
// déclenche le processus des contrôles commence au palier de son choix, étape
// à partir de laquelle le principe de l'ordre économique est rétabli et
// incontournable. Tout contrôle sauté est alors dénié.") Controls *below* that
// starting step are therefore NOT denied — only suits skipped from the start
// onward are. Every later bid of the exchange starts from clubs, so economic
// order is mandatory from then on.
//
// The sequence never crosses the trump game level: signing off in the fit
// stays unambiguous.
func (e *Engine) controlBid(p *playerState, trump Suit) (Call, meaning, bool) {
	h := p.hand
	partner := e.ps[partnerOf(p.seat)]
	last, _, _ := e.lastBid()
	game := gameOfTrump(trump)
	// The exchange normally never crosses the trump game level: signing off in
	// the fit must stay unambiguous. The ceiling lifts to the small slam only
	// when the five level is already paid for -- the auction sits strictly
	// above game (a cue exchange under way) with the combined minimum in the
	// slam zone -- or when initiating from the game level itself with a real
	// cushion beyond that zone (e.g. over partner's direct jump to game, which
	// left no room below it). The trump suit is not part of this rotation up
	// there: once the side controls run out, continueControlBid decides
	// whether the fit above game shows the trump honour or stops.
	ceiling := game
	cMin := h.HLD(trump) + partner.shownMin
	if cMin >= 36 || (last.higherThan(game) && cMin >= 33) {
		ceiling = bidSuit(6, trump)
	}
	// Where the rotation starts. The player who opens the exchange names the
	// control immediately below the one he urgently needs, so partner's
	// cheapest answer is exactly that control; everything below the starting
	// step stays unsaid rather than denied. Later bids always restart from
	// clubs, economic order being mandatory from the first step onward.
	start := Clubs
	if !e.sideHasCued(p) {
		if u, ok := e.urgentControl(p, trump); ok {
			for s := u - 1; s >= Clubs; s-- {
				if s == trump {
					continue
				}
				if _, _, _, _, held := controlKind(h, s, trump); held {
					start = s
					break
				}
			}
		}
	}
	var denied [4]bool
	for s := start; s <= Spades; s++ {
		// Suits already cue-bid by either player are settled: move on instead
		// of jumping to repeat the same strain a level higher.
		if p.ctrlShown[s] || p.ctrlDenied[s] || partner.ctrlShown[s] {
			continue
		}
		c := bid(last.Level, s.Strain())
		if !c.higherThan(last) {
			c = bid(last.Level+1, s.Strain())
		}
		// Cueing at or beyond the ceiling would commit the pair past an
		// unconfirmed slam try: skip the suit — without marking it denied —
		// and keep looking, a later suit in the rotation may still cue lower
		// (over 5H the spade cue at 5S sits under the club one at 6C).
		if !ceiling.higherThan(c) || !e.legal(p.seat, c) {
			continue
		}
		aboveGame := !game.higherThan(c)
		if s == trump && aboveGame {
			continue // the fit above game is a sign-off, never a cue
		}
		fr, en, bump, keycard, held := controlKind(h, s, trump)
		if !held {
			denied[s] = true
			continue
		}
		// A singleton or void this hand has already disclosed — a splinter,
		// the Drury singleton answer — is not worth a control bid: the cue
		// would only repeat what partner already knows and steer the auction
		// at the wrong suit. Skip it without denying; the shortness stays on
		// the table. With no other control below game the hand signs off in
		// the fit [S-3b].
		if !keycard && h.Len(s) <= 1 && p.shortShown[s] {
			continue
		}
		mn := m(p.shownMin+bump, -1, fr, en).withLen(trump, h.Len(trump)).asForcing()
		mn.keycardShown = keycard
		if aboveGame && 33-partner.shownMin > mn.minPts {
			// A cue past the game level announces the strength that makes the
			// five level safe: at least the slam zone opposite partner's floor.
			mn.minPts = 33 - partner.shownMin
		}
		// Successive cues each bump the already-bumped floor, which can
		// compound past what the hand is actually worth: a promise is honest
		// only up to the hand's own trump-fit valuation.
		if v := h.HLD(trump); mn.minPts > v {
			mn.minPts = v
		}
		mn.controlBid = true
		mn.controlSuit = s
		mn.deniedCtrl = denied
		return c, mn, true
	}
	return Call{}, meaning{}, false
}

// nextStep is the bid immediately above c on the auction ladder (3S -> 3NT,
// 3NT -> 4C). The "relais contrôle" answers are always this one step.
func nextStep(c Call) Call {
	n := c.steps() + 1
	return Call{Kind: KindBid, Level: n/5 + 1, Strain: Strain(n % 5)}
}

// controlRelayAsk implements the two "relais contrôle" bids available after
// partner's non-forcing three-level raise of a major fit.
//
// The control one needs to hear about may sit below every control one holds,
// leaving no economic bid able to ask for it: over 1S - 3S there is no way to
// ask for clubs, and over 1H - 3H none for clubs nor for spades. Two
// conventional bids fill the gap:
//
//   - 3SA asks for the CLUB control, on either major fit;
//   - 3S asks for the SPADE control, on a heart fit only.
//
// The positive answer is the immediately higher step (3S -> 3SA, 3SA -> 4C).
// Without the control the step is skipped, which denies it, and the hand names
// its own controls in economic order instead (controlRelayAnswer).
func (e *Engine) controlRelayAsk(p *playerState, trump Suit) (Call, meaning, bool) {
	if !trump.IsMajor() || e.sideHasCued(p) {
		return Call{}, meaning{}, false
	}
	partner := e.ps[partnerOf(p.seat)]
	last, lastSeat, ok := e.lastBid()
	// Only over the codified, non-forcing three-level raise: a forcing raise
	// gives 3SA its other meaning (slamDoubtNT).
	if !ok || lastSeat != partner.seat || last != bidSuit(3, trump) ||
		partner.lastM == nil || partner.lastM.forcing {
		return Call{}, meaning{}, false
	}
	unknown := func(s Suit) bool {
		if partner.ctrlShown[s] {
			return false
		}
		_, _, _, _, held := controlKind(p.hand, s, trump)
		return !held
	}
	// Priority goes to the control that no natural cue could ever reach.
	// On a heart fit a spade control can never be shown below 4H (4S is
	// already past the game), so 3S is the only way to ask for it — and it
	// must be asked first, before the exchange climbs. Clubs sit below every
	// other cue, so hearing about them needs 3SA. A missing diamond control
	// needs no convention at all: bidding 4C asks for it.
	var c Call
	var asked Suit
	switch {
	case trump == Hearts && unknown(Spades):
		c, asked = bid(3, SSpades), Spades
	case unknown(Clubs):
		c, asked = bid(3, SNoTrump), Clubs
	default:
		return Call{}, meaning{}, false
	}
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	mn := m(p.shownMin, -1,
		"relais contrôle, demande le contrôle à "+suitNameFR[asked],
		"control relay, asks for the "+suitNameEN[asked]+" control").
		withLen(trump, p.hand.Len(trump)).asForcing()
	mn.ctrlRelay = true
	mn.ctrlRelaySuit = asked
	return c, mn, true
}

// controlRelayAnswer answers a "relais contrôle": the immediately higher step
// with the control asked for, otherwise the cheapest control above it -- and
// skipping the relay step denies the control it asked for.
func (e *Engine) controlRelayAnswer(p *playerState, trump, asked Suit) (Call, meaning) {
	last, _, _ := e.lastBid()
	game := gameOfTrump(trump)
	mk := func(c Call, s Suit, fr, en string, bump int, keycard bool, denied [4]bool) (Call, meaning) {
		mn := m(p.shownMin+bump, -1, fr, en).withLen(trump, p.hand.Len(trump)).asForcing()
		mn.controlBid, mn.controlSuit = true, s
		mn.keycardShown = keycard
		mn.deniedCtrl = denied
		if v := p.hand.HLD(trump); mn.minPts > v {
			mn.minPts = v
		}
		return c, mn
	}
	if fr, en, bump, keycard, held := controlKind(p.hand, asked, trump); held {
		c := nextStep(last)
		if e.legal(p.seat, c) {
			return mk(c, asked, "réponse positive au relais : "+fr, "positive relay answer: "+en, bump, keycard, [4]bool{})
		}
	}
	// No control there: skipping the step denies it. Name the cheapest control
	// above the relay instead, economic order from here on.
	var denied [4]bool
	denied[asked] = true
	for s := Clubs; s <= Spades; s++ {
		if s == asked {
			continue
		}
		c := bid(last.Level, s.Strain())
		if !c.higherThan(last) {
			c = bid(last.Level+1, s.Strain())
		}
		if !game.higherThan(c) || !e.legal(p.seat, c) {
			continue
		}
		fr, en, bump, keycard, held := controlKind(p.hand, s, trump)
		if !held {
			denied[s] = true
			continue
		}
		return mk(c, s, fr, en, bump, keycard, denied)
	}
	// Nothing left to show: sign off in the fit.
	if e.legal(p.seat, game) && game.higherThan(last) {
		mn := m(-1, -1, "pas de contrôle à "+suitNameFR[asked]+", arrêt à la manche",
			"no "+suitNameEN[asked]+" control, signs off in game").withLen(trump, p.hand.Len(trump))
		mn.deniedCtrl = denied
		return game, mn
	}
	return passCall, noInfo()
}

// slamDoubtNT implements the conventional 3SA that answers partner's *forcing*
// three-level major raise: "une main avec laquelle on est intéressé par
// l'exploration du chelem, donc jamais minimale, mais avec un inconvénient,
// une contra-indication dans l'un des critères" that make a slam. It is the
// "oui mais" between the plain yes (a control bid) and the plain no (bidding
// the game).
//
// The document offers a broad version (any contra-indication, regular shape and
// poor trump quality included) and a narrow one, and states a clear preference
// for the narrow reading: "un singleton dans la couleur du partenaire".
// Only that criterion is implemented, deliberately.
//
// Poor trump quality was tried and dropped: the test alone cannot tell a real
// reservation from a hand whose other values drown it. It fires on ♠AK9853
// ♥J643 ♦A ♣A3 — four ragged trumps, but three aces and a six-card side suit —
// where naming a control is plainly the better call. A singleton facing
// partner's own length has no such failure mode: it is a fact about the fit
// that no control bid can express, and it stays wrong however strong the rest
// of the hand is.
func (e *Engine) slamDoubtNT(p *playerState, trump Suit) (Call, meaning, bool) {
	h := p.hand
	partner := e.ps[partnerOf(p.seat)]
	short := false
	for s := Clubs; s <= Spades; s++ {
		if s != trump && partner.shownLens[s] >= 4 && h.Len(s) <= 1 {
			short = true
		}
	}
	if !short {
		return Call{}, meaning{}, false
	}
	c := bid(3, SNoTrump)
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	reason, reasonEN := "singleton dans votre couleur", "a singleton in your suit"
	mn := m(p.shownMin, -1,
		"3SA « oui mais » : intéressé par le chelem, mais "+reason,
		"conventional 3NT: interested in slam, but "+reasonEN).
		withLen(trump, h.Len(trump)).asForcing()
	mn.ctrlSlamDoubt = true
	return c, mn, true
}

// trumpNamedByBoth reports that the control machinery may speak about fit:
// either both hands have promised length in it [S-0], or the exchange is
// already under way, which settled the trump when it opened. A cue in the
// trump suit itself shows a key card, not a length, so it never reaches
// shownLens -- without the second clause the side would lose the thread of
// its own conversation in mid-sentence.
func (e *Engine) trumpNamedByBoth(p *playerState, fit Suit) bool {
	return e.fitExpressed(p, fit) || e.sideHasCued(p)
}

// hldFacingPartner values the hand for play in trump, discounting shortness in
// a suit partner has shown real length in.
//
// HLD pays for shortness because short suits let the trumps work. That premise
// fails exactly where partner is long: a void opposite six clubs is not three
// points of ruffing value, it is three points of waste -- his length is where
// our small trumps would have earned their keep, and the honours he holds
// there are tricks the shortness throws away rather than tricks it protects.
// The same instrument as hldAgainstTheirBidding, pointed at our own side.
func (e *Engine) hldFacingPartner(p *playerState, trump Suit) int {
	v := p.hand.HLD(trump)
	partner := e.ps[partnerOf(p.seat)]
	for s := Clubs; s <= Spades; s++ {
		if s == trump || partner.shownLens[s] < 4 {
			continue
		}
		switch p.hand.Len(s) {
		case 0:
			v -= 3
		case 1:
			v -= 2
		case 2:
			v -= 1
		}
	}
	return v
}

// slamProbeArmed reports the shared gate of the two slam-probe handlers: a fit,
// a combined count in the exploration window, and a slam actually in view --
// "ne démarrer les enchères de contrôle que si on envisage un chelem" [S-1].
// That means either the combined *maximum* reaches the 33-HLD slam zone (a
// wide-range partner -- a takeout double, an overcall -- may still hold slam
// values his floor hides), or the combined *minimum* is already within a king
// of it. A pair whose ceiling is a dead 29, facing a fully limited raise, has
// no slam to explore and the probe only muddies a plain game.
//
// That maximum is the *slam-valued* one. Twelve tricks cannot be made on a
// ruff the count only half believes in, and the points that vanish first are
// the ones HLD grants for shortness facing partner's own long suit -- the
// discount cMinSlam already applies to the minimum. A hand whose stiff sits
// under the suit partner bid reaches the zone on paper and nowhere else:
// `1D - 1H - 1S - 2S` with SAK97 H3 DAKQT962 C9 counts 23 HLD and lands on
// exactly 33 facing the raise's ceiling, but the singleton heart faces
// partner's own hearts and buys nothing. There is no slam to explore and the
// cue exchange only muddies a plain game.
//
// The maximum leniency itself only makes sense for the wide-range partner the
// comment above names -- a takeout double, an overcall -- whose floor really
// might be hiding more. A partner who raised on a narrow, fully codified
// range has already told the whole story: `1C - 1S - 2S` "soutien simple,
// 12-16HLD" is a four-point spread, and a raw ceiling that merely grazes 33
// against it is not a hand partner might still be holding back -- it is the
// single lucky case where he is at the very top of a bid that is, on
// average, well short of it. That is not "un espoir de chelem" [S-1], and
// launching the cue exchange on it wastes four rounds of bidding space to
// relearn what the raise already said.
//
// With a minor fit the bar is higher: any control past the 3NT level condemns
// the pair to five of the minor (eleven tricks) when the same count already
// affords 3NT (nine), so the minimum must be within two points of the zone --
// unless notrump is not a playable game anyway, in which case 5m is the game
// and the probe costs nothing.
//
// Both hands must already have bid: a fit computable from the opening's
// promised minimum plus this hand's own length is not one the side has
// bid to, and a responder's very first call has expressed nothing yet.
func (e *Engine) slamProbeArmed(ctx *concludeCtx) bool {
	partnerWideRange := ctx.partner.shownMax-ctx.partner.shownMin > 4
	return ctx.hasFit && ctx.cMin >= 29 && ctx.cMin < 33 &&
		ctx.p.bids > 0 && ctx.partner.bids > 0 && !e.bw[ctx.side].asked &&
		((ctx.cMaxSlam >= 33 && partnerWideRange) || ctx.cMin >= 30) &&
		(ctx.fit.IsMajor() || !ctx.ntOK || ctx.cMin >= 31)
}

// expressFit names the trump before the controls [S-0]: the cheapest raise of
// a fit this hand knows and partner does not, forcing, so that the control
// exchange can start next round on a suit both partners have shown.
//
// Only under the game. At or above it the raise is no longer a start but the
// conclusion itself, and choosing the contract is the value-based endgame's
// business, not this handler's.
func (e *Engine) expressFit(ctx *concludeCtx) (Call, meaning, bool) {
	p, fit := ctx.p, ctx.fit
	game := 4
	if !fit.IsMajor() {
		game = 5
	}
	c := e.cheapestCall(fit.Strain())
	if c.Level < 3 {
		// Never at the two level: there the raise is indistinguishable from a
		// bare preference, and partner would price the hand as a minimum just
		// as the side is opening a slam probe. Jump to three -- the gate above
		// has already established that a slam is in view.
		c = bid(3, fit.Strain())
	}
	if c.Level >= game || !e.legal(p.seat, c) || e.reservedByOffer(p, c) {
		// A raise that lands on the call partner's rebid has just reserved for
		// a convention -- a checkback, a relay -- says nothing about the trump:
		// he would answer the question he offered. The fit stays unexpressed
		// and the slam probe does not open here.
		return Call{}, meaning{}, false
	}
	return c, m(-1, -1,
		"soutien forcing : l'atout avant les contrôles",
		"forcing raise: the trump before the controls").
		withLen(fit, p.hand.Len(fit)).asForcing(), true
}

// initiateControls opens the slam exploration below the trump game: one of the
// two "relais contrôle" bids when the control that is needed cannot be asked
// for economically, otherwise an ordinary control bid.
func (e *Engine) initiateControls(p *playerState, trump Suit) (Call, meaning, bool) {
	if c, mn, ok := e.controlRelayAsk(p, trump); ok {
		return c, mn, true
	}
	c, mn, ok := e.controlBid(p, trump)
	if ok && !trump.IsMajor() && c.higherThan(bid(4, SNoTrump)) {
		// Opening the exchange above 4SA strands the pair over the ask: the
		// answer to a control past Blackwood can only be another control, and
		// the slam would have to be bid on faith. Ask now instead, while 4SA
		// is still there -- the same preference continueControlBid applies
		// once the exchange is under way. A major fit is left alone: its game
		// sits under 4SA, so a control above the ask there is a deliberate
		// run past it -- the road taken when Blackwood is barred outright.
		return e.askRatherThanCue(p, trump)
	}
	if ok && e.reservedByOffer(p, c) {
		// The first control falls on the call partner's rebid has just
		// reserved for his convention: he would answer the question instead
		// of hearing the control, and the exchange would be built on a
		// misunderstanding. Skipping to the next control is no better -- it
		// would deny the very control we hold -- so the exploration does not
		// start here, and the count places the contract.
		return Call{}, meaning{}, false
	}
	return c, mn, ok
}

// askRatherThanCue launches Blackwood in place of a control the pair cannot
// afford, when the ask is still available and the trump settled.
func (e *Engine) askRatherThanCue(p *playerState, trump Suit) (Call, meaning, bool) {
	side := sideOf(p.seat)
	fourNT := bid(4, SNoTrump)
	last, _, _ := e.lastBid()
	if e.bw[side].asked || p.answeredAces || !e.trumpAgreed(p, trump) ||
		!fourNT.higherThan(last) || !e.legal(p.seat, fourNT) {
		return Call{}, meaning{}, false
	}
	st, mn := e.blackwoodAsk(p, trump, p.hand.HLD(trump))
	e.bw[side] = st
	return fourNT, mn, true
}

// allSideSuitsControlled reports whether every non-trump suit has a first-round
// control accounted for -- shown by either hand during the cue-bid exchange, or
// an ace/shortness held here. When that holds, slam is a question of keycards
// rather than high-card points, so a distributional hand the raw count would
// undervalue can still be worth the ask.
func (e *Engine) allSideSuitsControlled(p *playerState, trump Suit) bool {
	partner := e.ps[partnerOf(p.seat)]
	for s := Clubs; s <= Spades; s++ {
		if s == trump {
			continue
		}
		if p.ctrlShown[s] || partner.ctrlShown[s] {
			continue
		}
		if p.hand.HasCard(s, 'A') || p.hand.Len(s) <= 1 {
			continue
		}
		return false
	}
	return true
}

// continueControlBid answers partner's control bid: show a further control
// if one remains, transition to Blackwood once 4SA is still reachable and
// the combined values are promising, or sign off in the agreed trump suit
// once neither is possible (docs/addon_4.md, "mécanisme des contrôles").
func (e *Engine) continueControlBid(p *playerState, trump Suit) (Call, meaning) {
	side := sideOf(p.seat)
	partner := e.ps[partnerOf(p.seat)]
	own := p.hand.HLD(trump)
	cMin := own + partner.shownMin
	last, lastSeat, _ := e.lastBid()
	fourNT := bid(4, SNoTrump)
	// A strong hand whose partnership has a control in every side suit should
	// ask for keycards even when the point count falls short: the controls, not
	// the raw points, are what a distributional slam turns on. Without this a
	// wide, low partner floor (e.g. a takeout-double answer, 0-7H) keeps cMin
	// under the threshold and the exchange dies in game with slam cold.
	strongAllControls := own >= 20 && e.allSideSuitsControlled(p, trump)
	// A hand that has answered the ace step does not ask: it has already told
	// partner where its aces are, while learning nothing in return. The
	// opener holds the whole picture -- his own hand plus the ace the step
	// named -- so the slam decision, and the 4SA that goes with it, are his.
	// Asking from this side buys an answer whose 0/3 or 1/4 ambiguity nothing
	// in the auction can resolve.
	// What the ask costs depends on the trump suit. In a minor the game is
	// already at the five level, so 4NT and its answers cost nothing. In a
	// major every answer sits above 4M: a keycard check that comes back short
	// strands the pair in five of the major -- eleven tricks for a game bonus
	// -- and there is no way back down. Below the slam zone the answer cannot
	// even change the decision: four keycards need 33 combined to bid the
	// slam [S-7], so only "all five" would do, and that is a bet on partner
	// holding what the auction never promised. The exchange signs off in 4M
	// instead, which is where the controls were heading anyway.
	// With a major the ask is cheap in level but dear in consequence: every
	// answer sits above 4M, so a check that comes back short strands the pair
	// in five of the major. It needs the slam zone.
	//
	// A minor was long treated as the free case -- the game is already at the
	// five level, so 4SA and its answers "cost nothing". That is only half
	// true. The answer ladder runs 5C-5D-5H-5S whatever the trump, so with
	// diamonds agreed the two upper steps sit *above* 5D: a hand that hears
	// "two keycards, no trump queen" and wants to stop has no stop left to
	// bid, and keycardAnswer bumps its own sign-off into the slam it was
	// refusing. Asking with a minor therefore commits the side, and it too
	// must wait for the count that means to bid one [S-1].
	askZone := cMin >= 33
	// The trump must be the one the side has settled on: the keycard answers
	// count its king and its queen, so a suit partner has never chosen leaves
	// them undefined (same rule as the ask that opens the exploration).
	// Below the zone the answer can still settle the slam by itself. Partner
	// has denied every control in a side suit where this hand lacks the ace,
	// so that ace -- exactly one of them -- is the opponents'; every other
	// side suit is controlled here or by the exchange. The keycards left for
	// partner are then the trump ace and king: holding both, the only key
	// missing is the side ace this hand's singleton or king covers, and the
	// slam is bid; short of that the answer stops in five. It is the ask the
	// count misses on fitted, shapely hands (1P - 2T - 2C - 3P - 3SA - 4C with
	// T9543 AQJ72 A7 5 facing AKQJ86 - 84 QJT84, 29 combined, six cold).
	keysDecide := cMin >= 29 && deniedSideAces(p.hand, partner, trump) == 1 &&
		e.allSideSuitsControlled(p, trump)
	blackwoodAvailable := !e.bw[side].asked && !p.answeredAces &&
		(askZone || strongAllControls || keysDecide) && e.trumpAgreed(p, trump) &&
		fourNT.higherThan(last) && e.legal(p.seat, fourNT)

	if c, mn, ok := e.controlBid(p, trump); ok {
		// A control bid that climbs past the trump game (or beyond 4NT) would
		// strand the pair above Blackwood. When the ask is still available and
		// slam is plausible, prefer to ask now rather than cue past it.
		pastGame := !gameOfTrump(trump).higherThan(c)
		if !(blackwoodAvailable && (pastGame || !fourNT.higherThan(c))) {
			return c, mn
		}
	}
	if blackwoodAvailable {
		st, mn := e.blackwoodAsk(p, trump, own)
		e.bw[side] = st
		return fourNT, mn
	}
	// Back to the trump suit at the game level: any bid of the fit below game
	// would now read as a trump-honour control bid. It is not a stop -- the
	// hand has simply run out of controls to show, which is all it claims.
	// Partner may hold everything that is missing and drive on; here the
	// auction only stops if he agrees.
	game := gameOfTrump(trump)
	c := game
	if !c.higherThan(last) {
		c = e.cheapestCall(trump.Strain())
	}
	if e.legal(p.seat, c) {
		// When cue-bids above game have pushed the landing spot to the six
		// level, the exchange has in fact located a control in every side
		// suit: name the slam for what it is.
		if c.Level >= 6 && cMin >= 33 && e.allSideSuitsControlled(p, trump) {
			return c, m(-1, -1, "petit chelem, tous les contrôles réunis", "small slam, every side suit controlled").withLen(trump, p.hand.Len(trump))
		}
		if c == game {
			return c, m(-1, -1, "plus de contrôle à montrer, je nomme la manche", "no further control to show, naming the game").withLen(trump, p.hand.Len(trump))
		}
		// Past the game the fit is no longer "the game", and the trump suit
		// is no longer mute: its ace or king is a control like any other.
		missing, deniedFR, deniedEN := e.missingSideControls(p, trump)
		if lastSeat == partnerOf(p.seat) && last.Strain == trump.Strain() {
			// Partner has just shown the trump control and asked for the one
			// still missing. Not holding it, stop there: the slam would give
			// the defence two quick tricks in that suit.
			fr, en := "rien à ajouter", "nothing to add"
			if deniedFR != "" {
				fr, en = "pas de contrôle à "+deniedFR, "no "+deniedEN+" control"
			}
			lvl := string(rune('0' + last.Level))
			return passCall, m(-1, -1, fr+", arrêt au palier de "+lvl, en+", stopping at the "+lvl+" level")
		}
		if fr, en, bump, keycard, held := controlKind(p.hand, trump, trump); held && c.Level == 5 && cMin >= 33 {
			// The side controls are all shown, the slam is in view, and the
			// trump honour is what is left to say: it invites the slam and
			// hands partner the decision on the suit this hand could not cue.
			if deniedFR != "" {
				fr, en = fr+", pas de contrôle à "+deniedFR, en+", no "+deniedEN+" control"
			}
			mn := m(p.shownMin+bump, -1, fr+" : invitation au chelem", en+": slam try").withLen(trump, p.hand.Len(trump))
			if 33-partner.shownMin > mn.minPts {
				mn.minPts = 33 - partner.shownMin
			}
			if mn.minPts > own {
				mn.minPts = own
			}
			mn.controlBid = true
			mn.controlSuit = trump
			mn.keycardShown = keycard
			mn.deniedCtrl = missing
			return c, mn
		}
		return c, m(-1, -1, "plus de contrôle à montrer, retour à l'atout", "no further control to show, back to the trump suit").withLen(trump, p.hand.Len(trump))
	}
	return passCall, noInfo()
}

// missingSideControls lists the side suits where neither hand has shown a
// control and this hand holds none: the suits a bid of the fit above game
// leaves to partner. The names come joined for the explanation.
func (e *Engine) missingSideControls(p *playerState, trump Suit) (missing [4]bool, fr, en string) {
	partner := e.ps[partnerOf(p.seat)]
	for s := Clubs; s <= Spades; s++ {
		if s == trump || p.ctrlShown[s] || partner.ctrlShown[s] {
			continue
		}
		if _, _, _, _, held := controlKind(p.hand, s, trump); held {
			continue
		}
		missing[s] = true
		if fr != "" {
			fr, en = fr+" ni à ", en+" or "
		}
		fr, en = fr+suitNameFR[s], en+suitNameEN[s]
	}
	return missing, fr, en
}

// ---------- Blackwood ----------

// isKingAsk reports whether opener's 4NT should be read as a king ask rather
// than an ace/keycard ask: after a 2D opening the responder has already
// pinpointed the aces through the ace-step responses, so once every ace is
// accounted for the opener switches to asking kings (docs/bidings.md). The
// aces are "located" only from what the auction disclosed: the ace-step
// response either pinned the exact count, or its floor already completes the
// four aces together with this hand's own -- never from partner's actual
// cards, which the opener cannot see.
func (e *Engine) isKingAsk(p *playerState) bool {
	partner := e.ps[partnerOf(p.seat)]
	if p.seat != e.opener || e.openCall != bid(2, SDiamonds) || !partner.answeredAces {
		return false
	}
	own := p.hand.Aces()
	if partner.acesExact {
		return own+partner.acesShown == 4
	}
	// Only four aces exist, so a floor that already completes them with our
	// own count pins partner's count exactly; anything less stays ambiguous
	// (the 3SA answer covers both "two mixed" and "three or more").
	return own+partner.acesShown >= 4
}

// blackwoodAsk builds the trump-agreed 4NT ask and its recorded state. When it
// is a king ask (isKingAsk), the trump queen stands in for the trump king as
// the fifth key; otherwise it stays an ordinary keycard ask.
func (e *Engine) blackwoodAsk(p *playerState, trump Suit, own int) (bwState, meaning) {
	st := bwState{asked: true, asker: p.seat, trump: trump}
	if e.isKingAsk(p) {
		st.kingAsk = true
		mn := m(own-1, -1,
			"Blackwood 4SA, appel aux Rois (Dame d'atout = 5e clef)",
			"4NT king Blackwood (trump queen is the fifth key)").withLen(trump, p.hand.Len(trump))
		mn.blackwood = true
		mn.forcing = true
		return st, mn
	}
	mn := m(own-1, -1, "Blackwood 4SA, demande des cartes clefs", "4NT keycard Blackwood").withLen(trump, p.hand.Len(trump))
	mn.blackwood = true
	mn.forcing = true
	return st, mn
}

// kingAnswer answers opener's king-ask 4NT: it counts the four kings plus the
// trump queen (the fifth key). The step ladder mirrors keycard Blackwood, but
// the trump queen is already folded into the count, so the two 5H/5S queen
// steps collapse into a single 5H.
func (e *Engine) kingAnswer(p *playerState, trump Suit) (Call, meaning) {
	keys := p.hand.KingKeys(trump)
	side := sideOf(p.seat)
	var c Call
	var fr, en string
	switch keys {
	case 0, 3:
		c, fr, en = bid(5, SClubs), "0 ou 3 clefs (Rois + Dame d'atout)", "0 or 3 king-keys"
		e.bw[side].keyStep = 0
	case 1, 4:
		c, fr, en = bid(5, SDiamonds), "1 ou 4 clefs (Rois + Dame d'atout)", "1 or 4 king-keys"
		e.bw[side].keyStep = 1
	default: // 2 or 5
		c, fr, en = bid(5, SHearts), "2 ou 5 clefs (Rois + Dame d'atout)", "2 or 5 king-keys"
		e.bw[side].keyStep = 2
	}
	mn := m(-1, -1, fr, en)
	mn.keyResp = true
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	return c, mn
}

// afterKings places the contract once responder has shown its king-keys. All
// five keys (the four kings and the trump queen) make the grand slam; short of
// that the aces are known present, so the small slam is safe.
func (e *Engine) afterKings(p *playerState) (Call, meaning) {
	side := sideOf(p.seat)
	trump := e.bw[side].trump
	ownKeys := p.hand.KingKeys(trump)
	// No prior disclosure mechanism exists for king-keys (the ask follows
	// directly on the opening's own ace-step exchange, before any control
	// bidding), so a genuinely ambiguous step -- as opposed to one the
	// asker's own count rules out -- always assumes the least favorable
	// count, exactly as disambiguatePair does with disclosed = 0.
	partnerKeys := 2 // step 2 (2 or 5) still needs disambiguating below
	switch e.bw[side].keyStep {
	case 0:
		partnerKeys = disambiguatePair(0, 3, ownKeys, 0, 5)
	case 1:
		partnerKeys = disambiguatePair(1, 4, ownKeys, 0, 5)
	case 2:
		partnerKeys = disambiguatePair(2, 5, ownKeys, 0, 5)
	}
	total := ownKeys + partnerKeys
	partner := e.ps[partnerOf(p.seat)]
	cMin := p.hand.HLD(trump) + partner.shownMin
	last, _, _ := e.lastBid()

	// All five keys is a laydown grand; a single one missing still bids the
	// grand on overwhelming combined strength (aces are all known present, so
	// the missing king is usually covered by shortness or discards), otherwise
	// the small slam is the limit.
	target := 6
	if total == 5 || (total == 4 && cMin >= 37) {
		target = 7
	}
	c := bidSuit(target, trump)
	for !c.higherThan(last) && target < 7 {
		target++
		c = bidSuit(target, trump)
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	switch {
	case target == 7 && total == 5:
		return c, m(-1, -1, "grand chelem, tous les Rois et la Dame d'atout", "grand slam, all kings and the trump queen")
	case target == 7:
		return c, m(-1, -1, "grand chelem sur la force combinée, une clef compensée", "grand slam on combined strength, one key covered")
	default:
		return c, m(-1, -1, "petit chelem, une clef manquante", "small slam, a key is missing")
	}
}

// kingAnswerNT answers opener's king-ask 4NT made with no trump agreed (after
// a 2D opening): the four aces are already located, so it simply counts kings.
// The answered count is recorded in the Blackwood state for the asker to read
// back -- the ladder is unambiguous (5C..5NT = 0..4 kings).
func (e *Engine) kingAnswerNT(p *playerState) (Call, meaning) {
	kings := p.hand.Kings()
	var c Call
	var fr, en string
	switch kings {
	case 0:
		c, fr, en = bid(5, SClubs), "0 Roi", "0 kings"
	case 1:
		c, fr, en = bid(5, SDiamonds), "1 Roi", "1 king"
	case 2:
		c, fr, en = bid(5, SHearts), "2 Rois", "2 kings"
	case 3:
		c, fr, en = bid(5, SSpades), "3 Rois", "3 kings"
	default:
		c, fr, en = bid(5, SNoTrump), "4 Rois, tous les Rois", "4 kings, all kings held"
	}
	mn := m(-1, -1, fr, en)
	mn.keyResp = true
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	e.bw[sideOf(p.seat)].keyStep = kings
	return c, mn
}

// afterKingsNT concludes a notrump king-ask. With all four kings (the aces
// already known present) the grand slam is on, as it is with three and an
// overwhelming combined count; otherwise the small slam is the limit -- but
// only if the pair actually holds the values for it. The ask is also asked
// from hands whose partner has shown little more than an ace, and a
// disappointing answer must be allowed to stop in 5SA rather than force a
// slam the count never reached.
func (e *Engine) afterKingsNT(p *playerState) (Call, meaning) {
	side := sideOf(p.seat)
	partner := e.ps[partnerOf(p.seat)]
	// Partner's king count comes from the answered step, never from the
	// hidden hand.
	total := p.hand.Kings() + e.bw[side].keyStep
	// Partner's honour floor: the range his bidding already promised, or the
	// aces and kings his conventional answers pinpointed, whichever is more.
	// Taking the larger of the two rather than the sum avoids counting the
	// same king twice, once inside a shown range and once as a keycard.
	floor := partner.shownMin
	if k := 4*partner.acesShown + 3*e.bw[side].keyStep; k > floor {
		floor = k
	}
	cMin := p.hand.H() + floor
	last, _, _ := e.lastBid()

	target := 6
	switch {
	case total == 4 || (total == 3 && cMin >= 37):
		target = 7
	case cMin < 33:
		target = 5
	}
	c := bid(target, SNoTrump)
	for !c.higherThan(last) && target < 7 {
		target++
		c = bid(target, SNoTrump)
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	switch target {
	case 7:
		return c, m(-1, -1, "grand chelem à Sans-Atout, les Rois requis (As déjà connus)", "grand slam in notrump, the required kings held (aces already known)")
	case 5:
		return c, m(-1, -1, "arrêt à 5SA, les Rois manquants laissent la force combinée sous le chelem", "signs off in 5NT, the missing kings leave the pair short of slam")
	}
	return c, m(-1, -1, "petit chelem à Sans-Atout, un Roi manquant", "small slam in notrump, a king missing")
}

func (e *Engine) keycardAnswer(p *playerState, trump Suit) (Call, meaning) {
	keys := p.hand.Keycards(trump)
	hasQ := p.hand.HasCard(trump, 'Q')
	side := sideOf(p.seat)
	var c Call
	var fr, en string
	switch {
	case keys == 0 || keys == 3:
		c, fr, en = bid(5, SClubs), "0 ou 3 cartes clefs", "0 or 3 keycards"
		e.bw[side].keyStep = 0
	case keys == 1 || keys == 4:
		c, fr, en = bid(5, SDiamonds), "1 ou 4 cartes clefs", "1 or 4 keycards"
		e.bw[side].keyStep = 1
	case !hasQ:
		c, fr, en = bid(5, SHearts), "2 cartes clefs sans la Dame d'atout", "2 keycards without the trump queen"
		e.bw[side].keyStep = 2
	default:
		c, fr, en = bid(5, SSpades), "2 cartes clefs et la Dame d'atout", "2 keycards with the trump queen"
		e.bw[side].keyStep = 3
	}
	mn := m(-1, -1, fr, en)
	mn.keyResp = true
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	return c, mn
}

// disambiguatePair resolves a Blackwood step that groups two possible counts
// (lo and hi) into a single bid, using only what a real asker could know:
// their own count, and whatever the partnership already disclosed about
// keycards before the ask (prior control bids). It is lo unless that is
// mathematically impossible (the partnership cannot hold more keys than
// exist, so partner's own count is forced up to hi) or partner has already
// disclosed more keycards than lo allows (a control bid shown before the
// ask). Left genuinely open, it assumes the least favorable count -- never
// the partner's actual hand.
func disambiguatePair(lo, hi, own, partnerDisclosed, maxTotal int) int {
	if own+hi > maxTotal {
		return lo
	}
	if partnerDisclosed > lo {
		return hi
	}
	return lo
}

// keycardHonourCeiling returns the most honour points partner can hold while
// holding no more than keys of the five keycards. Every keycard this hand does
// not hold and partner does not have sits with the opponents, out of partner's
// reach: what is left of the forty points is his ceiling. Partner keeps the
// dearest keycards (an ace is worth four, the trump king three), so the
// ceiling is the most generous reading of the low count.
func keycardHonourCeiling(h *Hand, trump Suit, keys int) int {
	var missing []int // values of the keycards this hand does not hold, aces first
	for s := Clubs; s <= Spades; s++ {
		if !h.HasCard(s, 'A') {
			missing = append(missing, 4)
		}
	}
	if !h.HasCard(trump, 'K') {
		missing = append(missing, 3)
	}
	opponents := 0
	for i := keys; i < len(missing); i++ {
		opponents += missing[i]
	}
	return 40 - h.H() - opponents
}

// deniedSideAces counts the side aces h does not hold in suits where partner
// has denied any control during the exchange: they cannot be his, so they sit
// with the opponents.
func deniedSideAces(h *Hand, partner *playerState, trump Suit) int {
	n := 0
	for s := Clubs; s <= Spades; s++ {
		if s != trump && partner.ctrlDenied[s] && !h.HasCard(s, 'A') {
			n++
		}
	}
	return n
}

func (e *Engine) afterKeycards(p *playerState) (Call, meaning) {
	side := sideOf(p.seat)
	trump := e.bw[side].trump
	partner := e.ps[partnerOf(p.seat)]
	ownKeys := p.hand.Keycards(trump)
	// An ambiguous step is read low [S-6], unless what the asker already
	// knows rules the low count out: his own count, the keycards a control
	// bid disclosed before the ask -- and the floor partner's bidding
	// promised. A hand that answers "zero or three" while its bidding has
	// promised as many points as a zero-keycard hand could possibly hold has
	// answered three: the honours that would have to make up that floor are
	// sitting in the opponents' hands with the keycards. The boundary counts
	// as ruled out, not as doubt -- at equality the low reading needs partner
	// to hold every remaining king, queen and jack and not one keycard, and
	// fearing that hand is asking for keycards with no intention of using
	// the answer.
	resolve := func(lo, hi int) int {
		if k := disambiguatePair(lo, hi, ownKeys, partner.keycardsShown, 5); k != lo {
			return k
		}
		// Never past what the pack holds: when the asker's own count already
		// rules the high reading out, the low one stands whatever the floor.
		if ownKeys+hi <= 5 && partner.shownMin >= keycardHonourCeiling(p.hand, trump, lo) {
			return hi
		}
		return lo
	}
	partnerKeys := 2 // steps 2 (with or without the queen) are exact
	switch e.bw[side].keyStep {
	case 0:
		partnerKeys = resolve(0, 3)
	case 1:
		partnerKeys = resolve(1, 4)
	}
	total := ownKeys + partnerKeys
	own := p.hand.HLD(trump)
	cMin := own + partner.shownMin
	last, _, _ := e.lastBid()

	// All five keycards justify the slam on controls alone. Missing one, it
	// depends which: a side ace is covered when the cue-bid exchange showed
	// that suit controlled (shortness ruffs the loser away); a missing trump
	// ace or king is a trump trick nothing covers, so the combined count must
	// genuinely reach the slam zone — a keycard check entered from the 29-32
	// control window must otherwise stop in five. Both ranks are known
	// accounted for only from disclosed information: this hand holding both,
	// or partner's own control bid having named the trump suit before the ask.
	// A third way to know: partner holds every keycard his control denials
	// leave him (the side aces he has none of excepted), so what is missing is
	// exactly one of those denied side aces.
	denied := deniedSideAces(p.hand, partner, trump)
	missingIsSideAce := (p.hand.HasCard(trump, 'A') && p.hand.HasCard(trump, 'K')) ||
		(partner.ctrlShown[trump] && (p.hand.HasCard(trump, 'A') || p.hand.HasCard(trump, 'K'))) ||
		(denied == 1 && partnerKeys == 5-ownKeys-denied)
	target := 5
	if total >= 5 || (total == 4 && (cMin >= 33 || (missingIsSideAce && e.allSideSuitsControlled(p, trump)))) {
		target = 6
	}
	if total == 5 && cMin >= 37 {
		target = 7
	}
	c := bidSuit(target, trump)
	if target == 5 && !c.higherThan(last) && last.Strain == trump.Strain() {
		// The keycard answer already occupies five of the trump suit:
		// passing it is the only sign-off left — bumping the level would
		// turn the stop into the very slam it declines.
		return passCall, m(-1, -1, "arrêt sur la réponse, cartes clefs insuffisantes", "passes the answer, not enough keycards")
	}
	for !c.higherThan(last) && target < 7 {
		target++
		c = bidSuit(target, trump)
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	switch target {
	case 7:
		return c, m(-1, -1, "grand chelem, toutes les cartes clefs", "grand slam, all keycards held")
	case 6:
		return c, m(-1, -1, "petit chelem", "small slam")
	default:
		return c, m(-1, -1, "arrêt à 5, cartes clefs ou points insuffisants", "signs off at the five level, keycards or strength missing")
	}
}

// keycardAnswerNT answers a Blackwood 4NT ask made with no trump suit
// agreed: plain ace count (0-4), no trump king/queen refinement possible.
// The answered count is recorded in the Blackwood state for the asker to
// read back -- the ladder is unambiguous (5C..5NT = 0..4 aces).
func (e *Engine) keycardAnswerNT(p *playerState) (Call, meaning) {
	aces := p.hand.Aces()
	var c Call
	var fr, en string
	switch aces {
	case 0:
		c, fr, en = bid(5, SClubs), "0 As", "0 aces"
	case 1:
		c, fr, en = bid(5, SDiamonds), "1 As", "1 ace"
	case 2:
		c, fr, en = bid(5, SHearts), "2 As", "2 aces"
	case 3:
		c, fr, en = bid(5, SSpades), "3 As", "3 aces"
	default:
		c, fr, en = bid(5, SNoTrump), "4 As, tous les As", "4 aces, all aces held"
	}
	mn := m(-1, -1, fr, en)
	mn.keyResp = true
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	e.bw[sideOf(p.seat)].keyStep = aces
	return c, mn
}

// afterKeycardsNT concludes a notrump keycard sequence: with no trump suit,
// only the four aces are at stake, so at most one may be missing for a
// small slam, and all four (plus enough combined strength) for a grand one.
func (e *Engine) afterKeycardsNT(p *playerState) (Call, meaning) {
	side := sideOf(p.seat)
	partner := e.ps[partnerOf(p.seat)]
	// Partner's ace count comes from the answered step, never from the
	// hidden hand.
	total := p.hand.Aces() + e.bw[side].keyStep
	own := p.hand.H()
	cMin := own + partner.shownMin
	last, _, _ := e.lastBid()

	target := 5
	if total >= 3 {
		target = 6
	}
	if total == 4 && cMin >= 37 {
		target = 7
	}
	c := bid(target, SNoTrump)
	for !c.higherThan(last) && target < 7 {
		target++
		c = bid(target, SNoTrump)
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	switch target {
	case 7:
		return c, m(-1, -1, "grand chelem à Sans-Atout, tous les As", "grand slam in notrump, all aces held")
	case 6:
		return c, m(-1, -1, "petit chelem à Sans-Atout", "small slam in notrump")
	default:
		return c, m(-1, -1, "arrêt à 5SA, au moins deux As manquants", "signs off at 5NT, at least two aces missing")
	}
}
