package engine

import (
	"fmt"
	"slices"
)

// cheapestCall returns the lowest legal bid in the given strain.
func (e *Engine) cheapestCall(st Strain) Call {
	last, _, ok := e.lastBid()
	if !ok {
		return bid(1, st)
	}
	c := bid(last.Level, st)
	if !c.higherThan(last) {
		c = bid(last.Level+1, st)
	}
	return c
}

// firstBidBy returns the first bid (not pass/double) made by a seat.
func (e *Engine) firstBidBy(seat int) (Call, bool) {
	for _, sc := range e.calls {
		if sc.Seat == seat && sc.Call.IsBid() {
			return sc.Call, true
		}
	}
	return Call{}, false
}

// passedBeforeOpening reports whether seat passed before partner's opening
// bid (docs/addon_6.md, section V): a later jump then carries the capped
// "rencontre" meaning rather than its usual unlimited one, since a passed
// hand cannot hold opening values.
func (e *Engine) passedBeforeOpening(seat int) bool {
	for _, sc := range e.calls {
		if sc.Seat == seat && sc.Call.Kind == KindPass {
			return true
		}
		if sc.Seat == e.opener && sc.Call.IsBid() {
			return false
		}
	}
	return false
}

func (e *Engine) lastCallBy(seat int) (SeatCall, bool) {
	for i := len(e.calls) - 1; i >= 0; i-- {
		if e.calls[i].Seat == seat {
			return e.calls[i], true
		}
	}
	return SeatCall{}, false
}

// doubledSuit returns the suit of the bid that seat's most recent double
// acted on. A double can only follow a bid (never a pass), so the call
// immediately before it in the auction is always that bid -- whether it is
// the original opening or a later reopening/balancing bid the opponents
// made after two passes. Reading the target from e.openCall instead would
// wrongly point back at the original opening in the reopening case.
func (e *Engine) doubledSuit(seat int) (Suit, bool) {
	for i := len(e.calls) - 1; i >= 0; i-- {
		if e.calls[i].Seat != seat || e.calls[i].Call.Kind != KindDouble {
			continue
		}
		// The suit taken out is the one the last *bid* named, which is not
		// always the neighbouring call: a reopening double comes after two
		// passes, and a pass carries the zero Strain -- Clubs -- so reading
		// the neighbour blindly took every balancing double for a takeout of
		// Clubs. Notrump has no suit to take out, and neither has a double
		// nobody has bid in front of.
		for j := i - 1; j >= 0; j-- {
			if !e.calls[j].Call.IsBid() {
				continue
			}
			if e.calls[j].Call.Strain > SSpades {
				return 0, false
			}
			return Suit(e.calls[j].Call.Strain), true
		}
		return 0, false
	}
	return 0, false
}

// interferenceAfterOpening reports whether a defender acted over the opening.
func (e *Engine) interferenceAfterOpening() bool {
	seen := false
	for _, sc := range e.calls {
		if seen && sideOf(sc.Seat) != sideOf(e.opener) && sc.Call.Kind != KindPass {
			return true
		}
		if sc.Seat == e.opener && sc.Call.IsBid() {
			seen = true
		}
	}
	return false
}

// ---------- opening ----------

func (e *Engine) opening(p *playerState) (Call, meaning) {
	h := p.hand
	hl, hp := h.HL(), h.H()
	tr := e.tr
	balanced := h.IsRegular() || h.IsSemiRegular()

	if tr.check(hl >= 24,
		"24 HL et plus → 2♦ forcing de manche [O-1]",
		"24+ HL → 2♦ game forcing [O-1]", pts(hl, "HL")) {
		e.gameForce[sideOf(p.seat)] = true
		mn := m(24, 40, "ouverture 2K forcing de manche, 24HL et plus", "2D opening, game forcing, 24+ HL").asForcing()
		return bid(2, SDiamonds), mn
	}
	if tr.check(hp >= 18 && hp <= 23 && (h.sortedLens()[0] >= 6 || hp >= 21) && !(hp <= 21 && balanced && hp >= 20),
		"18-23 H avec une sixième ou 21 H et plus, hors 20-21 H régulière → 2♣ fort [O-2]",
		"18-23 H with a six-card suit or 21+ H, not a balanced 20-21 → strong 2♣ [O-2]",
		pts(hp, "H")+", "+shape(h)) {
		return bid(2, SClubs), m(18, 23, "ouverture 2T fort indéterminé", "strong artificial 2C opening").asForcing()
	}
	if tr.check(hp >= 20 && hp <= 21 && balanced,
		"20-21 H, régulière ou semi-régulière → 2SA [O-3]",
		"20-21 H, balanced or semi-balanced → 2NT [O-3]", pts(hp, "H")+", "+shape(h)) {
		return bid(2, SNoTrump), m(20, 21, "ouverture 2SA, 20-21H régulier", "2NT opening, 20-21 balanced")
	}
	if tr.check(hp >= 15 && hp <= 17 && h.IsRegular(),
		"15-17 H, régulière → 1SA [O-4]",
		"15-17 H, balanced → 1NT [O-4]", pts(hp, "H")+", "+shape(h)) {
		return bid(1, SNoTrump), m(15, 17, "ouverture 1SA, 15-17H régulier", "1NT opening, 15-17 balanced")
	}
	// Solid seven-card minor, no outside strength (docs/bidings.md,
	// "L'OUVERTURE DE 3SA"): checked before the generic point-count openings
	// below, since a hand this concentrated can otherwise reach hp>=12 or
	// hl>=13 and be misread as an ordinary one-level opening, or fall into
	// the plain 7-card barrage at hp<=10 -- both hide the far more precise
	// description this shape affords.
	if _, ok := sevenCardSolidMinor(h); tr.check(ok,
		"mineure septième ARD, sans As ni Roi à côté → 3SA [O-5]",
		"solid seven-card minor (AKQ), no outside ace or king → 3NT [O-5]", "") {
		return bid(3, SNoTrump), m(9, 12, "ouverture de 3SA, mineure septième affranchie (AKQ et plus), sans force annexe", "3NT opening, solid seven-card minor (AKQ+), no outside strength")
	}
	if tr.check(hp >= 12 || hl >= 13,
		"12 H ou 13 HL → ouverture d'1 à la couleur [O-6]",
		"12 H or 13 HL → one-level suit opening [O-6]", pts(hp, "H")+", "+pts(hl, "HL")) {
		tr.in()
		s := openingSuitT(h, tr)
		tr.out()
		min := 5
		fr, en := "ouverture majeure, 5 cartes et plus, 12-23HL", "major-suit opening, 5+ cards, 12-23 HL"
		if !s.IsMajor() {
			min = 3
			fr, en = "ouverture mineure, 3 cartes et plus, 12-23HL", "minor-suit opening, 3+ cards, 12-23 HL"
		}
		return bidSuit(1, s), m(12, 23, fr, en).withLen(s, min)
	}
	if tr.check(hp >= 5 && hp <= 11,
		"5 à 11 H → barrage possible [O-7]",
		"5 to 11 H → a preempt is possible [O-7]", pts(hp, "H")) {
		tr.in()
		c, mn, ok := e.preemptOpening(h)
		tr.out()
		if ok {
			return c, mn
		}
	}
	tr.in()
	c, mn, ok := e.lightOpening(p)
	tr.out()
	if ok {
		p.lightOpen = true
		return c, mn
	}
	tr.note("aucune ouverture ne s'applique → Passe [O-8]", "no opening applies → Pass [O-8]")
	return passCall, m(0, 11, "", "")
}

// lightOpening covers the third- and fourth-seat openings that sit below the
// [O-6] threshold (docs/regles_moteur.md [O-7b]). Partner has already passed,
// so there is no game to be found and none to be missed: what is left to win
// is the lead direction of a real suit, and the level the opponents -- the
// side that holds the values -- have to start from.
//
// Third seat opens for the lead: a five-card suit worth leading, with the
// honour strength inside it rather than scattered over the hand
// (SuitH*2 >= H, the same concentration the preempts demand [O-7]). A hand
// whose points sit outside its long suit misdirects the lead it was meant to
// direct, and defends better than it opens.
//
// Fourth seat has no lead to direct -- passing the hand out scores nothing,
// which beats scoring minus. It opens on the rule of 15 only: honour points
// plus the spade length, spades being what decides who buys the partscore at
// the one level.
//
// The zone announced is 10-11: by construction [O-6] has already taken every
// hand of 12 H or 13 HL, so the ceiling is real and partner, himself capped
// by his opening pass [E-1b], reads a combined maximum that never invites.
func (e *Engine) lightOpening(p *playerState) (Call, meaning, bool) {
	// opening() only ever runs while nobody has opened, so every call made
	// so far is a pass and their number is the seat: 2 = third, 3 = fourth.
	seat := len(e.calls)
	h := p.hand
	hp := h.H()
	tr := e.tr
	if seat < 2 {
		return Call{}, meaning{}, false
	}
	if !tr.check(hp >= 10,
		"10 H et plus, en troisième ou quatrième position → ouverture légère [O-7b]",
		"10+ H, in third or fourth seat → light opening [O-7b]", pts(hp, "H")) {
		return Call{}, meaning{}, false
	}
	s := openingSuit(h)
	switch seat {
	case 2:
		if !tr.check(h.Len(s) >= 5 && h.GoodSuit(s) && h.SuitH(s)*2 >= hp,
			"troisième : belle couleur cinquième qui porte la moitié des points → 1 à la couleur",
			"third seat: a good five-card suit holding half the points → one of the suit",
			cards(h, s)) {
			return Call{}, meaning{}, false
		}
		mn := m(10, 11, "ouverture légère de troisième, belle couleur cinquième, 10-11H, indication d'entame", "light third-seat opening, good five-card suit, 10-11 H, lead-directing")
		return bidSuit(1, s), mn.withLen(s, 5), true
	case 3:
		if !tr.check(hp+h.Len(Spades) >= 15,
			"quatrième : règle des 15 (H + nombre de ♠ ≥ 15)",
			"fourth seat: rule of 15 (H + number of ♠ ≥ 15)",
			fmt.Sprintf("%d + %d = %d", hp, h.Len(Spades), hp+h.Len(Spades))) {
			return Call{}, meaning{}, false
		}
		min := 5
		if !s.IsMajor() {
			min = 3
		}
		mn := m(10, 11, "ouverture légère de quatrième, règle des 15, 10-11H", "light fourth-seat opening, rule of 15, 10-11 H")
		return bidSuit(1, s), mn.withLen(s, min), true
	}
	return Call{}, meaning{}, false
}

func openingSuit(h *Hand) Suit { return openingSuitT(h, nil) }

// openingSuitT is openingSuit with its tests recorded in tr (nil: untraced).
func openingSuitT(h *Hand, tr *tracer) Suit {
	lc, ld, lh, ls := h.Len(Clubs), h.Len(Diamonds), h.Len(Hearts), h.Len(Spades)
	if tr.check(ls >= 5 && ls >= lh && ls >= lc && ls >= ld,
		"5 ♠ ou plus, la couleur la plus longue → 1♠",
		"5+ ♠, the longest suit → 1♠", cards(h, Spades)) {
		if tr.check(ls == 5 && lc == 5 && h.H() >= 14,
			"exactement 5♠-5♣ avec 14 H et plus → 1♣",
			"exactly 5♠-5♣ with 14+ H → 1♣", cards(h, Clubs)+", "+pts(h.H(), "H")) {
			return Clubs // 5S-5C: 1C recommended from 14H
		}
		return Spades
	}
	if tr.check(lh >= 5 && lh >= ls && lh >= lc && lh >= ld,
		"5 ♥ ou plus, la couleur la plus longue → 1♥",
		"5+ ♥, the longest suit → 1♥", cards(h, Hearts)) {
		return Hearts
	}
	minors := cards(h, Diamonds) + " / " + cards(h, Clubs)
	switch {
	case tr.check(lc > ld, "♣ plus long que ♦ → 1♣", "♣ longer than ♦ → 1♣", minors):
		return Clubs
	case tr.check(ld > lc, "♦ plus long que ♣ → 1♦", "♦ longer than ♣ → 1♦", minors):
		return Diamonds
	case tr.check(lc == 3, "mineures 3-3 → 1♣", "3-3 in the minors → 1♣", minors):
		return Clubs // 3-3: open 1C
	default:
		tr.check(true, "mineures 4-4 ou 5-5 → 1♦", "4-4 or 5-5 in the minors → 1♦", minors)
		return Diamonds // 4-4 and 5-5: open 1D
	}
}

// sevenCardSolidMinor detects the shape for the 3NT opening (docs/bidings.md,
// "L'OUVERTURE DE 3SA"): a running seven-card minor with A, K and Q all
// present ("affranchie" -- at least ARDxxxx), and essentially nothing else
// in the hand -- no outside ace or king, at most one outside queen. The
// modern treatment drops the old requirement of an outside ace ("Autrefois,
// cette ouverture se faisait avec un As annexe : oubliez cette possibilité").
func sevenCardSolidMinor(h *Hand) (Suit, bool) {
	for _, s := range []Suit{Clubs, Diamonds} {
		if h.Len(s) != 7 {
			continue
		}
		if !h.HasCard(s, 'A') || !h.HasCard(s, 'K') || !h.HasCard(s, 'Q') {
			continue
		}
		clean, outsideQueens := true, 0
		for os := Clubs; os <= Spades; os++ {
			if os == s {
				continue
			}
			if h.HasCard(os, 'A') || h.HasCard(os, 'K') {
				clean = false
				break
			}
			if h.HasCard(os, 'Q') {
				outsideQueens++
			}
		}
		if clean && outsideQueens <= 1 {
			return s, true
		}
	}
	return 0, false
}

// weakTwoShape reports whether the hand is the one a weak two major describes
// [O-7a]. The barrage at the three and four levels asks far less: it buys its
// level with length alone, while the two level buys almost nothing and has to
// be paid for in the quality of the hand.
//
//   - exactly six cards in the major, and a suit worth playing: two of the top
//     five honours and at least the three honour points of QJ9xxx, the
//     classical minimum. JT9xxx has the shape and not the suit;
//   - no side suit of five cards -- a four-card minor is tolerated, a fifth
//     card makes the hand a two-suiter, which is not what the bid says;
//   - no four cards in the other major, which partner would never find;
//   - at most one defensive trick outside the suit, an ace or a king. Two of
//     them and the hand defends better than it preempts.
func weakTwoShape(h *Hand, long Suit) bool {
	if h.Len(long) != 6 || !long.IsMajor() {
		return false
	}
	if !h.GoodSuit(long) || h.SuitH(long) < 3 {
		return false
	}
	defence := 0
	for s := Clubs; s <= Spades; s++ {
		if s == long {
			continue
		}
		if h.Len(s) >= 5 {
			return false
		}
		if s.IsMajor() && h.Len(s) >= 4 {
			return false
		}
		if h.HasCard(s, 'A') {
			defence++
		}
		if h.HasCard(s, 'K') {
			defence++
		}
	}
	return defence <= 1
}

func (e *Engine) preemptOpening(h *Hand) (Call, meaning, bool) {
	long := h.Longest()
	n := h.Len(long)
	hp := h.H()
	otherMajor4 := (long != Hearts && h.Len(Hearts) >= 4) || (long != Spades && h.Len(Spades) >= 4)
	// Every preempt wants the honour strength concentrated in the suit
	// itself: with the points mostly outside (e.g. JT9xxx and seven
	// scattered points), the hand has too much defence and too weak a suit
	// to preempt -- it passes.
	tr := e.tr
	if !tr.check(h.SuitH(long)*2 >= h.H(),
		"la moitié des points au moins dans la couleur longue",
		"at least half the points in the long suit",
		fmt.Sprintf("%d H sur %d", h.SuitH(long), h.H())) {
		return Call{}, meaning{}, false
	}
	// No weak two in fourth seat: three passes have gone round, there is
	// nobody left to preempt, and the hand that would open one is worth a
	// pass instead [O-7a].
	fourthSeat := len(e.calls) == 3
	switch {
	case tr.check(weakTwoShape(h, long) && hp >= 6 && hp <= 11 && !fourthSeat,
		"six belles cartes en majeure, 6-11 H, pas en quatrième → 2 majeur faible [O-7a]",
		"six good cards in a major, 6-11 H, not fourth seat → weak two [O-7a]",
		cards(h, long)+", "+pts(hp, "H")):
		return bidSuit(2, long), m(6, 11, "2 majeur faible, 6 belles cartes", "weak two, six good cards").withLen(long, 6), true
	case tr.check(n == 7 && hp <= 10 && h.GoodSuit(long) && !otherMajor4,
		"belle septième, 10 H au plus, sans 4 cartes dans l'autre majeure → barrage à 3",
		"good seven-card suit, 10 H at most, no four cards in the other major → three-level preempt",
		cards(h, long)+", "+pts(hp, "H")):
		return bidSuit(3, long), m(5, 10, "barrage, 7 cartes", "preempt, seven cards").withLen(long, 7), true
	case tr.check(n >= 8 && hp <= 10,
		"huit cartes ou plus, 10 H au plus → barrage à 4",
		"eight or more cards, 10 H at most → four-level preempt",
		cards(h, long)+", "+pts(hp, "H")):
		return bidSuit(4, long), m(5, 10, "barrage, 8 cartes", "preempt, eight cards").withLen(long, 8), true
	}
	return Call{}, meaning{}, false
}

// ---------- responses ----------

func (e *Engine) respond(p *playerState) (Call, meaning) {
	oc := e.openCall
	// The strong, artificial 2C/2D opening promises nothing about its own
	// suit, so an intervention right over it (before the forced relay)
	// cannot be met with the generic "support the opened suit" treatment:
	// there is no suit to support. Handle it on its own before falling into
	// respondCompetitive (docs/bidings.md, "LA DÉFENSE APRÈS UNE
	// INTERVENTION DU N°2").
	if (oc == bid(2, SClubs) || oc == bid(2, SDiamonds)) && e.interferenceAfterOpening() {
		return e.respondStrongTwoInterference(p)
	}
	if e.interferenceAfterOpening() {
		return e.respondCompetitive(p)
	}
	switch {
	case oc == bid(1, SNoTrump):
		return e.respondNT(p, 1, 15)
	case oc == bid(2, SNoTrump):
		return e.respondNT(p, 2, 20)
	case oc == bid(2, SClubs):
		e.tr.check(true, "ouverture 2♣ forte du partenaire → relais 2♦ obligatoire, quelle que soit la main",
			"partner's strong 2♣ opening → forced 2♦ relay, whatever the hand", "")
		p.planned = func() (Call, meaning) { return e.respondAfter2C(p) }
		mn := m(0, 40, "2K relais obligatoire", "2D, forced relay").asForcing()
		mn.relay = true
		return bid(2, SDiamonds), mn
	case oc == bid(2, SDiamonds):
		return e.respondAces(p)
	case oc == bid(3, SNoTrump):
		return e.respondMinorAffranchie(p)
	case oc.Level == 1 && oc.Strain <= SDiamonds:
		return e.respondMinor(p, Suit(oc.Strain))
	case oc.Level == 1 && oc.Strain <= SSpades:
		return e.respondMajor(p, Suit(oc.Strain))
	case oc.Level == 2 && (oc.Strain == SHearts || oc.Strain == SSpades):
		return e.respondWeak2(p, Suit(oc.Strain))
	default:
		return e.conclude(p)
	}
}

func (e *Engine) respondMajor(p *playerState, M Suit) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	sup := h.Len(M)
	hld := h.HLD(M)
	tr := e.tr
	sym := suitSymbol[M]

	// A four-card spade suit over 1H takes priority only without a genuine
	// fit: three-card support is already enough for the sup >= 3 block below
	// to value properly (new suit before support when strong, 13HLD and up;
	// a direct raise otherwise) -- there is no reason to delay a known fit
	// by detouring through spades first.
	if M == Hearts && tr.check(h.Len(Spades) >= 4 && hl >= 6 && sup < 3,
		"sur 1♥ : 4 ♠ et 6 HL, sans fit à 3 cartes → 1♠ [RM-1]",
		"over 1♥: 4 ♠ and 6 HL, no three-card fit → 1♠ [RM-1]",
		cards(h, Spades)+", "+cards(h, Hearts)+", "+pts(hl, "HL")) {
		return bid(1, SSpades), m(6, 40, "changement de couleur 1 sur 1, 4 cartes et plus, forcing", "one-over-one response, 4+ cards, forcing").withLen(Spades, 4).asForcing()
	}
	// Rencontre after a prior pass (docs/addon_6.md, section V): having
	// already passed, a jump is capped and shows a fit plus a good 5+ card
	// suit rather than its usual unlimited meaning. There is no rencontre in
	// a direct (non-passed) response to a one-major opening.
	if sup >= 4 && e.passedBeforeOpening(p.seat) {
		tr.check(true, "main déjà passée avec 4 atouts → enchère de rencontre envisagée [RM-2]",
			"passed hand with four trumps → meeting bid considered [RM-2]", cards(h, M))
		tr.in()
		for _, cand := range []Suit{Clubs, Diamonds, Hearts, Spades} {
			if cand == M {
				continue
			}
			if c, mn, ok := e.tryRencontre(p, M, cand, 4, 1); ok {
				tr.check(true, "belle cinquième à "+suitSymbol[cand]+", 8-11 H → rencontre",
					"good five cards in "+suitSymbol[cand]+", 8-11 H → meeting bid", cards(h, cand))
				tr.out()
				return c, mn
			}
		}
		tr.check(false, "aucune couleur de rencontre (belle cinquième, 8-11 H)",
			"no meeting suit (good five cards, 8-11 H)", pts(h.H(), "H"))
		tr.out()
	}
	// Drury [RM-2b]: having passed, the hand facing a third- or
	// fourth-seat major opening cannot know whether that opening holds real
	// values -- the whole point of opening light in those seats. With a fit
	// and 11 HLD and up, 2C asks before the auction commits past 2M, and 2SA
	// says the same with four trumps and a singleton. The rencontre above
	// keeps its priority: it shows the fit *and* a source of tricks in one
	// bid, which the artificial ask cannot.
	if e.passedBeforeOpening(p.seat) {
		if c, mn, ok := e.druryAsk(p, M); tr.check(ok,
			"main déjà passée, fit et 11 HLD et plus → Drury [RM-2b]",
			"passed hand, fit and 11+ HLD → Drury [RM-2b]", cards(h, M)+", "+pts(hld, "HLD")) {
			return c, mn
		}
	}
	// Splinter: 4+ trumps, a singleton or void elsewhere, no good 5-card side
	// suit, 13-15 HLD (docs/addon_5.md).
	if tr.check(sup >= 4 && hld >= 13 && hld <= 15,
		"4 atouts et 13-15 HLD → splinter envisagé [RM-3]",
		"four trumps and 13-15 HLD → splinter considered [RM-3]", cards(h, M)+", "+pts(hld, "HLD")) {
		tr.in()
		short, ok := splinterSuit(h, M)
		tr.check(ok, "singleton ou chicane, sans belle cinquième à côté → splinter",
			"singleton or void, no good side five-card suit → splinter", shape(h))
		tr.out()
		if ok {
			c := bid(e.cheapestCall(short.Strain()).Level+2, short.Strain())
			if e.legal(p.seat, c) {
				kind := "singleton"
				kindEN := "a singleton"
				if h.Len(short) == 0 {
					kind, kindEN = "chicane", "a void"
				}
				fr := "Splinter : fit " + suitNameFR[M] + ", " + kind + " à " + suitNameFR[short] + ", 13-15HLD"
				en := "splinter: " + suitNameEN[M] + " fit, " + kindEN + " in " + suitNameEN[short] + ", 13-15 HLD"
				mn := m(13, 15, fr, en).withLen(M, 4).withShort(short, h.Len(short)).asForcing()
				mn.splinter = true
				e.gameForce[sideOf(p.seat)] = true
				return c, mn
			}
		}
	}
	if tr.check(sup >= 3,
		"3 atouts ou plus : fit "+sym+" [RM-4]",
		"three or more trumps: "+sym+" fit [RM-4]", cards(h, M)) {
		tr.in()
		defer tr.out()
		switch {
		case tr.check(hld >= 13,
			"13 HLD et plus → nouvelle couleur avant de soutenir",
			"13+ HLD → new suit before supporting", pts(hld, "HLD")):
			if s, lvl, ok := p.bestNewSuit(M); ok {
				p.planned = nil
				// bestNewSuit may fall back on a three-card minor: record only
				// the length actually held.
				mn := m(13, 40, "changement de couleur avant soutien, 13HLD et plus", "new suit before supporting, 13+ HLD").withLen(s, min(4, h.Len(s))).asForcing()
				return bidSuit(lvl, s), mn
			}
			// Every side suit is too short to name honestly (a hand built
			// around long trumps): let the generic engine value the fit
			// directly -- invitation, game or slam try on the combined count.
			tr.note("aucune couleur annexe à nommer : le fit est évalué directement (manche ou chelem)",
				"no side suit worth naming: the fit is valued directly (game or slam)")
			if c, mn := e.conclude(p); c.Kind != KindPass {
				return c, mn
			}
			return bidSuit(3, M), m(11, 12, "soutien à saut", "jump raise").withLen(M, min(4, sup)).asInvite()
		case tr.check(hld >= 11 && sup == 3,
			"11-12 HLD avec 3 atouts → 2SA fitté",
			"11-12 HLD with three trumps → conventional 2NT", pts(hld, "HLD")):
			mn := m(11, 12, "2SA fitté : 3 atouts, 11-12HLD, proposition de manche", "conventional 2NT: 3-card support, 11-12 HLD, game try").withLen(M, 3).asForcing()
			return bid(2, SNoTrump), mn
		case tr.check(hld >= 11,
			"11-12 HLD avec 4 atouts → soutien à saut",
			"11-12 HLD with four trumps → jump raise", pts(hld, "HLD")):
			return bidSuit(3, M), m(11, 12, "soutien à saut, 4 atouts, 11-12HLD", "jump raise, 4 trumps, 11-12 HLD").withLen(M, 4).asInvite()
		case tr.check(hld >= 6,
			"6-10 HLD → soutien simple",
			"6-10 HLD → single raise", pts(hld, "HLD")):
			return bidSuit(2, M), m(6, 10, "soutien simple, 6-10HLD", "single raise, 6-10 HLD").withLen(M, 3)
		}
		tr.note("moins de 6 HLD → Passe", "fewer than 6 HLD → Pass")
		return passCall, m(0, 5, "moins de 6 points", "fewer than 6 points")
	}
	if tr.check(hl >= 11,
		"sans fit, 11 HL et plus → changement de couleur à 2 [RM-5]",
		"no fit, 11+ HL → two-level new suit [RM-5]", pts(hl, "HL")) {
		s, lvl, ok := p.bestNewSuit(M)
		tr.in()
		tr.check(ok, "une couleur à nommer au palier de 2", "a suit to name at the two level", shape(h))
		tr.out()
		if ok {
			// A hand that has already passed cannot commit the side to game:
			// its own pass caps it at 11 [E-1b], and the opening it faces is
			// a third- or fourth-seat one that may itself be light [O-7b].
			// Eleven opposite ten is not a game, so the change of suit is
			// not forcing at all: it names the suit the side is likeliest to
			// play, and the opening -- which knows whether it was light --
			// is free to pass it there. Drury [RM-2b] is the one question a
			// passed hand still gets to ask, and it asks it with a fit.
			if tr.check(p.passedOpening,
				"main déjà passée : le 2 sur 1 n'est pas forcing [RM-1b]",
				"passed hand: the two-over-one is not forcing [RM-1b]", "") {
				mn := m(11, 11, "changement de couleur 2 sur 1, main déjà passée : non forcing", "two-over-one by a passed hand: not forcing")
				return bidSuit(lvl, s), mn.withLen(s, min(4, h.Len(s)))
			}
			// A two-over-one response commits the partnership to game regardless
			// of opener's rebid: the auction must not be passed out below game.
			e.gameForce[sideOf(p.seat)] = true
			return bidSuit(lvl, s), m(11, 40, "changement de couleur 2 sur 1, 11HL et plus, forcing de manche", "two-over-one response, 11+ HL, game forcing").withLen(s, min(4, h.Len(s))).asForcing()
		}
	}
	if tr.check(hl >= 6,
		"sans fit, 6-10 HL → 1SA [RM-6]",
		"no fit, 6-10 HL → 1NT [RM-6]", pts(hl, "HL")) {
		return bid(1, SNoTrump), m(6, 10, "1SA \"poubelle\", 6-10HL sans fit", "1NT response, 6-10 HL, no fit")
	}
	tr.note("moins de 6 HL → Passe", "fewer than 6 HL → Pass")
	return passCall, m(0, 5, "moins de 6 points", "fewer than 6 points")
}

// bestNewSuit picks the suit for a forcing change of suit over partner's M:
// the longest genuine 4+ card suit, else a minor of at least three cards --
// never shorter, or partner's raise could land the side in a 4-2 "fit".
// ok is false when no suit qualifies at all.
func (p *playerState) bestNewSuit(M Suit) (Suit, int, bool) {
	h := p.hand
	best, bestLen := Clubs, 0
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if s == M {
			continue
		}
		need := 4
		if M == Spades && s == Hearts {
			need = 5 // 2H over 1S promises five cards
		}
		if h.Len(s) >= need && h.Len(s) > bestLen {
			best, bestLen = s, h.Len(s)
		}
	}
	if bestLen == 0 {
		// fall back on the better minor, three cards at least
		for _, s := range []Suit{Diamonds, Clubs} {
			if s != M && h.Len(s) >= 3 && h.Len(s) > bestLen {
				best, bestLen = s, h.Len(s)
			}
		}
		if bestLen == 0 {
			return Clubs, 0, false
		}
	}
	lvl := 1
	if best.Strain() <= M.Strain() {
		lvl = 2
	}
	if M == Hearts && best == Spades {
		lvl = 1
	}
	return best, lvl, true
}

// splinterSuit finds the single suit worth splintering in: a singleton or a
// void other than the fit, with no good 5+ card side suit standing in the
// way (docs/addon_5.md). A good side suit, or more than one short suit,
// disqualifies the splinter.
func splinterSuit(h *Hand, M Suit) (Suit, bool) {
	short := Clubs
	shortCount := 0
	for s := Clubs; s <= Spades; s++ {
		if s == M {
			continue
		}
		switch {
		case h.Len(s) >= 5 && h.GoodSuit(s):
			return Clubs, false
		case h.Len(s) <= 1:
			short, shortCount = s, shortCount+1
		}
	}
	if shortCount != 1 {
		return Clubs, false
	}
	return short, true
}

// isRencontre reports whether the hand qualifies for a "rencontre"
// (GhBridge, docs/addon_6.md): 8-11 HCP, at least minFit cards in the fit
// suit, and a good 5+ card suit in candidate.
func isRencontre(h *Hand, fit Suit, minFit int, candidate Suit) bool {
	return h.H() >= 8 && h.H() <= 11 && h.Len(fit) >= minFit && h.Len(candidate) >= 5 && h.GoodSuit(candidate)
}

// rencontreCall builds the jump call and meaning for a rencontre in
// candidate, jumpLevels steps beyond the cheapest legal reply.
func (e *Engine) rencontreCall(p *playerState, fit, candidate Suit, minFit, jumpLevels int) (Call, meaning, bool) {
	h := p.hand
	if !isRencontre(h, fit, minFit, candidate) {
		return Call{}, meaning{}, false
	}
	c := bid(e.cheapestCall(candidate.Strain()).Level+jumpLevels, candidate.Strain())
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	fr := "enchère de rencontre, 5+ belles cartes à " + suitNameFR[candidate] + ", fit " + suitNameFR[fit] + ", 8-11H"
	en := "meeting bid, 5+ good " + suitNameEN[candidate] + ", " + suitNameEN[fit] + " fit, 8-11 HCP"
	mn := m(8, 11, fr, en).withLen(fit, minFit).withLen(candidate, 5).asForcing()
	return c, mn, true
}

// druryShortSuit returns the singleton or void of a hand holding four-card
// support, when there is exactly one: the 2SA Drury promises a short suit
// opener can then ask about, and two of them would leave the answer
// ambiguous.
func druryShortSuit(h *Hand, M Suit) (Suit, bool) {
	short, n := Clubs, 0
	for s := Clubs; s <= Spades; s++ {
		if s == M {
			continue
		}
		if h.Len(s) <= 1 {
			short, n = s, n+1
		}
	}
	return short, n == 1
}

// druryAsk produces the Drury response over partner's third- or fourth-seat
// major opening: 2C with three-card support (or four without a short suit),
// 2SA with four trumps and a singleton. Both are artificial and forcing, and
// both promise 11 HLD and up -- the zone where a passed hand must find out
// whether the opening was a real one before selling out to the game.
func (e *Engine) druryAsk(p *playerState, M Suit) (Call, meaning, bool) {
	if !M.IsMajor() || !e.passedBeforeOpening(p.seat) || !e.uncontested(p.seat) {
		return Call{}, meaning{}, false
	}
	h := p.hand
	sup := h.Len(M)
	if sup < 3 || h.HLD(M) < 11 {
		return Call{}, meaning{}, false
	}
	if short, ok := druryShortSuit(h, M); ok && sup >= 4 {
		c := bid(2, SNoTrump)
		if e.legal(p.seat, c) {
			mn := m(11, -1, "2SA Drury : fit quatrième et un singleton, 11HLD et plus (alerte)",
				"2NT Drury: four-card support with a singleton, 11+ HLD (alertable)").
				withLen(M, 4).withShort(short, h.Len(short)).asForcing()
			mn.druryShort = true
			return c, mn, true
		}
	}
	c := bid(2, SClubs)
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	mn := m(11, -1, "2T Drury : fit et 11HLD et plus, l'ouverture de troisième ou quatrième a-t-elle ses points ? (alerte)",
		"2C Drury: fit and 11+ HLD, asking whether the third/fourth-seat opening holds its values (alertable)").
		withLen(M, 3).asForcing()
	mn.drury = true
	return c, mn, true
}

// drurySecondSuit names the second suit of a genuine two-suiter over the
// Drury ask. Only a major can be shown: 2D is the artificial game try and 2C
// was the ask itself, so a club or diamond side suit has no room below the
// sign-off and waits for the next round. Over 1S the heart bid stays under
// 2S and costs nothing; over 1H the spade bid buys the two level in the
// wrong suit, so it asks for more.
func (e *Engine) drurySecondSuit(p *playerState, os Suit, hld int) (Call, meaning, bool) {
	other := Hearts
	if os == Hearts {
		other = Spades
	}
	h := p.hand
	if h.Len(os) < 5 || h.Len(other) < 4 || !h.GoodSuit(other) && h.SuitH(other) < 4 {
		return Call{}, meaning{}, false
	}
	c := bid(2, other.Strain())
	floor := 15
	if !bidSuit(2, os).higherThan(c) {
		floor = 17
	}
	if hld < floor || !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	fr := "beau bicolore " + suitNameFR[os] + "-" + suitNameFR[other] + " sur le Drury"
	en := "good " + suitNameEN[os] + "-" + suitNameEN[other] + " two-suiter over the Drury"
	mn := m(floor, 23, fr, en).withLen(os, 5).withLen(other, 4).asForcing()
	return c, mn, true
}

// druryRebid is opener's answer to the 2C Drury. The responder has promised
// a fit and 11 HLD, so the whole ladder is a subtraction against the
// combined thresholds [E-9]: 27 for the major game, the 29-32 window for the
// control exchange [S-3].
func (e *Engine) druryRebid(p *playerState, os Suit) (Call, meaning) {
	hld := e.hldAgainstTheirBidding(p, os)
	if c, mn, ok := e.drurySecondSuit(p, os, hld); ok {
		return c, mn
	}
	// The zones read off the combined thresholds against the 11 HLD the
	// Drury promised: 26 and less leaves the game out of reach, 27 puts it
	// there [E-9], 30 opens the control window [S-3]. A minimum opening
	// values at 13-14 HLD once the fit is counted, which is exactly the
	// hand the convention exists to stop below 3M.
	switch {
	case hld >= 19:
		c := bidSuit(3, os)
		if e.legal(p.seat, c) {
			return c, m(19, 23, "répétition à saut : ambition de chelem, commence les contrôles",
				"jump repeat: slam ambition, start the control exchange").withLen(os, 5).asForcing()
		}
	case hld >= 17:
		c := bidSuit(4, os)
		if e.legal(p.seat, c) {
			return c, m(17, 18, "je veux jouer la manche, ne pensons pas au chelem",
				"bidding the game, no slam interest").withLen(os, 5)
		}
	case hld >= 15:
		c := bid(2, SDiamonds)
		if e.legal(p.seat, c) {
			mn := m(15, 16, "2K Drury : ambition de manche, décris ton soutien (alerte)",
				"2D Drury: game ambition, describe your support (alertable)").asForcing()
			mn.druryGameTry = true
			return c, mn
		}
	}
	// Minimum opening: the sign-off the convention was invented to reach --
	// two of the major, below the game the raw count would otherwise have
	// bought.
	c := bidSuit(2, os)
	if e.legal(p.seat, c) {
		return c, m(12, 14, "ouverture minimale, restons-en là", "minimum opening, this is far enough").withLen(os, 5)
	}
	return e.conclude(p)
}

// druryShortRebid answers the 2NT Drury (four trumps and a singleton): with a
// minimum, three of the major and no further; otherwise 3C asks which
// singleton, since a short suit facing wasted honours is worth far less than
// the same count facing a fitting one.
func (e *Engine) druryShortRebid(p *playerState, os Suit) (Call, meaning) {
	if e.hldAgainstTheirBidding(p, os) <= 14 {
		c := bidSuit(3, os)
		if e.legal(p.seat, c) {
			return c, m(12, 14, "ouverture minimale, arrêt au palier de 3", "minimum opening, stopping at the three level").withLen(os, 5)
		}
		return e.conclude(p)
	}
	c := bid(3, SClubs)
	if !e.legal(p.seat, c) {
		return e.conclude(p)
	}
	mn := m(15, 23, "3T : quel est ton singleton ? (alerte)", "3C: which singleton? (alertable)").asForcing()
	mn.druryShortAsk = true
	return c, mn
}

// druryGameTryAnswer describes the support the 2C Drury only promised as
// three cards: the extra trump is the extra trick, so the level rises with
// the length -- and a fifth trump bids the game on distributional safety
// alone.
func (e *Engine) druryGameTryAnswer(p *playerState, os Suit) (Call, meaning) {
	sup := p.hand.Len(os)
	lvl := 2
	switch {
	case sup >= 5:
		lvl = 4
	case sup == 4:
		lvl = 3
	}
	c := bidSuit(lvl, os)
	if !e.legal(p.seat, c) {
		return e.conclude(p)
	}
	fr := fmt.Sprintf("%d cartes de soutien", sup)
	en := fmt.Sprintf("%d-card support", sup)
	if sup >= 5 {
		fr += ", la manche par sécurité distributionnelle"
		en += ", bidding game on distributional safety"
	}
	// The ask was about length, but the answer is also this hand's second
	// description: the Drury only promised eleven, and an opener judging the
	// game on that floor alone signs off two points short of what the hand
	// holds. Its own valuation in the fit is what the length answer carries.
	return c, m(max(11, p.hand.HLD(os)), -1, fr, en).withLen(os, min(5, sup))
}

// druryShortAnswer names the singleton the 2NT Drury promised. Clubs cannot
// be named -- 3C was the question -- so the return to opener's suit at the
// three level says clubs.
func (e *Engine) druryShortAnswer(p *playerState, os Suit) (Call, meaning) {
	short, ok := druryShortSuit(p.hand, os)
	if !ok {
		return e.conclude(p)
	}
	c := bid(3, short.Strain())
	if short == Clubs {
		c = bidSuit(3, os)
	}
	if !e.legal(p.seat, c) {
		return e.conclude(p)
	}
	kind, kindEN := "singleton", "singleton"
	if p.hand.Len(short) == 0 {
		kind, kindEN = "chicane", "void"
	}
	return c, m(11, -1, kind+" à "+suitNameFR[short], kindEN+" in "+suitNameEN[short]).
		withLen(os, 4).withShort(short, p.hand.Len(short))
}

func (e *Engine) respondMinor(p *playerState, ms Suit) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	tr := e.tr
	if tr.check(hl < 6, "moins de 6 HL → Passe", "fewer than 6 HL → Pass", pts(hl, "HL")) {
		return passCall, m(0, 5, "moins de 6 points", "fewer than 6 points")
	}
	// Majors first ("balayer les couleurs quatrièmes").
	lh, ls := h.Len(Hearts), h.Len(Spades)
	if tr.check(lh >= 4 || ls >= 4,
		"une majeure quatrième → 1 à la majeure (la plus longue, ♥ à 4-4)",
		"a four-card major → one of the major (the longer, ♥ with 4-4)",
		cards(h, Hearts)+", "+cards(h, Spades)) {
		var s Suit
		switch {
		case lh >= 5 || ls >= 5:
			if ls >= lh {
				s = Spades
			} else {
				s = Hearts
			}
		case lh >= 4:
			s = Hearts
		default:
			s = Spades
		}
		return bidSuit(1, s), m(6, 40, "réponse 1 sur 1 en majeure, 4 cartes et plus, forcing", "one-over-one major response, 4+ cards, forcing").withLen(s, 4).asForcing()
	}
	// Rencontre: single jump in the other minor, 4+ fit in the opened minor,
	// a good 5+ card suit there, 8-11 HCP (docs/addon_6.md).
	other := Diamonds
	if ms == Diamonds {
		other = Clubs
	}
	if c, mn, ok := e.rencontreCall(p, ms, other, 4, 1); tr.check(ok,
		"4 atouts et belle cinquième dans l'autre mineure, 8-11 H → rencontre",
		"four trumps and a good five cards in the other minor, 8-11 H → meeting bid",
		cards(h, ms)+", "+cards(h, other)) {
		return c, mn
	}
	if ms == Clubs && tr.check(h.Len(Diamonds) >= 4,
		"sur 1♣ : 4 ♦ et plus → 1♦", "over 1♣: 4+ ♦ → 1♦", cards(h, Diamonds)) {
		return bid(1, SDiamonds), m(6, 40, "réponse de 1K, 4 cartes et plus, forcing", "1D response, 4+ cards, forcing").withLen(Diamonds, 4).asForcing()
	}
	if tr.check(h.IsRegular(), "main régulière → réponse à Sans-Atout selon la force",
		"balanced hand → notrump response by strength", shape(h)) {
		tr.in()
		switch {
		case tr.check(hl <= 10, "6-10 HL → 1SA", "6-10 HL → 1NT", pts(hl, "HL")):
			return bid(1, SNoTrump), m(6, 10, "1SA, 6-10HL régulier sans majeure", "1NT, 6-10 HL, balanced, no four-card major")
		case tr.check(hl <= 12, "11-12 HL → 2SA", "11-12 HL → 2NT", pts(hl, "HL")):
			return bid(2, SNoTrump), m(11, 12, "2SA, 11-12HL régulier", "2NT, 11-12 HL, balanced").asInvite()
		case tr.check(hl <= 15, "13-15 HL → 3SA", "13-15 HL → 3NT", pts(hl, "HL")):
			return bid(3, SNoTrump), m(13, 15, "3SA, 13-15HL régulier", "3NT, 13-15 HL, balanced")
		}
		// Au-delà de 15 HL, la main régulière poursuit plus bas : les tests
		// suivants ne sont plus des sous-cas de celui-ci.
		tr.out()
	}
	hld := h.HLD(ms)
	sup := h.Len(ms)
	minSup := 5
	if ms == Diamonds {
		minSup = 4
	}
	if tr.check(hl >= 13, "13 HL et plus → main de manche", "13+ HL → game-going hand", pts(hl, "HL")) {
		tr.in()
		// A forcing change of suit needs a genuine suit: opener supports it
		// with four trumps, so naming a 2-3 card holding can strand the side
		// in a 4-2 "fit". (Over 1C, four diamonds were already bid above.)
		if tr.check(h.Len(other) >= 4,
			"4 cartes dans l'autre mineure → changement de couleur forcing",
			"four cards in the other minor → forcing new suit", cards(h, other)) {
			tr.out()
			lvl := 1
			if other.Strain() <= ms.Strain() {
				lvl = 2
			}
			return bidSuit(lvl, other), m(13, 40, "changement de couleur forcing, 13HL et plus", "forcing change of suit, 13+ HL").withLen(other, 4).asForcing()
		}
		// No four-card suit outside the opened minor: with every side suit
		// guarded, 3SA describes the hand better than any forcing bid.
		if hl <= 18 {
			stopped := true
			for s := Clubs; s <= Spades; s++ {
				if s != ms && !h.Stopper(s) {
					stopped = false
					break
				}
			}
			if tr.check(stopped, "13-18 HL, toutes les couleurs gardées → 3SA",
				"13-18 HL, every suit stopped → 3NT", pts(hl, "HL")) {
				tr.out()
				return bid(3, SNoTrump), m(13, 18, "3SA, 13-18HL, sans majeure ni couleur annonçable", "3NT, 13-18 HL, no major and no biddable suit")
			}
		}
		// Otherwise let the generic engine place the contract from the
		// combined count (fit raise, notrump game or slam try).
		tr.note("le contrat se cherche sur la force combinée (fit, Sans-Atout ou chelem)",
			"the contract is sought on the combined strength (fit, notrump or slam)")
		tr.out()
		if c, mn := e.conclude(p); c.Kind != KindPass {
			return c, mn
		}
	}
	if tr.check(sup >= minSup,
		fmt.Sprintf("%d atouts ou plus dans la mineure → soutien", minSup),
		fmt.Sprintf("%d+ trumps in the minor → raise", minSup), cards(h, ms)) {
		if tr.check(hld >= 11, "11-12 HLD → soutien à 3", "11-12 HLD → limit raise", pts(hld, "HLD")) {
			return bidSuit(3, ms), m(11, 12, "soutien à 3, 11-12HLD", "limit raise, 11-12 HLD").withLen(ms, sup).asInvite()
		}
		return bidSuit(2, ms), m(6, 10, "soutien simple, 6-10HLD, dénie une majeure", "single raise, 6-10 HLD, denies a major").withLen(ms, sup)
	}
	if tr.check(hl >= 11 && h.Len(other) >= 4,
		"11 HL et 4 cartes dans l'autre mineure → changement de couleur forcing",
		"11 HL and four cards in the other minor → forcing new suit", cards(h, other)) {
		lvl := 1
		if other.Strain() <= ms.Strain() {
			lvl = 2
		}
		return bidSuit(lvl, other), m(11, 40, "changement de couleur, forcing", "forcing change of suit").withLen(other, 4).asForcing()
	}
	tr.note("aucune autre réponse ne s'applique → 1SA", "no other response applies → 1NT")
	return bid(1, SNoTrump), m(6, 10, "1SA, 6-10HL", "1NT, 6-10 HL")
}

// respondNT handles responses to 1NT and 2NT openings (base = opening level).
func (e *Engine) respondNT(p *playerState, base, oMin int) (Call, meaning) {
	h := p.hand
	hp := h.H()
	lh, ls := h.Len(Hearts), h.Len(Spades)
	tr := e.tr
	majors := cards(h, Hearts) + ", " + cards(h, Spades)

	// Transfers with a five-card major: a genuine single-suiter only. A
	// four-five (or five-four) two-suited major hand goes through Stayman
	// instead, so it can explore either fit (docs/bidings.md, "chassé-croisé").
	if tr.check((lh >= 5 && ls < 4) || (ls >= 5 && lh < 4),
		"une majeure cinquième, sans 4 cartes dans l'autre → Texas",
		"a five-card major without four in the other → transfer", majors) {
		var M Suit
		if ls >= lh {
			M = Spades
		} else {
			M = Hearts
		}
		// "Misère dorée" (docs/addon_11.md, Bridgeur n°813): a limited hand
		// (7-8H) with a genuine singleton is worth more than its bare count
		// suggests opposite an ordinary 1NT, but a Texas transfer would just
		// sign off below game with no way to describe the shape. Route
		// through Stayman instead to locate a nine-card fit first.
		if base == 1 && tr.check(hp >= 7 && hp <= 8 && hasMisereDoreeSingleton(h, M),
			"7-8 H avec un singleton (« misère dorée ») → Stayman plutôt que Texas",
			"7-8 H with a singleton (« misère dorée ») → Stayman rather than a transfer",
			pts(hp, "H")+", "+shape(h)) {
			c := bid(2, SClubs)
			mn := m(7, 8, "Stayman, demande les majeures quatrièmes", "Stayman, asking for four-card majors").asForcing()
			mn.relay = true
			p.planned = func() (Call, meaning) { return e.afterStaymanMisereDoree(p, M) }
			return c, mn
		}
		var c Call
		if base == 1 {
			if M == Hearts {
				c = bid(2, SDiamonds)
			} else {
				c = bid(2, SHearts)
			}
		} else {
			if M == Hearts {
				c = bid(3, SDiamonds)
			} else {
				c = bid(3, SHearts)
			}
		}
		mn := m(0, 40, "Texas pour la majeure cinquième", "transfer, five-card major").withLen(M, 5).asForcing()
		mn.hasTexas, mn.texas = true, M
		completion := bid(c.Level, M.Strain())
		p.planned = func() (Call, meaning) { return e.afterTransfer(p, M, oMin, completion) }
		return c, mn
	}
	// Stayman with a four-card major.
	need := 8
	if base == 2 {
		need = 4
	}
	if tr.check((lh >= 4 || ls >= 4) && hp >= need,
		fmt.Sprintf("une majeure quatrième et %d H et plus → Stayman", need),
		fmt.Sprintf("a four-card major and %d+ H → Stayman", need), majors+", "+pts(hp, "H")) {
		var c Call
		if base == 1 {
			c = bid(2, SClubs)
		} else {
			c = bid(3, SClubs)
		}
		mn := m(need, 40, "Stayman, demande les majeures quatrièmes", "Stayman, asking for four-card majors").asForcing()
		mn.relay = true
		p.planned = func() (Call, meaning) { return e.afterStayman(p, oMin) }
		return c, mn
	}
	// Minor-suit Texas (1NT only): docs/bidings.md, "TEXAS MINEURS", "théorie
	// du singleton".
	if base == 1 {
		if c, mn, ok := e.minorTexas(p); tr.check(ok,
			"longue mineure (6 cartes, ou 5-5) → Texas mineur",
			"long minor (six cards, or 5-5) → minor transfer",
			cards(h, Diamonds)+", "+cards(h, Clubs)) {
			return c, mn
		}
	}
	// Strong minor two-suiter / long minor over a 2NT (opening or the balanced
	// rebid of a 2C/2D strong opening): show it through the minor Texas so a
	// minor slam can be found instead of settling for 3NT.
	if base == 2 {
		if c, mn, ok := e.strongMinorTwoSuiter(p); tr.check(ok,
			"longue mineure ou bicolore mineur, 11 HL et plus → Texas mineur",
			"long minor or minor two-suiter, 11+ HL → minor transfer",
			cards(h, Diamonds)+", "+cards(h, Clubs)) {
			return c, mn
		}
	}
	// Quantitative notrump ladder.
	return e.ntResponseLadder(p, base)
}

// strongMinorTwoSuiter lets responder show a slam-ambitious minor two-suiter
// (5-5 or better) or a long single minor with a genuine short suit over a
// strong 2NT -- a 2NT opening, or the balanced 2NT rebid of a 2C/2D strong
// opening (docs/bidings.md table p.37: 3S = club Texas, 4C = diamond Texas).
// The follow-up (afterStrongMinorTexas) reveals the second minor of a
// two-suiter; the slam itself is then driven by the generic control/Blackwood
// engine in conclude. Below real slam ambition (HL < 11) the hand simply takes
// its game through the notrump ladder rather than climbing past 3NT.
func (e *Engine) strongMinorTwoSuiter(p *playerState) (Call, meaning, bool) {
	h := p.hand
	if h.HL() < 11 {
		return Call{}, meaning{}, false
	}
	lc, ld := h.Len(Clubs), h.Len(Diamonds)
	twoSuited := lc >= 5 && ld >= 5
	var primary Suit
	switch {
	case twoSuited:
		primary = Clubs // transfer through clubs, reveal diamonds next
	case lc >= 6 && ld < 6:
		primary = Clubs
	case ld >= 6 && lc < 6:
		primary = Diamonds
	default:
		return Call{}, meaning{}, false
	}
	// A single long minor is only worth leaving 3NT for with a genuine
	// singleton or void; a 5-5 two-suiter always has one by construction.
	if !twoSuited {
		if _, ok := splinterSuit(h, primary); !ok {
			return Call{}, meaning{}, false
		}
	}
	c := bid(3, SSpades)
	frName, enName := "Texas Trèfle", "club Texas"
	if primary == Diamonds {
		c = bid(4, SClubs)
		frName, enName = "Texas Carreau", "diamond Texas"
	}
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	mn := m(11, 40, frName, enName).withLen(primary, h.Len(primary)).asForcing()
	mn.hasTexas, mn.texas = true, primary
	p.planned = func() (Call, meaning) { return e.afterStrongMinorTexas(p, twoSuited) }
	return c, mn, true
}

// afterStrongMinorTexas continues once opener has rectified the strong minor
// Texas. A two-suiter names the second minor (4D) to confirm the 5-5 double
// fit, game forcing; a single-suiter lets conclude place the contract from the
// length already shown. Either way the slam decision is left to the generic
// control/Blackwood engine.
func (e *Engine) afterStrongMinorTexas(p *playerState, twoSuited bool) (Call, meaning) {
	last, lastSeat, ok := e.lastBid()
	if !ok || lastSeat != partnerOf(p.seat) {
		return e.conclude(p)
	}
	if twoSuited {
		c := bid(4, SDiamonds)
		if last != c && e.legal(p.seat, c) {
			mn := m(11, 40, "bicolore 5T-5K, forcing de manche", "5-5 clubs-diamonds two-suiter, game forcing").
				withLen(Clubs, p.hand.Len(Clubs)).withLen(Diamonds, p.hand.Len(Diamonds)).asForcing()
			e.gameForce[sideOf(p.seat)] = true
			return c, mn
		}
	}
	return e.conclude(p)
}

// ntResponseLadder is the natural, quantitative notrump ladder used both as
// a direct response to 1SA/2SA and, with the same values, after Stayman
// once no major-suit fit or forcing continuation applies (docs/bidings.md,
// "Développements après la réponse au Stayman": "les enchères à Sans-Atout
// ont la même valeur qu'en réponse directe à l'ouverture").
func (e *Engine) ntResponseLadder(p *playerState, base int) (Call, meaning) {
	h := p.hand
	hp := h.H()
	tr := e.tr
	v := pts(hp, "H")
	if base == 1 {
		tr.note("sans majeure à montrer : réponse selon les points", "no major to show: response by points")
		switch {
		case tr.check(hp <= 7, "0-7 H → Passe", "0-7 H → Pass", v):
			return passCall, m(0, 7, "0-8H, pas d'espoir de manche", "0-8 points, no game interest")
		case tr.check(hp <= 9, "8-9 H → 2SA, proposition de manche", "8-9 H → 2NT, game invitation", v):
			return bid(2, SNoTrump), m(8, 9, "2SA, 8-9H, proposition de manche", "2NT, 8-9, game invitation").asInvite()
		case tr.check(hp <= 15, "10-15 H → 3SA", "10-15 H → 3NT", v):
			return bid(3, SNoTrump), m(10, 15, "3SA, 10-15H", "3NT, 10-15")
		case tr.check(hp <= 17, "16-17 H → 4SA quantitatif", "16-17 H → quantitative 4NT", v):
			mn := m(16, 17, "4SA quantitatif, proposition de petit chelem", "quantitative 4NT, small slam try")
			mn.slamInvite = true
			return bid(4, SNoTrump), mn
		default:
			tr.check(true, "18 H et plus → 6SA", "18+ H → 6NT", v)
			return bid(6, SNoTrump), m(18, 19, "6SA, 18-19H", "6NT, 18-19")
		}
	}
	// Facing a 2SA-type opening the ladder is pure arithmetic [E-9]: what
	// counts is this hand added to the floor -- and to the ceiling -- the
	// opening promised, not a fixed number of points. Facing 20-21 the slam
	// zone opens at 13 and the quantitative 4SA at 12; facing the 22-23
	// rebid of a strong 2C, at 11 and 10. A fixed 11/12 split closed at 3SA
	// with 33 combined points on the table. The quantitative try still needs
	// the narrow zone [S-13] asks for: facing an opening whose ceiling is
	// open (the 2K forcing-to-game floor of 24), the ceiling proves nothing
	// and the try belongs to the hand that knows its own.
	partner := e.ps[partnerOf(p.seat)]
	tr.note("sans majeure à montrer : réponse selon la force combinée", "no major to show: response by combined strength")
	sum := fmt.Sprintf("%d + %d-%d", hp, partner.shownMin, partner.shownMax)
	switch {
	case tr.check(hp <= 3, "0-3 H → Passe", "0-3 H → Pass", v):
		return passCall, m(0, 3, "jeu trop faible", "too weak to respond")
	case tr.check(hp+partner.shownMin >= 33,
		"33 H assurés avec le minimum du partenaire → recherche de chelem",
		"33 H guaranteed with partner's minimum → slam search", sum):
		// The slam zone is already reached: hand over to the generic
		// machinery, which asks for keycards [S-8] instead of closing.
		return e.conclude(p)
	case tr.check(hp+partner.shownMax >= 33 && partner.shownMax-partner.shownMin <= 7,
		"33 H possibles avec le maximum du partenaire → 4SA quantitatif",
		"33 H possible with partner's maximum → quantitative 4NT", sum):
		mn := m(hp, 40, "4SA quantitatif, proposition de petit chelem", "quantitative 4NT, small slam try")
		mn.slamInvite = true
		return bid(4, SNoTrump), mn
	default:
		tr.check(true, "sinon → 3SA", "otherwise → 3NT", sum)
		return bid(3, SNoTrump), m(4, 32-partner.shownMin, "3SA sur la force combinée", "3NT on combined strength")
	}
}

// minorTexas implements the responder's minor-suit Texas over a 1SA opening
// (docs/bidings.md, "TEXAS MINEURS", "théorie du singleton"): 2S transfers
// to clubs, 3C transfers to diamonds (mandatory rectification to 3D --
// unlike clubs, 2SA keeps its ordinary quantitative meaning for diamonds).
// It requires six cards without game interest (hl <= 7), or from 10 HL with
// a genuine singleton/void (6322 or 7222 shape). The strength is read in HL
// throughout, never HLD: the shortness pays only once a fit is found, and
// crediting it here would push the 8-9 band into the game-going branch. That
// band takes the notrump ladder instead, shortness or not -- eight points
// opposite 15-17 cannot reach game, and 1NT asks seven tricks where the
// transfer's 3C asks nine.
// A 5-5 clubs-diamonds two-suiter always goes through the club Texas, since
// that shape guarantees a short suit by construction.
func (e *Engine) minorTexas(p *playerState) (Call, meaning, bool) {
	h := p.hand
	hl := h.HL()
	lc, ld := h.Len(Clubs), h.Len(Diamonds)
	twoSuited := lc >= 5 && ld >= 5

	var primary Suit
	switch {
	case twoSuited:
		if hl < 10 {
			return Call{}, meaning{}, false
		}
		primary = Clubs
	case lc >= 6 && ld < 6:
		primary = Clubs
	case ld >= 6 && lc < 6:
		primary = Diamonds
	default:
		return Call{}, meaning{}, false
	}

	weak := hl <= 7
	gameGoing := hl >= 10
	if !weak && !twoSuited {
		if _, ok := splinterSuit(h, primary); !ok {
			return Call{}, meaning{}, false
		}
	}
	if !weak && !gameGoing {
		return Call{}, meaning{}, false
	}

	var c Call
	frName, enName := "Texas Trèfle", "club Texas"
	if primary == Clubs {
		c = bid(2, SSpades)
	} else {
		c = bid(3, SClubs)
		frName, enName = "Texas Carreau", "diamond Texas"
	}
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}

	mn := m(0, 7, frName, enName).withLen(primary, h.Len(primary)).asForcing()
	mn.hasTexas, mn.texas = true, primary
	if weak {
		p.planned = func() (Call, meaning) { return e.afterMinorTexasWeak(p) }
	} else {
		mn.minPts, mn.maxPts = 10, 40
		p.planned = func() (Call, meaning) { return e.afterMinorTexasGame(p, primary, twoSuited) }
	}
	return c, mn, true
}

// minorTransferAnswer rectifies a minor-suit Texas. Clubs alone has two
// possible answers: 2SA declines the transformation with a good fit (K
// doubleton, Q third, or four-plus clubs), else the plain rectification;
// diamonds has no such option, the rectification to 3D is mandatory.
func (e *Engine) minorTransferAnswer(p *playerState, primary Suit) (Call, meaning) {
	h := p.hand
	if primary == Clubs {
		goodFit := h.Len(Clubs) >= 4 ||
			(h.Len(Clubs) == 2 && h.HasCard(Clubs, 'K')) ||
			(h.Len(Clubs) == 3 && h.HasCard(Clubs, 'Q'))
		if goodFit {
			c := bid(2, SNoTrump)
			if e.legal(p.seat, c) {
				return c, m(-1, -1, "refus du Texas Trèfle, bon fit", "declines the club Texas, good fit").withLen(Clubs, h.Len(Clubs))
			}
		}
	}
	c := e.cheapestCall(primary.Strain())
	return c, m(-1, -1, "rectification du Texas mineur", "completes the minor Texas").withLen(primary, h.Len(primary))
}

// afterMinorTexasWeak concludes a weak (no game interest) minor Texas:
// opener's plain rectification is passed, but a declined club Texas (2SA,
// good fit) must be corrected back to 3C -- an absolute sign-off.
func (e *Engine) afterMinorTexasWeak(p *playerState) (Call, meaning) {
	if last, _, ok := e.lastBid(); ok && last == bid(2, SNoTrump) {
		c := bid(3, SClubs)
		if e.legal(p.seat, c) {
			return c, m(0, 7, "revient à 3T, arrêt absolu", "corrects back to 3C, an absolute sign-off")
		}
	}
	return passCall, m(0, 7, "arrêt, jeu faible", "sign-off, weak hand")
}

// afterMinorTexasGame concludes a game-going minor Texas by announcing the
// shape found at the time of the ask: a 5-5 two-suiter is shown naturally
// with 3D; otherwise the singleton is announced by the "meilleur résidu" --
// naming the major NOT held short -- or by 3NT for a short in the other
// minor.
func (e *Engine) afterMinorTexasGame(p *playerState, primary Suit, twoSuited bool) (Call, meaning) {
	h := p.hand
	if twoSuited {
		c := bid(3, SDiamonds)
		if e.legal(p.seat, c) {
			e.gameForce[sideOf(p.seat)] = true
			return c, m(10, 40, "bicolore 5T-5K, forcing de manche", "5-5 clubs-diamonds two-suiter, game forcing").withLen(Diamonds, 5).asForcing()
		}
	}
	short, ok := splinterSuit(h, primary)
	if !ok {
		c := bid(3, SNoTrump)
		return c, m(10, 40, "3SA, pas de singleton trouvé", "3NT, no singleton found")
	}
	otherMinor := Diamonds
	if primary == Diamonds {
		otherMinor = Clubs
	}
	switch short {
	case Spades:
		c := bid(3, SHearts)
		if e.legal(p.seat, c) {
			return c, m(10, 40, "singleton/chicane à Pique (meilleur résidu)", "singleton/void in spades (best residual)").withShort(Spades, h.Len(Spades)).asForcing()
		}
	case Hearts:
		c := bid(3, SSpades)
		if e.legal(p.seat, c) {
			return c, m(10, 40, "singleton/chicane à Cœur (meilleur résidu)", "singleton/void in hearts (best residual)").withShort(Hearts, h.Len(Hearts)).asForcing()
		}
	case otherMinor:
		c := bid(3, SNoTrump)
		if e.legal(p.seat, c) {
			// 3SA is itself the game: partner may pass it, so the bid is
			// descriptive, not forcing.
			return c, m(10, 40, "singleton/chicane dans l'autre mineure", "singleton/void in the other minor").withShort(otherMinor, h.Len(otherMinor))
		}
	}
	return passCall, noInfo()
}

func (e *Engine) afterTransfer(p *playerState, M Suit, oMin int, completion Call) (Call, meaning) {
	h := p.hand
	hp := h.H()
	// Opener super-accepted (a jump above the plain completion: four trumps,
	// maximum): the nine-card fit makes game good with any few points. The
	// plain completion itself is mandatory and promises nothing extra, and in
	// a contested auction a raised completion may simply have been pushed up
	// over the interference, so only an unforced jump qualifies.
	if last, lastSeat, ok := e.lastBid(); ok && lastSeat == partnerOf(p.seat) &&
		last.Strain == M.Strain() && last.higherThan(completion) &&
		oMin < 20 && e.uncontested(p.seat) {
		// With a slam in view [S-1], the super-accept is where the exploration
		// starts: it stops at three of the major and leaves every step up to
		// the game free, so the controls cost nothing there, whereas every
		// answer to 4NT already sits above 4M. The exchange moves on to
		// Blackwood by itself once the controls are out [S-4].
		partner := e.ps[partnerOf(p.seat)]
		if cMin, cMax := h.HLD(M)+partner.shownMin, h.HLD(M)+partner.shownMax; cMin >= 29 && (cMax >= 33 || cMin >= 31) {
			if c, mn, ok := e.initiateControls(p, M); ok {
				return c, mn
			}
			if c, mn := e.conclude(p); c.Kind != KindPass {
				return c, mn
			}
		}
		if hp >= 5 {
			return bidSuit(4, M), m(5, -1, "conclusion à la manche après la rectification à saut", "game after the super-accept").withLen(M, 5)
		}
		return passCall, m(0, 4, "arrêt, jeu trop faible malgré le fit", "sign-off, too weak even with the fit")
	}
	if oMin >= 20 { // facing 2NT/2C strong: thresholds shrink
		switch {
		case hp <= 3:
			return passCall, m(0, 3, "jeu trop faible pour la manche", "too weak for game")
		case hp <= 10 && h.Len(M) >= 6:
			return bidSuit(4, M), m(4, 10, "conclusion à la manche, 6 atouts", "game with a six-card suit").withLen(M, 6)
		case hp <= 10:
			return bid(3, SNoTrump), m(4, 10, "3SA, l'ouvreur choisit avec 3 atouts", "3NT, opener corrects with 3-card support")
		default:
			// 11+ opposite a 20-21 opener already reaches slam-zone combined
			// values -- a six-card suit only reinforces it, never a reason to
			// sign off in game. Let the generic engine explore
			// keycards/quantitative slam instead of settling for a flat game
			// regardless of strength.
			return e.conclude(p)
		}
	}
	switch {
	case hp >= 5 && hp <= 9 && h.Len(M) >= 6:
		return bidSuit(3, M), m(5, 9, "proposition de manche, 6 atouts", "game invitation, six trumps").withLen(M, 6).asInvite()
	case hp <= 7:
		return passCall, m(0, 7, "arrêt, jeu faible", "sign-off, weak hand")
	case hp <= 9:
		return bid(2, SNoTrump), m(8, 9, "2SA, 8-9H, proposition de manche", "2NT, 8-9, game invitation").asInvite()
	case hp <= 15:
		if h.Len(M) >= 6 {
			return bidSuit(4, M), m(10, 15, "conclusion à la manche, 6 atouts", "game with a six-card suit").withLen(M, 6)
		}
		return bid(3, SNoTrump), m(10, 15, "3SA, l'ouvreur choisit avec 3 atouts", "3NT, opener corrects with 3-card support")
	default:
		mn := m(16, 40, "4SA quantitatif", "quantitative 4NT")
		mn.slamInvite = true
		return bid(4, SNoTrump), mn
	}
}

// afterStayman dispatches on which of the four Stayman answers opener gave
// (docs/bidings.md, "Développements après la réponse au Stayman"). Answers
// sit at level 2 over a 1SA opening (base 1) and level 3 over a 2SA opening
// (base 2) -- the same conventions apply, shifted up one level.
func (e *Engine) afterStayman(p *playerState, oMin int) (Call, meaning) {
	base, askLevel := 1, 2
	if oMin >= 20 {
		base, askLevel = 2, 3
	}
	last, lastSeat, _ := e.lastBid()
	if lastSeat != partnerOf(p.seat) {
		return e.conclude(p)
	}
	switch last {
	case bid(askLevel, SDiamonds):
		return e.afterStaymanDenial(p, base, askLevel)
	case bid(askLevel, SHearts):
		return e.afterStaymanOneMajor(p, base, askLevel, Hearts)
	case bid(askLevel, SSpades):
		return e.afterStaymanOneMajor(p, base, askLevel, Spades)
	case bid(askLevel, SNoTrump):
		return e.afterStaymanBothMajors(p, base, askLevel)
	}
	return e.conclude(p)
}

// afterStaymanDenial continues after opener denies both four-card majors
// (2D for base 1, 3D for base 2): a 4-5 (or 5-4) two-suited major hand
// announces itself via the "chassé-croisé" -- naming the major held with
// exactly four cards promises five in the other one, game forcing -- a good
// 5+-card minor is forcing to game (staymanNewMinor), and everything else
// follows the same notrump ladder as a direct response to the opening.
func (e *Engine) afterStaymanDenial(p *playerState, base, askLevel int) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	lh, ls := h.Len(Hearts), h.Len(Spades)

	if hl >= 11 {
		switch {
		case lh == 4 && ls >= 5:
			c := bid(askLevel+1, SHearts)
			if e.legal(p.seat, c) {
				mn := m(11, 40, "chassé-croisé : 4 cartes à Cœur et 5 cartes à Pique, forcing de manche",
					"crisscross: four hearts and five spades, game forcing").withLen(Hearts, 4).withLen(Spades, 5).asForcing()
				e.gameForce[sideOf(p.seat)] = true
				return c, mn
			}
		case ls == 4 && lh >= 5:
			c := bid(askLevel+1, SSpades)
			if e.legal(p.seat, c) {
				mn := m(11, 40, "chassé-croisé : 4 cartes à Pique et 5 cartes à Cœur, forcing de manche",
					"crisscross: four spades and five hearts, game forcing").withLen(Spades, 4).withLen(Hearts, 5).asForcing()
				e.gameForce[sideOf(p.seat)] = true
				return c, mn
			}
		}
	}

	if c, mn, ok := e.staymanNewMinor(p, askLevel); ok {
		return c, mn
	}
	e.tr.note("après le Stayman, pas de fit majeur à jouer : réponse selon les points", "after Stayman, no major fit to play: response by points")
	return e.ntResponseLadder(p, base)
}

// staymanNewMinor implements the change of suit shared by every Stayman
// answer (3C/3D over a 1SA-level answer, 4C/4D over a 2SA-level one): a good
// 5+-card minor, forcing to game, denying a major fit and promising a
// singleton or void -- except with slam values (16HL+), where the
// shortness requirement is waived (docs/bidings.md).
func (e *Engine) staymanNewMinor(p *playerState, askLevel int) (Call, meaning, bool) {
	h := p.hand
	hl := h.HL()
	if hl < 11 {
		return Call{}, meaning{}, false
	}
	for _, s := range []Suit{Clubs, Diamonds} {
		if h.Len(s) < 5 {
			continue
		}
		short, hasShort := splinterSuit(h, s)
		slamHand := hl >= 16
		if !hasShort && !slamHand {
			continue
		}
		c := bid(askLevel+1, s.Strain())
		if !e.legal(p.seat, c) {
			continue
		}
		var fr, en string
		if hasShort {
			fr = "couleur cinquième à " + suitNameFR[s] + ", forcing de manche, singleton/chicane à " + suitNameFR[short]
			en = "five-card " + suitNameEN[s] + ", game forcing, singleton/void in " + suitNameEN[short]
		} else {
			fr = "couleur cinquième à " + suitNameFR[s] + ", forcing de manche, main de chelem"
			en = "five-card " + suitNameEN[s] + ", game forcing, slam hand"
		}
		mn := m(11, 40, fr, en).withLen(s, h.Len(s)).asForcing()
		if hasShort {
			mn = mn.withShort(short, h.Len(short))
		}
		return c, mn, true
	}
	return Call{}, meaning{}, false
}

// afterStaymanOneMajor continues after opener shows exactly one four-card
// major M (2H/2S for base 1, 3H/3S for base 2): with support, a splinter
// (15-17 HLD, singleton/void) or the "convention 2012" (18+ HLD, artificial
// 3-of-the-other-major) take priority over a plain natural, non-forcing
// raise; without support, a good 5+-card minor is forcing to game
// (staymanNewMinor), else the same notrump ladder as a direct response
// (docs/bidings.md).
func (e *Engine) afterStaymanOneMajor(p *playerState, base, askLevel int, M Suit) (Call, meaning) {
	h := p.hand
	other := Hearts
	if M == Hearts {
		other = Spades
	}

	if h.Len(M) >= 4 {
		hld := h.HLD(M)
		// The slam zone is a combined count [E-9], not a fixed number of
		// points: facing the 15-17 opening it starts at 18 HLD, but facing a
		// 2SA opening (20-21) at 13, and facing the 22-23 rebid of a strong
		// 2C at 11. Anchoring it on the floor the opening promised keeps the
		// slam-ambition bid and the splinter one notch below it in their
		// place whatever the opening was, instead of raising to game with a
		// hand the pair's own count puts past 33.
		slamZone := 33 - e.ps[partnerOf(p.seat)].shownMin
		if hld >= slamZone-3 && hld < slamZone {
			if short, ok := splinterSuit(h, M); ok {
				c := bid(askLevel+2, short.Strain())
				if e.legal(p.seat, c) {
					kind, kindEN := "singleton", "a singleton"
					if h.Len(short) == 0 {
						kind, kindEN = "chicane", "a void"
					}
					zone := fmt.Sprintf("%d-%dHLD", slamZone-3, slamZone-1)
					fr := "Splinter après Stayman : fit " + suitNameFR[M] + ", " + kind + " à " + suitNameFR[short] + ", " + zone
					en := "splinter after Stayman: " + suitNameEN[M] + " fit, " + kindEN + " in " + suitNameEN[short] + ", " + zone
					mn := m(slamZone-3, slamZone-1, fr, en).withLen(M, 4).withShort(short, h.Len(short)).asForcing()
					mn.splinter = true
					e.gameForce[sideOf(p.seat)] = true
					return c, mn
				}
			}
		}
		if hld >= slamZone {
			c := bid(askLevel+1, other.Strain())
			// Over a 2SA base the answer already sits at the three level, so
			// the artificial bid lands on the game itself (3C - 3H - 4P) or
			// one rung under it (3C - 3P - 4C), where it leaves partner
			// nothing to cue below the game. Both waste the room the generic
			// machinery still has: hand the slam hand over to the
			// control/keycard road instead of burning it.
			if askLevel >= 3 {
				return e.conclude(p)
			}
			if e.legal(p.seat, c) {
				mn := m(slamZone, 40, "convention 2012 : fit à "+suitNameFR[M]+", ambition de chelem",
					"2012 convention: "+suitNameEN[M]+" fit, slam ambition").withLen(M, 4).asForcing()
				e.gameForce[sideOf(p.seat)] = true
				return c, mn
			}
		}
		if hld >= 10 {
			// Game in a major is always four of it, never askLevel+2: that
			// formula lands on 4M over a 1SA opening (askLevel 2) but overshoots
			// to 5M over a 2SA opening (askLevel 3), needlessly bypassing the
			// making game.
			c := bidSuit(4, M)
			if e.legal(p.seat, c) {
				return c, m(10, 40, "conclusion à la manche dans le fit majeur, naturel non forcing", "raise to game in the major fit, natural non-forcing").withLen(M, 4)
			}
		}
		c := bidSuit(askLevel+1, M)
		if e.legal(p.seat, c) {
			// Over a 1SA base this raise is the 3M invitation (6-9); over a
			// 2SA base askLevel+1 is the game itself, which the fit accepts
			// on next to nothing facing 21-22 — and on strictly nothing when
			// the 2SA was a strong 2C/2D opener's rebid, the responder being
			// forced to answer whatever his count.
			if askLevel >= 3 {
				lo := 3
				if e.openCall == bid(2, SClubs) || e.openCall == bid(2, SDiamonds) {
					lo = 0
				}
				return c, m(lo, 9, "conclusion à la manche dans le fit majeur", "raise to game in the major fit").withLen(M, 4)
			}
			return c, m(6, 9, "soutien naturel non forcing", "natural non-forcing raise").withLen(M, 4)
		}
	}

	if c, mn, ok := e.staymanNewMinor(p, askLevel); ok {
		return c, mn
	}
	e.tr.note("après le Stayman, pas de fit majeur à jouer : réponse selon les points", "after Stayman, no major fit to play: response by points")
	return e.ntResponseLadder(p, base)
}

// afterStaymanBothMajors continues after opener shows both four-card
// majors (2SA for base 1, 3SA for base 2): pick whichever major responder
// prefers (the longer one, spades on a tie), then choose between naming the
// major itself for slam ambition (18HL+), a direct game raise (16-17HL, the
// relay minor one level higher), or the plain transfer that lets opener
// rectify to 3 or 4 depending on strength (docs/bidings.md, "Les
// transferts").
func (e *Engine) afterStaymanBothMajors(p *playerState, base, askLevel int) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	target := Spades
	if h.Len(Hearts) > h.Len(Spades) {
		target = Hearts
	}
	relay := Clubs
	if target == Spades {
		relay = Diamonds
	}

	// As after a single major [E-9], the slam zone is the combined count, not
	// a fixed 18 HL: facing the 22-23 rebid of a strong 2C the ambition
	// starts at 11, and the transfer that merely proposes game would
	// otherwise bury a hand the pair's own count puts well past 33.
	slamZone := 33 - e.ps[partnerOf(p.seat)].shownMin
	switch {
	case hl >= slamZone:
		c := bidSuit(askLevel+1, target)
		if e.legal(p.seat, c) {
			mn := m(slamZone, 40, "fit à "+suitNameFR[target]+", ambition de chelem", suitNameEN[target]+" fit, slam ambition").withLen(target, 4).asForcing()
			e.gameForce[sideOf(p.seat)] = true
			return c, mn
		}
	case hl >= slamZone-2:
		c := bid(askLevel+2, relay.Strain())
		if e.legal(p.seat, c) {
			mn := m(slamZone-2, 40, "fit à "+suitNameFR[target]+", conclusion à la manche", suitNameEN[target]+" fit, bids game").withLen(target, 4)
			mn.hasTexas, mn.texas = true, target
			return c, mn
		}
	default:
		c := bid(askLevel+1, relay.Strain())
		if e.legal(p.seat, c) {
			// Over a 1SA base the Stayman already promised 8+; over 2SA it
			// only took a few points, so the transfer floor follows the base —
			// and drops to zero when the 2SA was a strong 2C/2D opener's
			// rebid, the responder being forced whatever his count.
			lo := 8
			if base >= 2 {
				lo = 3
				if e.openCall == bid(2, SClubs) || e.openCall == bid(2, SDiamonds) {
					lo = 0
				}
			}
			mn := m(lo, 17, "transfert, propose la manche à "+suitNameFR[target]+", l'ouvreur rectifie selon sa force",
				"transfer, proposes game in "+suitNameEN[target]+", opener corrects by strength").asForcing()
			mn.staymanBothMajorsRelay, mn.texas = true, target
			return c, mn
		}
	}
	e.tr.note("après le Stayman, pas de fit majeur à jouer : réponse selon les points", "after Stayman, no major fit to play: response by points")
	return e.ntResponseLadder(p, base)
}

// staymanBothMajorsAnswer rectifies the game-proposing transfer after
// Stayman confirmed both majors (docs/bidings.md, "Les transferts"): opener
// completes to the cheapest level with a minimum, or jumps to game with a
// maximum.
func (e *Engine) staymanBothMajorsAnswer(p *playerState, target Suit) (Call, meaning) {
	c := e.cheapestCall(target.Strain())
	if p.hand.H() >= p.shownMax {
		jump := bid(c.Level+1, target.Strain())
		if e.legal(p.seat, jump) {
			return jump, m(-1, -1, "rectification à saut, maximum", "jump correction, maximum").withLen(target, 4)
		}
	}
	if e.legal(p.seat, c) {
		return c, m(-1, -1, "rectification du transfert", "completes the transfer").withLen(target, 4)
	}
	return passCall, noInfo()
}

// hasMisereDoreeSingleton reports whether h holds a genuine singleton (not a
// void) in one of the three suits other than the five-card major M, the
// shape test for the "misère dorée" (docs/addon_11.md).
func hasMisereDoreeSingleton(h *Hand, M Suit) bool {
	for s := Clubs; s <= Spades; s++ {
		if s != M && h.Len(s) == 1 {
			return true
		}
	}
	return false
}

// afterStaymanMisereDoree continues the "misère dorée" (Bridgeur n°813,
// J-P. Desmoulins, docs/addon_11.md): a limited hand (7-8H) with a genuine
// singleton and a five-card major, routed through Stayman instead of Texas
// to locate a nine-card fit before committing to game.
func (e *Engine) afterStaymanMisereDoree(p *playerState, M Suit) (Call, meaning) {
	last, lastSeat, hasBid := e.lastBid()
	if !hasBid || lastSeat != partnerOf(p.seat) {
		return e.conclude(p)
	}
	other := Hearts
	if M == Hearts {
		other = Spades
	}
	relay := Clubs
	if M == Spades {
		relay = Diamonds
	}

	// Opener shows the same major: the nine-card fit is worth game on its
	// own, no further judgment needed.
	if last == bid(2, M.Strain()) {
		c := bidSuit(4, M)
		if e.legal(p.seat, c) {
			return c, m(7, 8, "conclusion à la manche, fit neuvième", "bidding game, nine-card fit").withLen(M, 5)
		}
	}
	// Opener shows both majors 4-4: the fit in M is guaranteed too, but bid
	// game through the minor-suit Texas so opener remains declarer.
	if last == bid(2, SNoTrump) {
		c := bid(4, relay.Strain())
		if e.legal(p.seat, c) {
			mn := m(7, 8, "Texas pour "+suitNameFR[M]+", fit connu", "Texas for "+suitNameEN[M]+", known fit").withLen(M, 5)
			mn.hasTexas, mn.texas = true, M
			return c, mn
		}
	}
	// Otherwise opener denies M specifically (2D denying both majors, or
	// showing the other major): show the misère dorée directly, at the same
	// level, when M still ranks high enough to be available. When M ranks
	// below the major opener just named (e.g. opener's 2S over our five
	// hearts), 2SA is the only spot left below the three level.
	if last == bid(2, SDiamonds) || last == bid(2, other.Strain()) {
		c := bid(2, M.Strain())
		if e.legal(p.seat, c) {
			mn := m(7, 8, "Stayman misère dorée avec 5 "+suitNameFR[M]+", 7-8H et un singleton",
				"golden-misery Stayman, five "+suitNameEN[M]+", 7-8H and a singleton").withLen(M, 5).asInvite()
			return c, mn
		}
		c = bid(2, SNoTrump)
		if e.legal(p.seat, c) {
			mn := m(7, 8, "misère dorée à "+suitNameFR[M]+", proposition de manche", "golden-misery in "+suitNameEN[M]+", game proposal").withLen(M, 5)
			mn.misereDoreeAsk, mn.misereDoreeSuit = true, M
			return c, mn
		}
	}
	return e.conclude(p)
}

func (e *Engine) respondAfter2C(p *playerState) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	op := e.ps[partnerOf(p.seat)]
	sc, _ := e.lastCallBy(op.seat)
	if sc.Call == bid(2, SNoTrump) {
		e.tr.note("2♣ fort puis 2SA du partenaire (22-23 H) : on répond comme à une ouverture de 2SA", "strong 2♣ then 2NT from partner (22-23 H): respond as to a 2NT opening")
		return e.respondNT(p, 2, 22)
	}
	if sc.Call.IsBid() && sc.Call.Strain <= SSpades {
		os := Suit(sc.Call.Strain)
		sup := h.Len(os)
		if sup >= 3 {
			if hl <= 7 {
				if os.IsMajor() {
					return bidSuit(4, os), m(0, 7, "conclusion à la manche, fit sans valeur extérieure", "game, fit without outside values").withLen(os, 3)
				}
				// Minor fit. In a major the whole 0-7 zone lands on the same
				// bid because that bid is the game; in a minor the game is a
				// level higher, so the same raise leaves the opener guessing
				// across seven points -- and he needs about six of them to
				// bring in the eleventh trick. The zone is split so he does
				// not have to guess: with five and up, the responder bids the
				// game himself, the opener having announced nine tricks in
				// his suit [O-2].
				// The five points that buy the game are honour points: what
				// the opener is missing is two tricks, and length in a side
				// suit facing a one-suited hand rarely provides one.
				if h.H() >= 5 {
					if c := bidSuit(5, os); e.legal(p.seat, c) {
						return c, m(5, 7, "la manche mineure : le fit et de quoi fournir la onzième levée", "the minor game: the fit and enough for the eleventh trick").withLen(os, 3)
					}
				}
				return e.cheapestCall(os.Strain()), m(0, 4, "soutien minimum, moins de 5 points d'honneurs", "minimum raise, fewer than five honour points").withLen(os, 3)
			}
			// Three of the suit, or four when the opener has already used the
			// three level himself: the raise is the same bid either way, and
			// leaving it to the illegal-bid net [Z-1] to find the level would
			// make a rule out of a safety net.
			return e.cheapestCall(os.Strain()), m(8, 40, "soutien, espoir de chelem", "raise, slam interest").withLen(os, 3).asForcing()
		}
		if s, n := h.Longest(), h.Len(h.Longest()); n >= 5 && hl >= 5 && s != os {
			return e.cheapestCall(s.Strain()), m(5, 40, "couleur cinquième, 5H et plus", "five-card suit, 5+ points").withLen(s, 5).asForcing()
		}
		// The waiting 2NT is a 0-7 bid: it says "nothing to describe", and
		// the opener re-evaluates against that ceiling. Two hands must not
		// use it. One is the hand of 8 HL and up: announcing a ceiling it
		// does not have is what leaves the opener rebidding his suit as a
		// minimum, and 8 opposite the 18 the opening promises already reaches
		// the notrump game. The other is any hand whose opener has taken the
		// three level, where there is nothing left to wait for -- his
		// six-card suit is worth about nine tricks on its own [O-2].
		//
		// Both name 3SA instead: without a fit and without a suit of our own,
		// it is the game that needs the fewest tricks, and a raise on two
		// cards would stop below every game there is.
		waiting := e.cheapestCall(SNoTrump)
		if waiting.Level == 2 && hl <= 7 {
			return waiting, m(0, 7, "enchère d'attente", "waiting bid").asForcing()
		}
		if e.ntSafe(p) {
			if waiting.Level == 2 {
				// 8-11, and the ceiling matters as much as the floor: left
				// open, an eight-count reads like a possible fifteen and the
				// opener starts hunting a slam that is not there. The hand
				// that really holds more has no way to say so here -- the
				// slam road through this sequence is the fitted raise, which
				// is forcing and unlimited.
				return bid(3, SNoTrump), m(8, 11,
					"3SA : 8-11 HL face à l'ouverture de 2T, la manche y est",
					"3NT: 8-11 HL opposite the strong 2C opening, the game is there")
			}
			return bid(3, SNoTrump), m(-1, -1,
				"3SA : l'ouverture annonce neuf levées, je place la manche",
				"3NT: the opening announced nine tricks, placing the game")
		}
		return e.cheapestCall(os.Strain()), m(-1, -1,
			"soutien faute de mieux, Sans-Atout n'est pas jouable",
			"support for want of anything better, notrump is unsafe").withLen(os, sup)
	}
	return e.conclude(p)
}

func (e *Engine) respondAces(p *playerState) (Call, meaning) {
	p.answeredAces = true
	h := p.hand
	hp := h.H()
	aces := []Suit{}
	for s := Clubs; s <= Spades; s++ {
		if h.HasCard(s, 'A') {
			aces = append(aces, s)
		}
	}
	mk := func(c Call, min, max int, fr, en string) (Call, meaning) {
		mn := m(min, max, fr, en).asForcing()
		return c, mn
	}
	switch len(aces) {
	case 0:
		p.acesShown, p.acesExact = 0, true
		short := h.sortedLens()[3] <= 1
		if hp >= 8 && !short {
			return mk(bid(2, SNoTrump), 8, 40, "2SA : pas d'As, 8H et plus, pas de courte", "2NT: no ace, 8+ points, no short suit")
		}
		// The step covers two hands, not one (docs/bidings.md, "L'OUVERTURE DE
		// 2♦") : "Pas d'As, 0-7H, ou 8H et plus avec une courte". Capping it
		// at 7 would be a lie about the second kind, and an irreversible one —
		// a later bid can raise the floor but never lift a ceiling already
		// announced, so the pair could no longer count its way to a slam.
		return mk(bid(2, SHearts), 0, 40,
			"2C : pas d'As — 0-7H, ou 8H et plus avec une courte",
			"2H: no ace — 0-7, or 8+ with a short suit")
	case 1:
		p.acesShown, p.acesExact = 1, true
		switch aces[0] {
		case Hearts, Spades:
			return mk(bid(2, SSpades), 4, 40, "2P : l'As de Cœur ou de Pique", "2S: the ace of hearts or spades")
		case Clubs:
			return mk(bid(3, SClubs), 4, 40, "3T : l'As de Trèfle", "3C: the ace of clubs")
		default:
			return mk(bid(3, SDiamonds), 4, 40, "3K : l'As de Carreau", "3D: the ace of diamonds")
		}
	case 2:
		sameColor := (aces[0] == Clubs && aces[1] == Spades) || (aces[0] == Diamonds && aces[1] == Hearts)
		sameRank := (aces[0] == Clubs && aces[1] == Diamonds) || (aces[0] == Hearts && aces[1] == Spades)
		switch {
		case sameColor:
			p.acesShown, p.acesExact = 2, true
			return mk(bid(3, SHearts), 8, 40, "3C : deux As de même couleur", "3H: two aces of the same colour")
		case sameRank:
			p.acesShown, p.acesExact = 2, true
			return mk(bid(3, SSpades), 8, 40, "3P : deux As de même rang", "3S: two aces of the same rank")
		default:
			// 3SA also covers the three- and four-ace hands below: from
			// partner's seat the step only guarantees "two or more".
			p.acesShown, p.acesExact = 2, false
			return mk(bid(3, SNoTrump), 8, 40, "3SA : deux As mélangés", "3NT: two mixed aces")
		}
	default:
		p.acesShown, p.acesExact = 2, false
		return mk(bid(3, SNoTrump), 12, 40, "deux As et plus", "two or more aces")
	}
}

// respondMinorAffranchie answers the 3NT opening (docs/bidings.md,
// "L'OUVERTURE DE 3SA"): a solid seven-card minor, unknown to responder --
// only the opener knows whether it is clubs or diamonds. Passing is right
// the overwhelming majority of the time: the opening already names the best
// available contract, and the responder has no way to add tricks without
// real extra playing strength. With genuine controls plus an outside
// shortness (a ruff wherever the true suit ends up), responder can look
// beyond 3NT: he names clubs at his target level -- the level-safe anchor,
// "I want to play here, whichever minor it is". Opener passes if his suit
// is clubs, or corrects to diamonds at the very same level if it is
// diamonds (the docs' "Principle": naming clubs never risks an extra
// level). Anchoring on diamonds instead -- deliberately accepting a jump
// specifically when opener's suit turns out to be clubs, as in the docs'
// second worked example -- is a judgment call the engine does not attempt
// to model; it always takes the safe route.
func (e *Engine) respondMinorAffranchie(p *playerState) (Call, meaning) {
	h := p.hand
	short := false
	for s := Clubs; s <= Spades; s++ {
		if h.Len(s) <= 1 {
			short = true
			break
		}
	}
	if h.H() < 14 || !short {
		return passCall, m(-1, -1, "passe, meilleur contrat", "pass, best contract")
	}
	level := 4
	if h.H() >= 17 {
		level = 5
	}
	c := bid(level, SClubs)
	if !e.legal(p.seat, c) {
		return passCall, m(-1, -1, "passe, meilleur contrat", "pass, best contract")
	}
	mn := m(-1, -1,
		"essai au-delà de 3SA, ancre sur Trèfle : rectifiez à Carreau si c'est votre couleur",
		"try beyond 3NT, anchored on clubs: correct to diamonds if that is your suit")
	return c, mn
}

// affranchieRebid answers responder's try beyond the 3NT solid-minor
// opening: pass when the suit responder anchored on already matches ours,
// else correct to the real minor at the cheapest legal level (same level if
// the real suit outranks the anchor -- diamonds over clubs -- one level
// higher if it does not).
func (e *Engine) affranchieRebid(p *playerState) (Call, meaning) {
	last, _, hasBid := e.lastBid()
	trueSuit, ok := sevenCardSolidMinor(p.hand)
	if !ok || !hasBid || last.Strain > SDiamonds {
		return passCall, noInfo()
	}
	if Suit(last.Strain) == trueSuit {
		return passCall, m(-1, -1, "passe, couleur trouvée", "pass, matches the real suit")
	}
	c := bid(last.Level, trueSuit.Strain())
	if !c.higherThan(last) {
		c = bid(last.Level+1, trueSuit.Strain())
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	mn := m(-1, -1, "rectification vers la mineure réelle", "corrects to the real minor").withLen(trueSuit, 7)
	mn.affranchieCorrection = true
	return c, mn
}

// respondWeak2 covers the responder's table over a weak 2H/2S opening
// (docs/addon_7.md). Four trumps or more is "attaque-défense": always raise
// straight to game, whatever the point count, since the same call doubles
// as a law-of-total-tricks sacrifice and as a genuine game-going hand. With
// only 2-3 trumps, the point count decides: 18-20 HLD is game with no slam
// interest (bid it directly), while 15-17 (game possible) or 21+ (slam
// possible) go through the 2NT relay-and-fit-ask instead.
func (e *Engine) respondWeak2(p *playerState, M Suit) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	sup := h.Len(M)
	hld := h.HLD(M)
	switch {
	case sup >= 4:
		return bidSuit(4, M), m(0, 40, "soutien à la manche, attaque-défense (4 atouts, quel que soit le nombre de points)", "raise to game, attack-or-defence (four trumps, regardless of point count)").withLen(M, 4)
	case sup >= 2 && hld >= 18 && hld <= 20:
		return bidSuit(4, M), m(18, 20, "manche directe en attaque, 18-20HLD", "direct raise to game, 18-20 HLD").withLen(M, sup)
	case sup >= 2 && (hld >= 21 || (hld >= 15 && hld <= 17)):
		return bid(2, SNoTrump), m(15, 40, "2SA relais-fitté, espoir de manche ou de chelem", "2NT relay-and-fit-ask, game or slam interest").asForcing()
	case hl >= 13 && func() bool {
		for _, s := range []Suit{Spades, Hearts} {
			if s != M && h.GoodSuit(s) {
				return true
			}
		}
		return false
	}():
		var s Suit
		if M == Hearts {
			s = Spades
		} else {
			s = Hearts
		}
		return e.cheapestCall(s.Strain()), m(13, 40, "changement de couleur, forcing", "new suit, forcing").withLen(s, 5).asForcing()
	}
	// Rencontre: jump one level beyond the cheapest minor-suit reply,
	// support for the weak two and a good 5+ card minor, 8-11 HCP
	// (docs/addon_6.md), in anticipation of high-level competitive bidding.
	if sup == 3 {
		for _, s := range []Suit{Clubs, Diamonds} {
			if c, mn, ok := e.rencontreCall(p, M, s, 3, 1); ok {
				return c, mn
			}
		}
		return bidSuit(3, M), m(0, 14, "prolongement du barrage", "extending the preempt").withLen(M, 3)
	}
	if hl >= 16 && h.IsRegular() {
		return bid(3, SNoTrump), m(16, 40, "conclusion à 3SA", "3NT to play")
	}
	return passCall, m(0, 14, "", "")
}

// tryRencontre builds the rencontre call in candidate unless it lands
// exactly at a major's own game level, which stays natural, to play
// (docs/addon_6.md, section III exception). A rencontre in a minor is
// capped at the four level: the message is the fit plus the side suit,
// 8-11 H, and it is fully delivered by 4m -- pushing it to 5m sells a
// level the camp may need, and lands beyond the major game the fit is
// most often heading for.
func (e *Engine) tryRencontre(p *playerState, fit, candidate Suit, minFit, jumpLevels int) (Call, meaning, bool) {
	base := e.cheapestCall(candidate.Strain())
	if !candidate.IsMajor() && base.Level+jumpLevels > 4 {
		jumpLevels = 4 - base.Level
		if jumpLevels < 1 {
			return Call{}, meaning{}, false // no room left for a jump
		}
	}
	c := bid(base.Level+jumpLevels, candidate.Strain())
	if candidate.IsMajor() && c == bidSuit(4, candidate) {
		return Call{}, meaning{}, false
	}
	return e.rencontreCall(p, fit, candidate, minFit, jumpLevels)
}

// competitiveRencontre looks for a rencontre jump over adverse interference
// (docs/addon_6.md, sections II-III). After a double, a minor opening only
// offers the other minor (a jump into a major stays a natural weak barrage);
// a major opening offers any suit. After an overcall, the rencontre needs a
// double jump, except when a weak two-level major intervention over a minor
// opening already sits high enough that a single jump suffices.
func (e *Engine) competitiveRencontre(p *playerState, os Suit, doubled bool, interference Call) (Call, meaning, bool) {
	const minFit = 4
	if doubled && !os.IsMajor() {
		other := Diamonds
		if os == Diamonds {
			other = Clubs
		}
		return e.tryRencontre(p, os, other, minFit, 1)
	}

	jumpLevels := 1
	if !doubled {
		jumpLevels = 2
		if !os.IsMajor() && interference.IsBid() && interference.Level == 2 &&
			interference.Strain <= SSpades && Suit(interference.Strain).IsMajor() {
			jumpLevels = 1
		}
	}
	for _, s := range []Suit{Clubs, Diamonds, Hearts, Spades} {
		if s == os {
			continue
		}
		if interference.IsBid() && s.Strain() == interference.Strain {
			continue // the opponents' own suit
		}
		if c, mn, ok := e.tryRencontre(p, os, s, minFit, jumpLevels); ok {
			return c, mn, true
		}
	}
	return Call{}, meaning{}, false
}

// respondStrongTwoInterference answers a defender's intervention over the
// strong, artificial 2C/2D opening before the forced relay (docs/bidings.md,
// "LA DÉFENSE APRÈS UNE INTERVENTION DU N°2"): every suit bid is fully
// natural here (at least five good cards, 5H and up) since the opening
// itself never promised anything about its own suit; the double shows the
// same 5H floor without a good natural bid available.
func (e *Engine) respondStrongTwoInterference(p *playerState) (Call, meaning) {
	h := p.hand
	if h.H() < 5 {
		return passCall, m(0, 4, "moins de 5H, pas d'enchère naturelle", "fewer than 5H, no natural bid")
	}
	var rho Call
	if len(e.calls) > 0 {
		rho = e.calls[len(e.calls)-1].Call
	}
	best, bestLen := Suit(-1), 0
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if rho.IsBid() && s == Suit(rho.Strain) {
			continue // the opponents' own suit
		}
		if h.Len(s) >= 5 && h.GoodSuit(s) && h.Len(s) > bestLen {
			best, bestLen = s, h.Len(s)
		}
	}
	if bestLen > 0 {
		c := e.cheapestCall(best.Strain())
		if e.legal(p.seat, c) {
			// The opening's own auto-forcing nature carries through: at least
			// 18H (2C) or 24+ HL (2D) opposite, opener must not let this die.
			return c, m(5, 40, "naturelle, 5 belles cartes et 5H et plus", "natural, five good cards and 5H+").withLen(best, bestLen).asForcing()
		}
	}
	if e.legal(p.seat, doubleCall) {
		return doubleCall, m(5, 40, "Contre, 5H et plus sans bonne enchère naturelle", "double, 5H+ with no good natural bid").asForcing()
	}
	return passCall, m(0, 4, "pas d'enchère naturelle disponible", "no natural bid available")
}

// respondCompetitive simplifies responses when the defenders interfered.
func (e *Engine) respondCompetitive(p *playerState) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	oc := e.openCall

	if oc == bid(1, SNoTrump) {
		if suits := e.opponentSuits(p); len(suits) == 1 {
			return e.respondRubensohl(p, suits[0])
		}
	}

	// An unbid major comes before any support of the opening [Rm-2, RC-1b]:
	// supporting would deny it, and the fit most often lies there. Five cards
	// are named naturally; exactly four go through the Spoutnik double
	// [RC-6] -- which is why the natural bid promises five as soon as an
	// overcall has left the double free to say it. Supporting partner's major
	// with three cards comes first: that fit is already known.
	if oc.Strain <= SSpades {
		os := Suit(oc.Strain)
		var oppBid [4]bool
		for _, s := range e.opponentSuits(p) {
			oppBid[s] = true
		}
		overcalled := false
		if n := len(e.calls); n > 0 {
			rho := e.calls[n-1]
			overcalled = rho.Call.IsBid() && rho.Call.Strain <= SSpades && sideOf(rho.Seat) != sideOf(p.seat)
		}
		if !(os.IsMajor() && h.Len(os) >= 3) {
			best, bestLen := Suit(-1), 0
			for _, s := range []Suit{Spades, Hearts} {
				c := e.cheapestCall(s.Strain())
				need, floor := 5, 11
				if c.Level == 1 {
					floor = 6
					if !overcalled {
						// Over a takeout double the one-level bid is the only
						// way to show the major: four cards still bid it.
						need = 4
					}
				}
				if s == os || oppBid[s] || h.Len(s) < need || hl < floor || c.Level > 2 || !e.legal(p.seat, c) {
					continue
				}
				// The longest major, spades on a tie from five cards; with two
				// four-card majors, hearts first.
				if best < 0 || h.Len(s) > bestLen || (bestLen == 4 && h.Len(s) == 4) {
					best, bestLen = s, h.Len(s)
				}
			}
			if best >= 0 {
				c := e.cheapestCall(best.Strain())
				fr, en, floor := "changement de couleur au palier de 1, forcing", "one-level new suit, forcing", 6
				if c.Level == 2 {
					fr, en, floor = "changement de couleur 2 sur 1, forcing", "two-over-one new suit, forcing", 11
				}
				return c, m(floor, 40, fr, en).withLen(best, bestLen).asForcing()
			}
			if c, mn, ok := e.spoutnikDouble(p, os, oppBid, overcalled); ok {
				return c, mn
			}
		}
	}

	if oc.Strain <= SSpades {
		var rho Call
		if len(e.calls) > 0 {
			rho = e.calls[len(e.calls)-1].Call
		}
		doubled := rho.Kind == KindDouble
		if c, mn, ok := e.competitiveRencontre(p, Suit(oc.Strain), doubled, rho); ok {
			return c, mn
		}
	}

	if oc.Strain <= SSpades {
		os := Suit(oc.Strain)
		sup := h.Len(os)
		minSup := 3
		if !os.IsMajor() {
			minSup = 4
		}
		if sup >= minSup {
			hld := h.HLD(os)
			// From 13 HLD game values are known: conclude directly instead of
			// a limit raise that a minimum opener would pass.
			if hld >= 13 {
				if c, mn := e.conclude(p); c.Kind != KindPass {
					return c, mn
				}
			}
			// Law of total tricks (Jean-René Vernes): the number of tricks
			// available to both sides combined is approximately the number
			// of combined trumps, so a nine-card fit is worth competing to
			// the three level and a ten-card fit to the four level,
			// regardless of high-card points. It overrides the point-based
			// ceiling below whenever it would otherwise leave the auction
			// lower than the trump length alone already justifies -- a weak
			// hand must not be talked out of the level its fit affords.
			partner := e.ps[partnerOf(p.seat)]
			trumps := sup + partner.shownLens[os]
			lawLevel := 0
			switch {
			case trumps >= 10:
				lawLevel = 4
			case trumps >= 9:
				lawLevel = 3
			}
			cheapest := e.cheapestCall(os.Strain())
			raiseTo := func() Call {
				if lawLevel > cheapest.Level {
					return bid(lawLevel, os.Strain())
				}
				return cheapest
			}
			byLaw := lawLevel > cheapest.Level
			switch {
			case hld >= 11:
				c := raiseTo()
				if c.Level <= 4 {
					mn := m(11, 40, "soutien en compétition, 11HLD et plus", "competitive raise, 11+ HLD").withLen(os, sup)
					if byLaw {
						mn.fr, mn.en = "soutien loi des levées totales, fit 9 cartes et plus", "law of total tricks raise, nine-card fit or more"
					} else {
						mn = mn.asInvite()
					}
					return c, mn
				}
			case hld >= 6:
				c := raiseTo()
				if c.Level <= 4 {
					fr, en := "soutien simple en compétition", "competitive single raise"
					if byLaw {
						fr, en = "soutien loi des levées totales, fit 9 cartes et plus", "law of total tricks raise, nine-card fit or more"
					}
					return c, m(0, 40, fr, en).withLen(os, sup)
				}
			}
		}
	}
	// New suit at the one level stays forcing: show it cheaply, up the line.
	last, _, _ := e.lastBid()
	newSuitOK := func(s Suit) bool {
		if oc.Strain <= SSpades && s == Suit(oc.Strain) {
			return false
		}
		return !(last.IsBid() && last.Strain == s.Strain())
	}
	if hl >= 6 {
		for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
			if !newSuitOK(s) {
				continue
			}
			c := e.cheapestCall(s.Strain())
			if h.Len(s) >= 4 && c.Level == 1 {
				return c, m(6, 40, "changement de couleur au palier de 1, forcing", "one-level new suit, forcing").withLen(s, 4).asForcing()
			}
		}
	}
	// A new suit at the two level needs 11+ HL; among the candidates bid the
	// longest suit (ties to the higher-ranking one), not merely the first by
	// rank -- with 7 clubs and 5 diamonds, 2C is the natural call, not 2D.
	if hl >= 11 {
		best, bestLen := Clubs, 0
		for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
			if !newSuitOK(s) {
				continue
			}
			if h.Len(s) >= 5 && e.cheapestCall(s.Strain()).Level == 2 && h.Len(s) > bestLen {
				best, bestLen = s, h.Len(s)
			}
		}
		if bestLen > 0 {
			return e.cheapestCall(best.Strain()), m(11, 40, "changement de couleur 2 sur 1, forcing", "two-over-one new suit, forcing").withLen(best, 5).asForcing()
		}
	}
	if hl >= 8 && hl <= 10 && last.IsBid() && last.Strain <= SSpades && h.Stopper(Suit(last.Strain)) {
		c := e.cheapestCall(SNoTrump)
		if c.Level == 1 {
			return c, m(8, 10, "1SA, 8-10HL avec arrêt", "1NT, 8-10 HL with a stopper")
		}
	}
	if hl >= 11 {
		return e.conclude(p)
	}
	// This pass denies 11 HL and nothing less. While 1SA is still available at
	// the one level, an 8-10 hand with a stopper would have taken it, so the
	// pass reads as weak and keeps its 0-7 ceiling. Once the intervention has
	// pushed notrump to the two level that escape is gone: an 8-10 hand with
	// no four-card suit to show at the one level has no call at all and lands
	// here too, and capping it at 7 would announce a ceiling the system never
	// promised -- partner would stop short of the games the pair holds.
	maxShown := 7
	if e.cheapestCall(SNoTrump).Level > 1 {
		maxShown = 10
	}
	return passCall, m(0, maxShown, "", "")
}

// spoutnikDouble is the responder's negative double [RC-6]: over an opposing
// overcall it shows exactly four cards in the unbid major(s) -- the length a
// natural bid can no longer promise, since five cards name the suit. Opener
// answers it by naming the fit when he holds it (answerSpoutnik). It asks for
// six points over a one-level overcall, eight over a two-level one: partner
// may have to answer at the two level.
func (e *Engine) spoutnikDouble(p *playerState, os Suit, oppBid [4]bool, overcalled bool) (Call, meaning, bool) {
	h := p.hand
	last, _, _ := e.lastBid()
	if !overcalled || last.Level > 2 || !e.legal(p.seat, doubleCall) {
		return Call{}, meaning{}, false
	}
	floor := 6
	if last.Level >= 2 {
		floor = 8
	}
	if h.HL() < floor {
		return Call{}, meaning{}, false
	}
	var majors [4]bool
	var names, namesEN []string
	for _, s := range []Suit{Hearts, Spades} {
		if s == os || oppBid[s] || h.Len(s) != 4 {
			continue
		}
		majors[s] = true
		names = append(names, suitNameFR[s])
		namesEN = append(namesEN, suitNameEN[s])
	}
	if len(names) == 0 {
		return Call{}, meaning{}, false
	}
	fr := "contre Spoutnik, 4 cartes à " + names[0]
	en := "negative double, four " + namesEN[0]
	if len(names) == 2 {
		fr = "contre Spoutnik, les deux majeures quatrièmes"
		en = "negative double, both four-card majors"
	}
	mn := m(floor, 40, fr, en)
	mn.spoutnik = true
	mn.spoutnikSuits = majors
	for s := Clubs; s <= Spades; s++ {
		if majors[s] {
			mn = mn.withLen(s, 4)
		}
	}
	return doubleCall, mn, true
}

// answerSpoutnik answers partner's negative double. The double promised four
// cards in the major it named, so opener names the fit as soon as he holds
// four of them, by strength; without the fit he describes his own hand, and
// must not pass -- the double is a bid, not a penalty [RC-6].
func (e *Engine) answerSpoutnik(p *playerState, majors [4]bool) (Call, meaning) {
	h := p.hand
	var oppBid [4]bool
	for _, s := range e.opponentSuits(p) {
		oppBid[s] = true
	}
	best, bestLen := Suit(-1), 0
	for s := Clubs; s <= Spades; s++ {
		if majors[s] && h.Len(s) >= 4 && h.Len(s) > bestLen {
			best, bestLen = s, h.Len(s)
		}
	}
	if best >= 0 {
		hld := h.HLD(best)
		c := e.cheapestCall(best.Strain())
		fr, en := "réponse au Spoutnik, le fit majeur, minimum", "answer to the negative double, the major fit, minimum"
		lo, hi := 12, 16
		switch {
		case hld >= 19:
			c, lo, hi = bidSuit(4, best), 19, 40
			fr, en = "réponse au Spoutnik, le fit majeur, 19HLD et plus : la manche", "answer to the negative double, the major fit, 19+ HLD: game"
		case hld >= 17:
			c, lo, hi = bid(c.Level+1, best.Strain()), 17, 18
			fr, en = "réponse au Spoutnik, le fit majeur à saut, 17-18HLD", "answer to the negative double, jump in the major fit, 17-18 HLD"
		}
		if c.Level <= 4 && e.legal(p.seat, c) {
			mn := m(lo, hi, fr, en).withLen(best, bestLen)
			if hi <= 18 {
				mn = mn.asInvite()
			}
			return c, mn
		}
	}
	// No fit for the major: describe instead. Passing is not an option -- it
	// would turn the negative double into a penalty one.
	var opp Suit = Suit(-1)
	if last, _, ok := e.lastBid(); ok && last.Strain <= SSpades {
		opp = Suit(last.Strain)
	}
	if opp >= 0 && h.Stopper(opp) {
		c := e.cheapestCall(SNoTrump)
		hp := h.H()
		switch {
		case hp >= 18 && c.Level <= 3:
			return bid(3, SNoTrump), m(18, 40, "pas le fit majeur, arrêt adverse : 3SA", "no major fit, a stopper: 3NT")
		case c.Level <= 2 && e.legal(p.seat, c):
			return c, m(12, 17, "pas le fit majeur, arrêt dans la couleur adverse", "no major fit, a stopper in the opponents' suit")
		}
	}
	if e.openCall.Strain <= SSpades {
		os := Suit(e.openCall.Strain)
		if h.Len(os) >= 6 {
			c := e.cheapestCall(os.Strain())
			if c.Level <= 3 && e.legal(p.seat, c) {
				return c, m(12, 17, "pas le fit majeur, 6 cartes dans la couleur d'ouverture", "no major fit, six cards in the opened suit").withLen(os, 6)
			}
		}
	}
	for _, s := range []Suit{Clubs, Diamonds, Hearts, Spades} {
		if oppBid[s] || h.Len(s) < 4 || s == Suit(e.openCall.Strain) {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if c.Level <= 2 && e.legal(p.seat, c) {
			return c, m(12, 17, "pas le fit majeur, couleur quatrième", "no major fit, a four-card suit").withLen(s, 4)
		}
	}
	return e.conclude(p)
}

// respondRubensohl implements the responder's side of the Rubensohl
// convention over partner's 1SA opening after a natural suit intervention
// in v (docs/bidings.md, "LE RUBENSOHL", "Développements simplifiés"):
// double is positive (6H+, at least two cards in v, no singleton there), a
// suit above v at the two level is natural and weak, and everything from
// 2SA up is a transfer ("Texas") -- except the step that would transfer
// into v itself, which instead shows a singleton/void there and asks for
// the other major(s) ("Texas impossible").
func (e *Engine) respondRubensohl(p *playerState, v Suit) (Call, meaning) {
	h := p.hand
	hl := h.HL()

	// "Tendance Stayman": the double is for hands without a 5+-card suit of
	// their own to describe more precisely via the natural or Texas routes
	// below.
	if hl >= 6 && h.Len(v) >= 2 && h.sortedLens()[0] < 5 && e.legal(p.seat, doubleCall) {
		mn := m(6, 40, "Contre Rubensohl, positif, tendance Stayman, sans singleton à "+suitNameFR[v],
			"Rubensohl double, positive, Stayman tendency, no singleton in "+suitNameEN[v])
		mn.rubensohlDouble = true
		return doubleCall, mn
	}

	// Natural, weak two-level suit above the intervention.
	if hl <= 7 {
		for _, s := range []Suit{Diamonds, Hearts, Spades} {
			if s.Strain() <= v.Strain() || h.Len(s) < 5 {
				continue
			}
			c := bid(2, s.Strain())
			if e.legal(p.seat, c) {
				return c, m(0, 7, "naturel, plutôt faible, non forcing", "natural, rather weak, non forcing").withLen(s, 5)
			}
		}
	}

	// From 2SA up: Texas ladder (Trèfle, Carreau, Cœur, Pique), except the
	// step landing on v, which becomes the "Texas impossible" ask.
	if hl >= 8 {
		ladder := []Suit{Clubs, Diamonds, Hearts, Spades}
		for i, s := range ladder {
			c := bid(2, SNoTrump)
			if i > 0 {
				c = bid(3, ladder[i-1].Strain())
			}
			if s == v {
				if h.Len(v) <= 1 && e.legal(p.seat, c) {
					mn := m(8, 40, "Texas impossible, chicane/singleton à "+suitNameFR[v]+", demande les majeures",
						"impossible Texas, singleton/void in "+suitNameEN[v]+", asks for the majors").asForcing()
					mn.rubensohlAsk = true
					return c, mn
				}
				continue
			}
			if h.Len(s) >= 5 && e.legal(p.seat, c) {
				mn := m(0, 40, "Texas Rubensohl pour "+suitNameFR[s], "Rubensohl transfer to "+suitNameEN[s]).withLen(s, 5).asForcing()
				mn.hasTexas, mn.texas = true, s
				if s.IsMajor() {
					completion := bid(c.Level, s.Strain())
					if !completion.higherThan(c) {
						completion = bid(c.Level+1, s.Strain())
					}
					p.planned = func() (Call, meaning) { return e.afterTransfer(p, s, 15, completion) }
				} else {
					p.planned = func() (Call, meaning) { return e.afterRubensohlMinorTexas(p, s) }
				}
				return c, mn
			}
		}
	}

	// Game values with no clear route above: bid 3NT directly with a stopper
	// in v, or with a 4-3-3-3 shape and four cards in a minor (docs/bidings.md,
	// remark from A. Lévy: "quoi d'autre ?"). Otherwise, ask for a stopper
	// with 3S rather than guess (SEF 2018 p.29, "3♠ = demande d'arrêt pour
	// 3SA, quelle que soit l'intervention").
	if hl >= 10 {
		l := h.sortedLens()
		flat433 := l[0] == 4 && l[1] == 3 && l[2] == 3 && l[3] == 3 && !h.Longest().IsMajor()
		if h.Stopper(v) || flat433 {
			c := bid(3, SNoTrump)
			if e.legal(p.seat, c) {
				return c, m(10, 40, "3SA naturel", "natural 3NT")
			}
		}
		c := bidSuit(3, Spades)
		if e.legal(p.seat, c) {
			mn := m(10, 40, "3P, demande d'arrêt pour 3SA quelle que soit l'intervention", "3S, asks for a stopper for 3NT regardless of the intervention suit").asForcing()
			mn.rubensohlStopperAsk = true
			return c, mn
		}
	}
	return passCall, m(0, 9, "jeu trop faible pour agir", "too weak to act")
}

// afterRubensohlMinorTexas continues a Rubensohl transfer to a minor. Unlike
// a major transfer at the two level, the opener's 3m completion is mandatory
// (there is no room for a super-accept jump), so it promises nothing extra:
// a limited hand signs off there, and only with the combined count in the
// game zone does the generic engine pick between 3NT, the minor game or a
// slam try.
func (e *Engine) afterRubensohlMinorTexas(p *playerState, s Suit) (Call, meaning) {
	partner := e.ps[partnerOf(p.seat)]
	if p.hand.HLD(s)+partner.shownMin >= 25 {
		if c, mn := e.conclude(p); c.Kind != KindPass {
			return c, mn
		}
	}
	return passCall, m(-1, -1, "arrêt à trois dans la mineure, jeu limité", "sign-off at three of the minor, limited hand")
}

// rubensohlDoubleAnswer answers responder's positive Rubensohl double: name
// the cheaper 4+-card major that isn't the intervention suit; without one,
// pass either to convert the double to penalty (stopper there and a
// maximum) or simply because there is nothing better to do.
func (e *Engine) rubensohlDoubleAnswer(p *playerState) (Call, meaning) {
	h := p.hand
	var v Suit
	if suits := e.opponentSuits(p); len(suits) > 0 {
		v = suits[0]
	}
	for _, s := range []Suit{Hearts, Spades} {
		if s == v || h.Len(s) < 4 {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "répond sa majeure au Contre Rubensohl", "answers the Rubensohl double with a major").withLen(s, 4)
		}
	}
	if h.Stopper(v) && h.H() >= 17 {
		return passCall, m(-1, -1, "conversion du Contre en punitif, tenue et maximum", "converts the double to penalty, stopper and maximum")
	}
	return passCall, m(-1, -1, "pas de majeure quatrième", "no four-card major")
}

// rubensohlAskAnswer answers responder's "impossible Texas" (singleton or
// void in the intervention suit v, asking for the other major(s)): show a
// 4-card major that isn't v, else 3NT with a stopper in v, else cue-bid v
// to deny both.
func (e *Engine) rubensohlAskAnswer(p *playerState) (Call, meaning) {
	h := p.hand
	var v Suit
	if suits := e.opponentSuits(p); len(suits) > 0 {
		v = suits[0]
	}
	for _, s := range []Suit{Hearts, Spades} {
		if s == v || h.Len(s) < 4 {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "4 cartes à "+suitNameFR[s], "four "+suitNameEN[s]).withLen(s, 4)
		}
	}
	if h.Stopper(v) {
		c := bid(3, SNoTrump)
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "pas de majeure quatrième, arrêt à "+suitNameFR[v], "no four-card major, stopper in "+suitNameEN[v])
		}
	}
	c := e.cheapestCall(v.Strain())
	if e.legal(p.seat, c) {
		return c, m(-1, -1, "pas de majeure quatrième, pas d'arrêt à "+suitNameFR[v], "no four-card major, no stopper in "+suitNameEN[v])
	}
	return passCall, noInfo()
}

// rubensohlStopperAskAnswer answers responder's 3S (SEF 2018 p.29, "demande
// d'arrêt pour 3SA quelle que soit l'intervention"): bid 3NT with a stopper
// in the intervention suit v, else cue-bid v to deny it.
func (e *Engine) rubensohlStopperAskAnswer(p *playerState) (Call, meaning) {
	h := p.hand
	var v Suit
	if suits := e.opponentSuits(p); len(suits) > 0 {
		v = suits[0]
	}
	if h.Stopper(v) {
		c := bid(3, SNoTrump)
		if e.legal(p.seat, c) {
			return c, m(-1, -1, "arrêt à "+suitNameFR[v], "stopper in "+suitNameEN[v])
		}
	}
	c := e.cheapestCall(v.Strain())
	if e.legal(p.seat, c) {
		return c, m(-1, -1, "pas d'arrêt à "+suitNameFR[v], "no stopper in "+suitNameEN[v])
	}
	return passCall, noInfo()
}

// ---------- opener rebid ----------

// lightOpenerRebid is the second half of [O-7b]: having opened below the
// threshold, the hand has already spent its whole story on the opening and
// never bids again of its own accord. Partner is a passed hand, so there is
// no game behind his answer and nothing to compete for; the second bid is
// exactly where a light opening turns a plus score into a minus one, and
// where the ordinary rebids would promise the 12-14 the hand does not have.
//
// It speaks only when partner's call forces it -- the Drury ask [RM-2b], the
// rencontre [RM-2] -- and then says the least it can: the cheapest repeat of
// the opened suit, with the 10-11 it holds. That is the "non" the Drury
// question was asking for.
func (e *Engine) lightOpenerRebid(p *playerState) (Call, meaning, bool) {
	resp := e.ps[partnerOf(p.seat)]
	rm := resp.lastM
	forced := rm != nil && (rm.forcing || rm.relay)
	if !forced {
		return passCall, m(-1, 11, "ouverture légère de troisième ou quatrième : elle a déjà tout dit, elle passe", "light third- or fourth-seat opening: it has said everything, it passes"), true
	}
	oc := e.openCall
	if !oc.IsBid() || oc.Strain > SSpades {
		return Call{}, meaning{}, false
	}
	last, _, _ := e.lastBid()
	rep := bid(max(oc.Level, last.Level), oc.Strain)
	if !rep.higherThan(last) {
		rep = bid(last.Level+1, oc.Strain)
	}
	if rep.Level > 4 || !e.legal(p.seat, rep) {
		return Call{}, meaning{}, false
	}
	return rep, m(-1, 11, "ouverture légère : non, les points n'y sont pas, on s'arrête là", "light opening: no, the points are not there, this is the ceiling"), true
}

func (e *Engine) openerRebid(p *playerState) (Call, meaning) {
	if p.lightOpen {
		if c, mn, ok := e.lightOpenerRebid(p); ok {
			return e.untraced(c, mn)
		}
	}
	resp := e.ps[partnerOf(p.seat)]
	rm := resp.lastM
	rsc, hasResp := e.lastCallBy(resp.seat)
	oc := e.openCall

	// The negative double is a bid, not a penalty: it must be answered
	// before any of the natural rebid machinery [RC-6].
	if rm != nil && rm.spoutnik && hasResp && rsc.Call.Kind == KindDouble {
		return e.untraced(e.answerSpoutnik(p, rm.spoutnikSuits))
	}

	switch {
	case oc == bid(1, SNoTrump) || oc == bid(2, SNoTrump):
		if rm != nil && rm.relay && rsc.Call.IsBid() {
			e.tr.note("le partenaire a demandé Stayman : montrer ses majeures quatrièmes",
				"partner asked Stayman: show the four-card majors")
			return e.staymanAnswer(p, oc.Level)
		}
		if rm != nil && rm.hasTexas {
			if rm.texas.IsMajor() {
				e.tr.note("le partenaire a fait un Texas : rectifier dans sa majeure",
					"partner transferred: complete into his major")
				return e.transferAnswer(p, rm.texas)
			}
			return e.untraced(e.minorTransferAnswer(p, rm.texas))
		}
		if rm != nil && rm.rubensohlDouble {
			return e.untraced(e.rubensohlDoubleAnswer(p))
		}
		if rm != nil && rm.rubensohlAsk {
			return e.untraced(e.rubensohlAskAnswer(p))
		}
		if rm != nil && rm.rubensohlStopperAsk {
			return e.untraced(e.rubensohlStopperAskAnswer(p))
		}
		return e.untraced(e.conclude(p))
	case oc == bid(2, SClubs):
		if hasResp && rsc.Call == bid(2, SDiamonds) {
			e.tr.note("2♣ fort, relais 2♦ du partenaire : décrire la main",
				"strong 2♣, partner's 2♦ relay: describe the hand")
			return e.strongRebid(p)
		}
		return e.untraced(e.conclude(p))
	case oc == bid(2, SDiamonds):
		return e.untraced(e.gfRebid(p))
	case oc == bid(3, SNoTrump):
		if hasResp && rsc.Call.IsBid() && rsc.Call.Strain <= SDiamonds {
			return e.untraced(e.affranchieRebid(p))
		}
		return e.untraced(e.conclude(p))
	case oc.Level == 2 && (oc.Strain == SHearts || oc.Strain == SSpades):
		if hasResp && rsc.Call == bid(2, SNoTrump) {
			return e.untraced(e.weak2Feature(p, Suit(oc.Strain)))
		}
		return e.untraced(e.conclude(p))
	case oc.Level == 1 && oc.Strain <= SSpades:
		return e.naturalRebid(p, rsc, hasResp)
	}
	return e.untraced(e.conclude(p))
}

func (e *Engine) staymanAnswer(p *playerState, base int) (Call, meaning) {
	h := p.hand
	h4, s4 := h.Len(Hearts) >= 4, h.Len(Spades) >= 4
	// Standard scheme (docs/bidings.md, "LE STAYMAN"): the suit one step above
	// the ask denies both majors, then 2H/2S (3H/3S over 2SA) show one major
	// without the other, and 2SA/3SA show both.
	tr := e.tr
	majors := cards(h, Hearts) + ", " + cards(h, Spades)
	if base == 1 {
		switch {
		case tr.check(h4 && s4, "4 ♥ et 4 ♠ → 2SA", "4 ♥ and 4 ♠ → 2NT", majors):
			return bid(2, SNoTrump), m(-1, -1, "4 cartes à Cœur et 4 cartes à Pique", "four hearts and four spades").withLen(Hearts, 4).withLen(Spades, 4)
		case tr.check(h4, "4 ♥ sans 4 ♠ → 2♥", "4 ♥ without 4 ♠ → 2♥", majors):
			return bid(2, SHearts), m(-1, -1, "4 cartes à Cœur sans 4 cartes à Pique", "four hearts, not four spades").withLen(Hearts, 4)
		case tr.check(s4, "4 ♠ sans 4 ♥ → 2♠", "4 ♠ without 4 ♥ → 2♠", majors):
			return bid(2, SSpades), m(-1, -1, "4 cartes à Pique sans 4 cartes à Cœur", "four spades, not four hearts").withLen(Spades, 4)
		default:
			tr.check(true, "pas de majeure quatrième → 2♦", "no four-card major → 2♦", majors)
			return bid(2, SDiamonds), m(-1, -1, "pas de majeure quatrième", "no four-card major")
		}
	}
	switch {
	case tr.check(h4 && s4, "4 ♥ et 4 ♠ → 3SA", "4 ♥ and 4 ♠ → 3NT", majors):
		return bid(3, SNoTrump), m(-1, -1, "4 cartes à Cœur et 4 cartes à Pique", "four hearts and four spades").withLen(Hearts, 4).withLen(Spades, 4)
	case tr.check(h4, "4 ♥ sans 4 ♠ → 3♥", "4 ♥ without 4 ♠ → 3♥", majors):
		return bid(3, SHearts), m(-1, -1, "4 cartes à Cœur sans 4 cartes à Pique", "four hearts, not four spades").withLen(Hearts, 4)
	case tr.check(s4, "4 ♠ sans 4 ♥ → 3♠", "4 ♠ without 4 ♥ → 3♠", majors):
		return bid(3, SSpades), m(-1, -1, "4 cartes à Pique sans 4 cartes à Cœur", "four spades, not four hearts").withLen(Spades, 4)
	default:
		tr.check(true, "pas de majeure quatrième → 3♦", "no four-card major → 3♦", majors)
		return bid(3, SDiamonds), m(-1, -1, "pas de majeure quatrième", "no four-card major")
	}
}

func (e *Engine) transferAnswer(p *playerState, t Suit) (Call, meaning) {
	h := p.hand
	c := e.cheapestCall(t.Strain())
	if e.tr.check(t.IsMajor() && h.Len(t) >= 4 && p.shownMin+2 <= p.hand.H() && c.Level <= 2,
		"4 atouts et un maximum → rectification à saut",
		"four trumps and a maximum → super-accept", cards(h, t)+", "+pts(h.H(), "H")) {
		// super-accept with four trumps and a maximum
		return bid(c.Level+1, t.Strain()), m(p.shownMin+2, p.shownMax, "rectification à saut : 4 atouts, maximum", "super-accept: four trumps, maximum").withLen(t, 4)
	}
	e.tr.note("sinon → rectification simple", "otherwise → plain completion")
	return c, m(-1, -1, "rectification du Texas", "completing the transfer").withLen(t, 0)
}

func (e *Engine) strongRebid(p *playerState) (Call, meaning) {
	h := p.hand
	hp := h.H()
	tr := e.tr
	for _, s := range []Suit{Spades, Hearts} {
		if tr.check(h.Len(s) >= 6 || (h.Len(s) >= 5 && hp >= 21),
			"6 "+suitSymbol[s]+", ou 5 avec 21 H → 2"+suitSymbol[s],
			"six "+suitSymbol[s]+", or five with 21 H → 2"+suitSymbol[s], cards(h, s)+", "+pts(hp, "H")) {
			return bidSuit(2, s), m(18, 23, "6 cartes et 18-21H ou 5 cartes et 21-22H, forcing", "six cards 18-21 or five cards 21-22, forcing").withLen(s, 5).asForcing()
		}
	}
	// The 2NT rebid genuinely shows 22-23: an 18-21 hand that opened 2C on
	// its six-card suit (a semi-regular 6-3-2-2 included) must name the suit,
	// not inflate its strength behind a "balanced" 2NT.
	if tr.check((h.IsRegular() || h.IsSemiRegular()) && hp >= 22,
		"22-23 H, régulière ou semi-régulière → 2SA", "22-23 H, balanced → 2NT", pts(hp, "H")+", "+shape(h)) {
		return bid(2, SNoTrump), m(22, 23, "22-23H, jeu (semi-)régulier", "22-23 balanced")
	}
	long := h.Longest()
	if tr.check(h.Len(long) >= 6, "couleur sixième → 3 à la couleur", "six-card suit → three of the suit", cards(h, long)) {
		return bidSuit(3, long), m(18, 23, "belle couleur longue, forcing", "long strong suit, forcing").withLen(long, 6).asForcing()
	}
	if tr.check(h.Len(long) >= 5, "couleur cinquième, main irrégulière → 3 à la couleur",
		"five-card suit, unbalanced → three of the suit", cards(h, long)) {
		// Unbalanced with a five-card minor (a 5-5 two-suiter or 5-4-4-0):
		// name it rather than hide a possible void behind a "balanced" 2NT
		// that the transfer machinery would then trust for a doubleton fit.
		return bidSuit(3, long), m(18, 23, "couleur cinquième, jeu irrégulier, forcing", "five-card suit, unbalanced, forcing").withLen(long, 5).asForcing()
	}
	// No five-card suit at all: a 4-4-4-1 three-suiter has no long suit to
	// name, so a 3-level suit rebid would lie about the holding. Rebid 2NT on
	// strength instead.
	tr.note("sans couleur longue → 2SA", "no long suit → 2NT")
	return bid(2, SNoTrump), m(22, 23, "22-23H, sans couleur longue", "22-23, no long suit")
}

func (e *Engine) gfRebid(p *playerState) (Call, meaning) {
	h := p.hand
	long := h.Longest()
	if h.Len(long) >= 5 {
		c := e.cheapestCall(long.Strain())
		return c, m(24, 40, "couleur cinquième et plus, forcing de manche", "five-card or longer suit, game forcing").withLen(long, 5).asForcing()
	}
	c := e.cheapestCall(SNoTrump)
	// Over a three-level ace response the "default" notrump rebid is already
	// 3SA -- a game partner will pass, which buries every hand above the bare
	// 24HL the opening promised. With all four aces located the opening's own
	// tool applies instead (docs/bidings.md, "Le 4SA de l'ouvreur est un appel
	// aux Rois").
	if isGame(c) {
		if kc, kmn, ok := e.strongTwoDKingAsk(p); ok {
			return kc, kmn
		}
	}
	// Both hands have a floor -- the opening's own and the one the ace step
	// promised -- and once the two add up to the slam zone the default 3SA
	// buries the deal: partner passes a game the pair was always beyond. Name
	// the slam the count promises rather than the game (33 H for six, 37 for
	// seven); with no fit found, notrump is where it plays.
	partner := e.ps[partnerOf(p.seat)]
	if cMin := h.H() + partner.shownMin; cMin >= 33 {
		lvl := 6
		if cMin >= 37 {
			lvl = 7
		}
		sc := bid(lvl, SNoTrump)
		if sc.higherThan(c) && e.legal(p.seat, sc) {
			fr, en := "petit chelem à Sans-Atout : 33 points réunis", "small slam in notrump: 33 points between the hands"
			if lvl == 7 {
				fr, en = "grand chelem à Sans-Atout : 37 points réunis", "grand slam in notrump: 37 points between the hands"
			}
			return sc, m(cMin-partner.shownMin, 40, fr, en)
		}
	}
	mn := m(24, 40, "redemande par défaut, jeu régulier", "default rebid, balanced hand")
	// Below game the rebid stays forcing (the 2D opening is game-forcing);
	// at 3SA the game is reached and partner may pass.
	if !isGame(c) {
		mn = mn.asForcing()
	}
	return c, mn
}

// strongTwoDKingAsk replaces opener's default notrump rebid by the king ask
// when that default would already be a game partner can pass. The ace-step
// response pins partner's aces but leaves his range wide open, so signing off
// caps a hand the auction never got to describe. Asking is only worth the
// five level it costs when a single king from partner would carry the pair to
// the 33-honour small slam zone; the ask itself can still stop in 5SA when the
// answer disappoints (afterKingsNT).
func (e *Engine) strongTwoDKingAsk(p *playerState) (Call, meaning, bool) {
	side := sideOf(p.seat)
	if e.bw[side].asked || !e.isKingAsk(p) {
		return Call{}, meaning{}, false
	}
	partner := e.ps[partnerOf(p.seat)]
	if p.hand.H()+partner.shownMin+3 < 33 {
		return Call{}, meaning{}, false
	}
	c := bid(4, SNoTrump)
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	mn := m(p.hand.H()-1, -1, "Blackwood 4SA, appel aux Rois (As déjà localisés)", "4NT king Blackwood (aces already located)")
	mn.blackwood = true
	mn.forcing = true
	e.bw[side] = bwState{asked: true, asker: p.seat, noTrump: true, kingAsk: true}
	return c, mn, true
}

// weak2Feature answers the 2NT relay-and-fit-ask over a weak 2H/2S opening
// (docs/addon_7.md). A minimum (9-11 HLD) simply repeats the suit at the
// 3-level. Otherwise (12-14 HLD, "strong" for the weak-two range), priority
// goes to mentioning an outside ace or king in a 3+ card suit at the
// 3-level; failing that, a maximum with a singleton announces it directly
// at the 4-level (after a 2H opening specifically, a spade singleton is
// shown by repeating 4H, since 3S is already taken by the outside-force
// meaning); with neither feature, 3NT concludes.
func (e *Engine) weak2Feature(p *playerState, M Suit) (Call, meaning) {
	h := p.hand
	hld := h.HLD(M)
	if hld <= 11 {
		return bidSuit(3, M), m(9, 11, "jeu minimal, 9-11HLD", "minimum hand, 9-11 HLD").withLen(M, 6)
	}
	for s := Spades; s >= Clubs; s-- {
		if s != M && h.Len(s) >= 3 && (h.HasCard(s, 'A') || h.HasCard(s, 'K')) {
			c := e.cheapestCall(s.Strain())
			if c.Level <= 3 {
				return c, m(12, 14, "12-14HLD, force extérieure (As ou Roi)", "12-14 HLD, an outside ace or king").withLen(s, 3).asForcing()
			}
		}
	}
	for s := Clubs; s <= Spades; s++ {
		if s != M && h.Len(s) == 1 {
			if M == Hearts && s == Spades {
				return bid(4, SHearts), m(12, 14, "maximum, singleton Pique", "maximum, singleton spade").withLen(M, 6)
			}
			return bidSuit(4, s), m(12, 14, "maximum, singleton", "maximum, singleton").withLen(s, 1)
		}
	}
	return bid(3, SNoTrump), m(12, 14, "maximum, pas de force ni singleton", "maximum, no outside force or singleton")
}

// naturalRebid covers opener's second call after a one-level suit opening.
func (e *Engine) naturalRebid(p *playerState, rsc SeatCall, hasResp bool) (Call, meaning) {
	h := p.hand
	os := Suit(e.openCall.Strain)
	resp := e.ps[partnerOf(p.seat)]
	rm := resp.lastM

	if !hasResp || rsc.Call.Kind == KindPass {
		return e.untraced(e.openerReopen(p, os))
	}
	rc := rsc.Call
	if rc.Kind != KindBid {
		return e.untraced(e.conclude(p))
	}
	tr := e.tr

	// Drury [RM-2b]: the ask is artificial, so nothing below may
	// read 2T as a club suit or 2SA as a natural notrump raise. Interference
	// after the ask cancels the scheme -- the convention lives "dans le
	// silence adverse" -- and the generic machinery takes over.
	if rm != nil && (rm.drury || rm.druryShort) {
		if !e.uncontested(p.seat) {
			// Interference cancelled the scheme; whatever happens next, 2T is
			// not a club suit to raise, so the generic machinery -- which
			// reads the fit from the promised lengths -- takes over.
			return e.untraced(e.conclude(p))
		}
		if rm.drury {
			return e.untraced(e.druryRebid(p, os))
		}
		return e.untraced(e.druryShortRebid(p, os))
	}

	// Splinter and control-bid responses: the fit (and short suit) are
	// already known from their own meaning (docs/addon_5.md, addon_4.md); the
	// generic slam engine takes over rather than rebidOverNewSuit, which
	// would misread the named suit as a natural one to support.
	if rm != nil && (rm.splinter || rm.controlBid) {
		return e.untraced(e.conclude(p))
	}

	// Conventional 2NT raise over a major (3 trumps, 11-12 HLD).
	if os.IsMajor() && rc == bid(2, SNoTrump) && rm != nil && tr.check(rm.lens[os] >= 3,
		"2SA fitté du partenaire (3 atouts, 11-12 HLD) → accepter ou refuser la manche",
		"partner's conventional 2NT (three trumps, 11-12 HLD) → accept or decline game", "") {
		// Value the hand for acceptance by HL plus genuine shortness only. A
		// doubleton in an otherwise balanced 5-3-3-2 adds a distributional
		// point that does not become a playing trick opposite a limited
		// three-card raise, so it must not lift a bare minimum opening into a
		// failing game (same rule as the generic invitation, engine.go). Only
		// a singleton or void counts toward straining to the manche.
		val := h.HL()
		for s := Clubs; s <= Spades; s++ {
			if s == os {
				continue
			}
			switch h.Len(s) {
			case 0:
				val += 3
			case 1:
				val += 2
			}
		}
		tr.in()
		defer tr.out()
		switch {
		case tr.check(val <= 13, "13 points au plus (HL, singleton +2, chicane +3) → refus : 3 de la majeure",
			"13 points at most (HL, singleton +2, void +3) → decline: three of the major", fmt.Sprintf("%d", val)):
			return bidSuit(3, os), m(12, 13, "jeu minimal, refus de la proposition", "minimum, declining the game try").withLen(os, 5)
		default:
			tr.check(true, "14 points et plus → manche", "14 points or more → game", fmt.Sprintf("%d", val))
			return bidSuit(4, os), m(14, 23, "accepte la proposition de manche", "accepting the game try").withLen(os, 5)
		}
	}
	// Invitations and conventional strong responses are settled generically.
	if rm != nil && (rm.invite || rm.slamInvite) {
		return e.untraced(e.conclude(p))
	}
	// Raise of the opened suit.
	if tr.check(rc.Strain == os.Strain(),
		"le partenaire soutient la couleur d'ouverture", "partner raises the opened suit", "") {
		hld := e.hldAgainstTheirBidding(p, os)
		tr.in()
		defer tr.out()
		switch rc.Level {
		case 2:
			switch {
			// Direct game only when it stands even opposite the floor of the
			// 6-10 raise (22 + 6 = 28 ≥ 27): with less, partner may hold a
			// bare 6 and the hand must go through a game try instead.
			case os.IsMajor() && tr.check(hld >= 22, "22 HLD et plus → manche directe",
				"22+ HLD → game directly", pts(hld, "HLD")):
				return bidSuit(4, os), m(22, 23, "conclusion à la manche", "bidding game")
			case os.IsMajor() && tr.check(hld >= 17, "17-21 HLD → essai de manche",
				"17-21 HLD → game try", pts(hld, "HLD")):
				// Facing a simple raise, the bare rebid of the trump suit is a
				// barrage (docs/bidings.md, "EN FACE D'UN SOUTIEN MAJEUR
				// SIMPLE"): a real game try must go through a new suit needing
				// help, or failing that a generalized 2NT try.
				tr.in()
				if c, mn, ok := e.helpSuitGameTry(p, os); tr.check(ok,
					"une couleur annexe qui a besoin d'aide → essai dans cette couleur",
					"a side suit needing help → help-suit game try", shape(h)) {
					p.invited = true
					return c, mn
				}
				if c := bid(2, SNoTrump); tr.check(e.legal(p.seat, c), "sinon → essai généralisé 2SA",
					"otherwise → generalized 2NT try", "") {
					p.invited = true
					return c, m(17, 21, "essai de manche généralisé", "generalized game try").asInvite()
				}
				if c := bidSuit(3, os); e.legal(p.seat, c) {
					return c, m(17, 21, "enchère d'essai pour la manche", "game try").asInvite()
				}
				// The opponents' bidding has taken every spot below game: a
				// bid of the trump suit here can no longer be a try (it lands
				// at the game level itself), so call it what it is.
				if c := e.cheapestCall(os.Strain()); e.legal(p.seat, c) {
					return c, m(17, 21, "conclusion à la manche, plus de place pour un essai", "bidding game outright, no room left to try")
				}
				return e.untraced(e.conclude(p))
			case tr.check(hld >= 17, "17 HLD et plus → essai à 3", "17+ HLD → three-level try", pts(hld, "HLD")):
				return bidSuit(3, os), m(17, 21, "enchère d'essai pour la manche", "game try").asInvite()
			case os.IsMajor() && tr.check(h.Len(os) >= 6, "6 cartes dans la majeure → barrage à 3",
				"six cards in the major → preemptive three", cards(h, os)):
				return bidSuit(3, os), m(12, 16, "barrage, prolongement du soutien", "preemptive raise, extending the fit").withLen(os, 6)
			default:
				tr.note("12-16 HLD : la manche est hors d'atteinte → Passe", "12-16 HLD: game is out of reach → Pass")
				return passCall, m(12, 16, "", "")
			}
		case 3:
			// "La décision d'explorer le chelem est uniquement le fait du
			// joueur qui vient de recevoir l'information de ce soutien
			// limite" : the jump raise is a narrow, trustworthy zone, so once
			// the combined count reaches the exploration window the slam
			// machinery gets the call -- blasting the game here would throw
			// away the one moment the decision can be taken.
			if hld+resp.shownMin >= 29 {
				return e.untraced(e.conclude(p))
			}
			// The jump raise's own floor, not a fixed 11: the same shape --
			// three of the opened suit -- is also what the law of total
			// tricks bids in a partscore battle [L-1b], on trump length
			// alone and promising nothing. Reading that one as an invitation
			// turns a fit shown to protect a partscore into a game.
			if os.IsMajor() && tr.check(hld+resp.shownMin >= 27,
				"soutien à saut : 27 points ensemble avec le minimum du partenaire → manche",
				"jump raise: 27 points together with partner's minimum → game",
				fmt.Sprintf("%d + %d", hld, resp.shownMin)) {
				return bidSuit(4, os), m(15, 23, "accepte l'invitation", "accepting the invitation")
			}
			return e.untraced(e.conclude(p))
		default:
			return e.untraced(e.conclude(p))
		}
	}
	// Notrump responses.
	if tr.check(rc.Strain == SNoTrump && rc.Level == 1, "le partenaire répond 1SA (6-10 HL, sans fit)",
		"partner answers 1NT (6-10 HL, no fit)", "") {
		tr.in()
		defer tr.out()
		return e.rebidOverOneNT(p, os)
	}
	if rc.Strain == SNoTrump {
		return e.untraced(e.conclude(p))
	}
	// New suit by responder (forcing): describe the hand.
	tr.note("le partenaire nomme une nouvelle couleur (forcing) : décrire la main",
		"partner names a new suit (forcing): describe the hand")
	tr.in()
	defer tr.out()
	return e.rebidOverNewSuit(p, os, Suit(rc.Strain), rc.Level)
}

func (e *Engine) rebidOverOneNT(p *playerState, os Suit) (Call, meaning) {
	h := p.hand
	hp := h.H()
	// The 2SA/3SA rebids over the 1SA response promise a genuinely balanced
	// hand (docs/bidings.md, "la redemande à SA" : 17-18H régulier for 2SA,
	// 18-19H régulier for 3SA). An unbalanced hand — e.g. a 6-card major with
	// a singleton — must instead show its shape naturally, so gate the notrump
	// rebids on the same balanced test used for the new-suit rebid below.
	balanced := (h.IsRegular() || h.IsSemiRegular()) && h.Len(h.Longest()) <= 5
	tr := e.tr
	switch {
	case os.IsMajor() && tr.check(h.Len(os) >= 7 && h.HLD(os)+e.ps[partnerOf(p.seat)].shownMin >= gameThreshold(os, true),
		"majeure septième qui vaut la manche avec le minimum du partenaire → 4 de la majeure",
		"seven-card major worth game with partner's minimum → four of the major", cards(h, os)):
		// A self-sufficient seven-card major is worth game on its own playing
		// strength even opposite the 6-10 notrump response: counted as trump,
		// its length and side shortness (e.g. a void) add distribution the bare
		// HL total misses. When HLD plus partner's floor reaches game, conclude
		// in four of the major rather than making the merely invitational,
		// non-forcing jump rebid below — which partner may pass with a maximum,
		// the fit in the long suit being unknown to him.
		return bidSuit(4, os), m(18, 21, "conclusion à la manche, bel unicolore auto-suffisant", "bidding game, self-sufficient one-suiter").withLen(os, 7)
	case tr.check(balanced && hp >= 18, "régulière, 18-19 H → 3SA", "balanced, 18-19 H → 3NT", pts(hp, "H")+", "+shape(h)):
		return bid(3, SNoTrump), m(18, 19, "18-19H, conclusion", "18-19, bidding game")
	case tr.check(balanced && hp >= 17, "régulière, 17 H → 2SA, proposition", "balanced, 17 H → 2NT, invitation", pts(hp, "H")+", "+shape(h)):
		return bid(2, SNoTrump), m(17, 18, "17-18H régulier, proposition", "17-18 balanced, invitation").asInvite()
	case tr.check(h.Len(os) >= 6 && h.HL() >= 17, "6 cartes et 17 HL → répétition à saut",
		"six cards and 17 HL → jump rebid", cards(h, os)+", "+pts(h.HL(), "HL")):
		// Strong irregular one-suiter: too good for the 13-16 simple rebid, and
		// unable to bid 2SA (not balanced). The jump repetition shows 17-19HL
		// and a good six-card suit, non-forcing (docs/bidings.md, "la
		// répétition à saut de la couleur d'ouverture").
		mn := m(17, 19, "répétition à saut, bel unicolore", "jump rebid, good six-card suit").withLen(os, 6).asInvite()
		mn.openerMinorRebid = !os.IsMajor()
		return bidSuit(3, os), mn
	case tr.check(h.Len(os) >= 6, "6 cartes → répétition au palier de 2", "six cards → two-level rebid", cards(h, os)):
		mn := m(13, 16, "répétition, 6 cartes", "rebid, six-card suit").withLen(os, 6)
		mn.openerMinorRebid = !os.IsMajor()
		return bidSuit(2, os), mn
	default:
		// The cheap second suit "promet une main irrégulière" (docs/bidings.md,
		// "LE BICOLORE ÉCONOMIQUE"; its 15-17H exception, the "faux bicolore
		// économique", names a *three*-card minor and never reaches here). A
		// regular hand has nothing to gain by showing a 4-4 second suit over
		// the 6-10 notrump response: with 17H and up it has already rebid
		// notrump above, and below that game is out of reach and 1SA is the
		// better contract, so it passes.
		if !h.IsRegular() {
			for s := Spades; s >= Clubs; s-- {
				if s != os && h.Len(s) >= 4 && s.Strain() < os.Strain() {
					tr.check(true, "main irrégulière, 4 cartes dans une couleur moins chère → bicolore économique",
						"unbalanced, four cards in a cheaper suit → economical two-suiter", cards(h, s))
					return e.cheapSecondSuit(p, s, bidSuit(2, s), h.HL())
				}
			}
		}
		if tr.check(h.Len(os) >= 5 && !h.IsRegular(), "main irrégulière, 5 cartes → répétition par défaut",
			"unbalanced, five cards → default rebid", cards(h, os)+", "+shape(h)) {
			return bidSuit(2, os), m(12, 14, "répétition par défaut", "default rebid").withLen(os, 5)
		}
		tr.note("main régulière minimale : 1SA est le bon contrat → Passe", "minimum balanced hand: 1NT is the right contract → Pass")
		return passCall, m(12, 14, "jeu régulier minimal", "minimum balanced hand")
	}
}

// cheapSecondSuit builds opener's rebid in a second suit that ranks below the
// opening [RO-19]. Both seats where that rebid arises -- over a suit response
// and over the 1SA "poubelle" -- go through here, the two having drifted apart
// once already: one capped the bid at 17 while the other announced 12-19 and
// tested no strength at all.
//
// The cheap bid is one partner is free to pass, so it cannot also carry the
// hands that want to hear from him: its ceiling is 17. From 18 the same two
// suits are shown one level higher, and the jump is what says so -- without it
// a nineteen-count and a twelve-count made the same call, and the responder had
// no way to tell them apart.
//
// Unless the auction is already forced to game, in which case there is nothing
// to distinguish: partner cannot pass the cheap bid, so it already carries the
// whole range, and the level the jump would spend is room the slam exploration
// needs. The zone announced there carries no ceiling, the bid limiting nothing.
func (e *Engine) cheapSecondSuit(p *playerState, s Suit, cheap Call, hl int) (Call, meaning) {
	economic := m(12, 17, "bicolore économique", "cheap second suit, 12-17").withLen(s, 4)
	if e.gameForce[sideOf(p.seat)] {
		return cheap, m(12, -1, "bicolore économique, sans plafond : le forcing de manche est déjà engagé",
			"cheap second suit, unlimited: the auction is already game forcing").withLen(s, 4)
	}
	if hl <= 17 {
		return cheap, economic
	}
	jump := bid(cheap.Level+1, s.Strain())
	if !e.legal(p.seat, jump) {
		return cheap, economic
	}
	if hl >= 20 {
		e.gameForce[sideOf(p.seat)] = true
	}
	return jump, m(18, 23, "saut dans la seconde couleur, forcing", "jump in the second suit, forcing").withLen(s, 4).asForcing()
}

// hldAgainstTheirBidding values the hand for the agreed trump, then takes
// back the distribution credit the auction has just made worthless: the
// shortness bonus of a **singleton honour in a suit the opponents have bid**.
// Their ace sits over it, the honour falls under it, and what remains is a
// bare singleton whose ruffing value the honour has already been paid for. A
// singleton small card keeps its credit -- nothing there was ever going to
// take a trick -- and so does a singleton honour in a suit nobody has named,
// where the ace may still be with partner.
//
// Without this, a singleton king facing opponents who have bid and raised the
// suit is worth five points to the engine -- three of honour, two of shortness
// -- and five points is exactly what turns a hand with no game into one that
// makes a game try.
// hAgainstTheirBidding counts the honours that are still worth their face
// value. It is the honour-point pendant of hldAgainstTheirBidding, and it
// takes back the same card: the **singleton honour in a suit the opponents
// have bid**. hldAgainstTheirBidding drops the shortness credit and leaves the
// honour standing, which is right when the count is a playing-strength one;
// when the count is a pure honour count the card itself is what has to go,
// because a bare king or queen under their opening is not three points or two,
// it is a card that falls under their ace.
//
// A singleton ace keeps everything -- it is a control, and controls do not
// care who bid the suit.
func (e *Engine) hAgainstTheirBidding(p *playerState, trump Suit) int {
	v := p.hand.H()
	for _, s := range e.opponentSuits(p) {
		if s == trump || p.hand.Len(s) != 1 {
			continue
		}
		switch {
		case p.hand.HasCard(s, 'K'):
			v -= 3
		case p.hand.HasCard(s, 'Q'):
			v -= 2
		case p.hand.HasCard(s, 'J'):
			v -= 1
		}
	}
	return v
}

func (e *Engine) hldAgainstTheirBidding(p *playerState, trump Suit) int {
	v := p.hand.HLD(trump)
	for _, s := range e.opponentSuits(p) {
		if s == trump || p.hand.Len(s) != 1 {
			continue
		}
		if p.hand.HasCard(s, 'K') || p.hand.HasCard(s, 'Q') || p.hand.HasCard(s, 'J') {
			v -= 2 // the singleton credit HLD granted
		}
	}
	return v
}

// helpSuitGameTry picks a side suit "needing support" for opener's natural
// game try after partner's simple raise of a major (docs/bidings.md, "EN FACE
// D'UN SOUTIEN MAJEUR SIMPLE"): a suit where opener holds losers partner
// could help cover (2-4 cards, no ace or king), rather than an already
// self-sufficient suit.
//
// A try is a question, and a question needs an answer that is not the game
// itself: the suit named must leave three of the trump available above it, so
// that partner without the help can sign off there. Named above that -- 3S on
// a heart fit -- the "try" only lets partner bid 4H or pass out a contract in
// a suit nobody agreed, which is not a decline but an accident.
func (e *Engine) helpSuitGameTry(p *playerState, trump Suit) (Call, meaning, bool) {
	h := p.hand
	signOff := bidSuit(3, trump)
	for s := Spades; s >= Clubs; s-- {
		if s == trump {
			continue
		}
		n := h.Len(s)
		if n < 2 || n > 4 || h.HasCard(s, 'A') || h.HasCard(s, 'K') {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if !signOff.higherThan(c) {
			continue
		}
		if c.Level <= 3 && e.legal(p.seat, c) {
			mn := m(17, 21, "essai de manche, couleur nécessitant un appui", "game try, suit needing help").withLen(s, n).asInvite()
			mn.helpSuitTry = true
			mn.helpSuit = s
			return c, mn, true
		}
	}
	return Call{}, meaning{}, false
}

func (e *Engine) rebidOverNewSuit(p *playerState, os, rs Suit, respLevel int) (Call, meaning) {
	h := p.hand
	hp := h.H()
	hl := h.HL()
	sup := h.Len(rs)
	tr := e.tr

	if tr.check(sup >= 4, "4 cartes dans la couleur du partenaire → soutien",
		"four cards in partner's suit → raise", cards(h, rs)) {
		hld := h.HLD(rs)
		tr.in()
		defer tr.out()
		base := respLevel + 1
		// A jump raise of responder's minor after a two-over-one lands at
		// the four level and bypasses 3NT, which with a minor fit is almost
		// always the better game (nine tricks against eleven). The
		// two-over-one already made the auction game-forcing, so the
		// strength-showing jump buys nothing and costs the 3NT contract:
		// reserve it for hands where slam is very likely (33 HLD combined
		// opposite the 11 HL the response promises) and otherwise raise
		// below 3NT with a widened range.
		minorPast3NT := !rs.IsMajor() && base+1 >= 4
		switch {
		case rs.IsMajor() && tr.check(hld >= 20, "20 HLD et plus → manche", "20+ HLD → game", pts(hld, "HLD")):
			return bidSuit(4, rs), m(20, 23, "soutien à la manche, 20HLD et plus", "raise to game, 20+ HLD").withLen(rs, 4)
		case minorPast3NT && tr.check(hld >= 22, "22 HLD : saut au-delà de 3SA, ambition de chelem",
			"22 HLD: jump past 3NT, slam ambition", pts(hld, "HLD")):
			return bidSuit(base+1, rs), m(22, 23, "soutien à saut au-delà de 3SA, ambition de chelem", "jump raise past 3NT, slam ambition").withLen(rs, 4).asForcing()
		case !minorPast3NT && tr.check(hld >= 17, "17-19 HLD → soutien à saut", "17-19 HLD → jump raise", pts(hld, "HLD")):
			return bidSuit(base+1, rs), m(17, 19, "soutien à saut, 17-19HLD", "jump raise, 17-19 HLD").withLen(rs, 4).asInvite()
		case minorPast3NT:
			tr.check(true, "soutien sans sauter, pour garder 3SA", "raise without jumping, keeping 3NT", pts(hld, "HLD"))
			return bidSuit(base, rs), m(12, 21, "soutien, palier de 3SA préservé", "raise, keeping 3NT available").withLen(rs, 4)
		default:
			tr.check(true, "12-16 HLD → soutien simple", "12-16 HLD → single raise", pts(hld, "HLD"))
			return bidSuit(base, rs), m(12, 16, "soutien simple, 12-16HLD", "single raise, 12-16 HLD").withLen(rs, 4)
		}
	}

	balanced := (h.IsRegular() || h.IsSemiRegular()) && h.Len(h.Longest()) <= 5
	if tr.check(balanced && hp >= 18 && hp <= 19, "régulière, 18-19 H → 2SA", "balanced, 18-19 H → 2NT",
		pts(hp, "H")+", "+shape(h)) {
		mn := m(18, 19, "2SA, 18-19H régulier", "2NT, 18-19 balanced")
		// After a minor opening and a one-level major response, the jump to
		// 2NT denies four cards in responder's major but may hold four in the
		// other one: 3C checkback applies (alert).
		if !os.IsMajor() && rs.IsMajor() && respLevel == 1 {
			mn = m(18, 19,
				"saut à 2SA, 18-19H régulier, sans 4 cartes dans votre majeure (alerte : 3T Checkback disponible)",
				"jump to 2NT, 18-19 balanced, no four-card support (alert: 3C checkback available)")
			mn.checkbackOffer = true
		}
		return bid(2, SNoTrump), mn
	}
	if respLevel == 1 && tr.check(balanced && hp <= 14, "régulière, 12-14 H → redemande à Sans-Atout",
		"balanced, 12-14 H → notrump rebid", pts(hp, "H")+", "+shape(h)) {
		// In the opponents' silence the balanced minimum rebids 1NT. Once
		// interference has taken that call away, the same hand can only show
		// itself at 2NT, which additionally promises a stopper in the enemy
		// suit (docs/bidings.md). Without a stopper, fall through to a suit rebid.
		if c := bid(1, SNoTrump); e.legal(p.seat, c) {
			tr.check(true, "1SA est encore possible → 1SA", "1NT is still available → 1NT", "")
			mn := m(12, 14, "redemande à 1SA, 12-14H régulier", "1NT rebid, 12-14 balanced")
			// In the quiet sequences 1m - 1M - 1SA and 1H - 1S - 1SA,
			// responder's 2C Roudi applies over this rebid to check the 5-3
			// major fit (docs/addon_9.md).
			if rs.IsMajor() && e.uncontested(p.seat) {
				mn = m(12, 14,
					"redemande à 1SA, 12-14H régulier (2T Roudi disponible)",
					"1NT rebid, 12-14 balanced (2C Roudi available)")
				mn.roudiOffer = true
			}
			return c, mn
		}
		stopped := true
		for _, adv := range e.opponentSuits(p) {
			if !h.Stopper(adv) {
				stopped = false
				break
			}
		}
		if tr.check(stopped, "intervention : arrêt dans la couleur adverse → 2SA",
			"interference: stopper in the enemy suit → 2NT", "") {
			if c := bid(2, SNoTrump); e.legal(p.seat, c) {
				mn := m(12, 14, "redemande à 2SA, 12-14H régulier avec arrêt dans la couleur adverse", "2NT rebid, 12-14 balanced, stopper in the enemy suit")
				for _, adv := range e.opponentSuits(p) {
					mn = mn.withStopper(adv)
				}
				return c, mn
			}
		}
	}
	// A 2/1 response already reads as competitive interference (the natural
	// 1-level answer was denied), so the balanced minimum rebid needs a
	// stopper in the intervention suit just as it does after a 1-level
	// response -- doubly so once the intervener's partner has also raised,
	// confirming real length there rather than a bare overcall.
	if respLevel == 2 && tr.check(balanced && hp <= 14 && e.ntSafe(p),
		"réponse au palier de 2 : régulière, 12-14 H, couleurs gardées → 2SA",
		"two-level response: balanced, 12-14 H, suits guarded → 2NT", pts(hp, "H")+", "+shape(h)) {
		mn := m(12, 14, "redemande à 2SA, 12-14H", "2NT rebid, 12-14")
		for _, adv := range e.opponentSuits(p) {
			mn = mn.withStopper(adv)
		}
		return bid(2, SNoTrump), mn
	}

	// Three-card support for responder's forcing 1-level major, in
	// competition [RO-18b]. In the opponents' silence opener paints shape
	// instead (1NT rebid, second suit...) and a raise stays reserved for
	// four trumps; once the intervention has taken the balanced descriptions
	// above away, a hand with values cannot pass a forcing response, so it
	// raises partner's major on the known 4-3 fit -- competitively with a
	// minimum, invitationally (jump) with extras, straight to game when the
	// support count alone is worth it.
	// The same holds after a two-over-one, where the response promised 11 HL
	// and the auction is forced: the raise lands at the three level, so the
	// simple one needs no more than the opening itself, and the jump is the
	// game.
	if sup == 3 && rs.IsMajor() && respLevel == 2 && tr.check(!e.uncontested(p.seat),
		"en compétition, 3 cartes dans la majeure du partenaire → soutien sur fit 4-3 [RO-18b]",
		"in competition, three cards in partner's major → raise on the 4-3 fit [RO-18b]", cards(h, rs)) {
		hld := e.hldAgainstTheirBidding(p, rs)
		switch {
		case hld >= 17 && e.legal(p.seat, bidSuit(4, rs)):
			return bidSuit(4, rs), m(17, 23, "soutien à la manche sur fit 4-3, la réponse promet 11HL", "raise to game on the 4-3 fit, the response promised 11 HL").withLen(rs, 3)
		case e.legal(p.seat, bidSuit(3, rs)):
			return bidSuit(3, rs), m(12, 16, "soutien sur fit 4-3 : la réponse 2 sur 1 est forcing", "raise on the 4-3 fit: the two-over-one is forcing").withLen(rs, 3)
		}
	}

	if sup == 3 && rs.IsMajor() && respLevel == 1 && tr.check(!e.uncontested(p.seat),
		"en compétition, 3 cartes dans la majeure du partenaire → soutien sur fit 4-3 [RO-18b]",
		"in competition, three cards in partner's major → raise on the 4-3 fit [RO-18b]", cards(h, rs)) {
		hld := e.hldAgainstTheirBidding(p, rs)
		base := respLevel + 1 // simple raise level (2)
		switch {
		case hld >= 20 && e.legal(p.seat, bidSuit(4, rs)):
			return bidSuit(4, rs), m(20, 23, "soutien à la manche sur fit 4-3, 20HLD et plus", "raise to game on the 4-3 fit, 20+ HLD").withLen(rs, 3)
		case hld >= 16 && e.legal(p.seat, bidSuit(base+1, rs)):
			p.invited = true
			return bidSuit(base+1, rs), m(16, 19, "soutien à saut sur fit 4-3, essai de manche", "jump raise on the 4-3 fit, game try").withLen(rs, 3).asInvite()
		case hld >= 13 && e.legal(p.seat, bidSuit(base, rs)):
			return bidSuit(base, rs), m(12, 15, "soutien compétitif sur fit 4-3", "competitive raise on the 4-3 fit").withLen(rs, 3)
		}
	}

	// Second suit.
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if s == os || s == rs || h.Len(s) < 4 {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if c.Level == 1 {
			tr.check(true, "4 cartes à "+suitSymbol[s]+", nommable au palier de 1 → nouvelle couleur",
				"four "+suitSymbol[s]+", biddable at the one level → new suit", cards(h, s))
			// A second suit named at the one level (1C-1H-1S) shows the shape
			// and denies a jump or a reverse, but says nothing about the point
			// level -- opener clarifies his strength next round. Only a
			// two-level second suit carries the "bicolore économique" 12-17
			// message [RO-19].
			return c, m(-1, -1, "changement de couleur au palier de 1", "new suit at the one level").withLen(s, 4)
		}
		if c.Level == 2 && s.Strain() < os.Strain() {
			tr.check(true, "4 cartes à "+suitSymbol[s]+", moins chère que l'ouverture → bicolore économique [RO-19]",
				"four "+suitSymbol[s]+", cheaper than the opening → economical two-suiter [RO-19]", cards(h, s))
			// The cheap second suit is a minimum bid partner is free to pass,
			// so it cannot also carry the hands that want to hear from him:
			// its ceiling is 17 [RO-19]. From 18 the same two suits are shown
			// one level higher, and the jump is what says so -- without it a
			// nineteen-count and a twelve-count made the same call, and the
			// responder had no way to tell them apart.
			//
			// Unless the auction is already forced to game, in which case
			// there is nothing to distinguish: partner cannot pass the cheap
			// bid, so it already carries the whole range, and the level the
			// jump would spend is room the slam exploration needs. The zone
			// announced there carries no ceiling, the bid limiting nothing.
			return e.cheapSecondSuit(p, s, c, hl)
		}
		if tr.check(hl >= 18 && c.Level <= 2,
			"4 cartes à "+suitSymbol[s]+", plus chère, 18 HL et plus → bicolore cher (forcing)",
			"four "+suitSymbol[s]+", higher-ranking, 18+ HL → reverse (forcing)", pts(hl, "HL")) {
			// The reverse names its second suit above the first, so the first
			// is at least as long: four cards there, not the bare three a
			// minor opening promises [RO-19]. Recording it matters -- without
			// it partner reads the opening's minimum and values his own
			// shortness facing the real suit as ruffing power rather than as
			// waste.
			mn := m(18, 23, "bicolore cher, forcing", "reverse, forcing").
				withLen(s, 4).withLen(os, 4).asForcing()
			mn.reverse = true
			return c, mn
		}
		if tr.check(hl >= 20 && c.Level <= 3, "20 HL et plus → bicolore à saut, forcing de manche",
			"20+ HL → jump shift, game forcing", pts(hl, "HL")) {
			e.gameForce[sideOf(p.seat)] = true
			return bid(c.Level, s.Strain()), m(20, 23, "bicolore à saut, forcing de manche", "jump shift, game forcing").withLen(s, 4).asForcing()
		}
	}
	if tr.check(h.Len(os) >= 6 && hl >= 17, "6 cartes et 17 HL → répétition à saut",
		"six cards and 17 HL → jump rebid", cards(h, os)+", "+pts(hl, "HL")) {
		c := e.cheapestCall(os.Strain())
		return bid(c.Level+1, os.Strain()), m(17, 19, "répétition à saut, bel unicolore", "jump rebid, good six-card suit").withLen(os, 6)
	}
	if tr.check(h.Len(os) >= 5, "5 cartes et plus dans la couleur d'ouverture → répétition",
		"five or more cards in the opened suit → rebid it", cards(h, os)) {
		c := e.cheapestCall(os.Strain())
		if c.Level <= 2 || respLevel >= 2 {
			// The two-level repetition normally shows six cards, but the
			// no-better-bid hands (15-17, semi-regular, no economic second
			// suit) sometimes repeat a five-card suit: record only the length
			// actually held, so partner never counts a fit on a phantom card.
			n := 5
			if c.Level >= 2 && h.Len(os) >= 6 {
				n = 6
			}
			return c, m(13, 17, "répétition de la couleur d'ouverture", "rebidding the opened suit").withLen(os, n)
		}
		// The intervention has pushed the repetition to the three level, and
		// has taken away every description above it -- the notrump rebids need
		// a stopper the hand may not hold, the second suit is not there. A
		// genuinely long suit still has to be shown: passing buries a source
		// of tricks partner has no other way to hear about, and leaves him
		// bidding notrump on a hand that belongs in the suit. The free bid is
		// paid for in playing strength rather than honours -- six cards and a
		// hand worth the level it costs.
		if h.Len(os) >= 6 && c.Level == 3 && !e.uncontested(p.seat) &&
			e.hldAgainstTheirBidding(p, os) >= 15 && e.legal(p.seat, c) {
			return c, m(12, 17, "répétition au palier de 3 sur intervention, 6 cartes et plus",
				"three-level rebid over the intervention, six cards or more").withLen(os, 6)
		}
	}
	c := e.cheapestCall(SNoTrump)
	if tr.check(c.Level <= 2 && e.ntSafe(p), "sinon, couleurs gardées → Sans-Atout par défaut",
		"otherwise, suits guarded → default notrump", "") {
		return c, m(12, 14, "redemande par défaut à Sans-Atout", "default notrump rebid")
	}
	tr.note("aucune redemande ne s'applique → Passe", "no rebid applies → Pass")
	return passCall, noInfo()
}

// openerReopen handles opener's second turn when responder passed over interference.
func (e *Engine) openerReopen(p *playerState, os Suit) (Call, meaning) {
	h := p.hand
	hp := h.H()
	last, lastSeat, hasBid := e.lastBid()
	if !hasBid || sideOf(lastSeat) == sideOf(p.seat) {
		return passCall, noInfo()
	}
	if hp >= 18 && last.Level <= 3 && e.legal(p.seat, doubleCall) {
		return doubleCall, m(18, 23, "contre de réveil, jeu fort", "strong reopening double").asForcing()
	}
	if h.Len(os) >= 6 {
		c := e.cheapestCall(os.Strain())
		if c.Level <= 3 {
			return c, m(12, 17, "répétition, 6 cartes", "rebid, six cards").withLen(os, 6)
		}
	}
	return passCall, noInfo()
}

// ---------- defenders ----------

func (e *Engine) overcall(p *playerState) (Call, meaning) {
	h := p.hand
	hp, hl := h.H(), h.HL()
	last, lastSeat, hasBid := e.lastBid()
	if !hasBid {
		return passCall, m(0, 11, "", "")
	}
	var oppSuit Suit
	hasOppSuit := false
	if last.Strain <= SSpades {
		oppSuit, hasOppSuit = Suit(last.Strain), true
	}
	tr := e.tr
	lastFR, lastEN := callSym(last)
	tr.note("l'adversaire a ouvert ("+lastFR+") : intervenir ou passer",
		"the opponents opened ("+lastEN+"): overcall or pass")

	// Classic balancing seat (docs/regles_moteur.md §9.3): the opponents' one-level suit
	// opening has come back after two passes, so a pass would end the
	// auction. Both opponents are limited there and partner owns part of the
	// missing strength: dedicated ranges apply, about six points below the
	// direct-seat ones.
	if hasOppSuit && last.Level == 1 && e.isClassicReopen() {
		return e.untraced(e.reopenBid(p, oppSuit))
	}

	// Specified Michaels cue-bid: a 5-5+ two-suiter over a one-level opening
	// (docs/addon_3.md), no upper limit on values.
	if hasOppSuit && last.Level == 1 {
		if c, mn, ok := e.michaelsShape(p, oppSuit); tr.check(ok,
			"bicolore 5-5 → cue-bid Michaels",
			"5-5 two-suiter → Michaels cue-bid", shape(h)) {
			return c, mn
		}
	}

	// Natural 1NT overcall [I-4]: 16-18 H, regular, and a stopper in their
	// suit, over a one-level opening. The stopper is not just a condition on
	// the bid, it is part of what the bid says -- "et arrêt" -- so it is
	// recorded: without that, partner's own notrump arithmetic keeps refusing
	// the contract for want of a stopper the overcall has already guaranteed.
	//
	// Tried before the strong takeout double, whose floor it overlaps at 18.
	// The double there promises nothing but points; this bid names a shape, a
	// stopper and a two-point band in one call, and a hand that fits it has
	// nothing to gain from the vaguer description. Above the band the double
	// takes over again.
	if last.Level == 1 && hasOppSuit && tr.check(hp >= 16 && hp <= 18 && h.IsRegular() && h.Stopper(oppSuit),
		"16-18 H, régulière, arrêt dans leur couleur → 1SA [I-4]",
		"16-18 H, balanced, stopper in their suit → 1NT [I-4]", pts(hp, "H")+", "+shape(h)) {
		c := bid(1, SNoTrump)
		if e.legal(p.seat, c) {
			return c, m(16, 18, "intervention à 1SA, 16-18H et arrêt", "1NT overcall, strong balanced with a stopper").withStopper(oppSuit)
		}
	}
	// Strong takeout double: from 18H the hand is beyond every natural
	// overcall, so the double carries no shape promise.
	if tr.check(hp >= 18 && e.legal(p.seat, doubleCall) && last.Level <= 4,
		"18 H et plus → contre « toutes distributions »",
		"18+ H → takeout double, any shape", pts(hp, "H")) {
		return doubleCall, m(18, 40, "contre \"toutes distributions\", 18H et plus", "takeout double, any shape, 18+")
	}
	// Landy: over an opposing 1NT opening, the seat right behind the opener
	// shows both majors in a single bid [I-3b]. Below the "any shape" double,
	// so the ceiling the convention announces stays honest.
	if e.inLandySeat() {
		if c, mn, ok := e.landyShape(p); tr.check(ok,
			"sur leur 1SA : les deux majeures (5-4 et plus), 10 HL → Landy 2♣ [I-3b]",
			"over their 1NT: both majors (5-4 or more), 10 HL → Landy 2♣ [I-3b]",
			cards(h, Hearts)+", "+cards(h, Spades)) {
			return c, mn
		}
	}
	// Natural suit overcall. Every suit the opponents named, whenever they did,
	// is off the table: repeating one of them is a cue-bid whatever our length,
	// and the suit is by definition theirs — 1C 1S 1NT leaves South no natural
	// 2S, however good its five spades look.
	var oppNamed [4]bool
	for _, s := range e.opponentSuits(p) {
		oppNamed[s] = true
	}
	// Over their 1NT, the cheapest club call is the Landy 2C [I-3b]: the seat
	// that owns the convention cannot also use it naturally. A club suit
	// there stays unbid -- the price of describing both majors in one bid,
	// and the reason the convention is worth its cost far more often than
	// the club hands it silences.
	landySeat := e.inLandySeat()
	var best Suit
	bestLen := 0
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if oppNamed[s] || (s == Clubs && landySeat) {
			continue
		}
		if h.Len(s) > bestLen && h.GoodSuit(s) {
			best, bestLen = s, h.Len(s)
		}
	}
	// Over their 1NT opening the natural overcall needs six cards [I-5b], and
	// no amount of honour strength substitutes for them. The opener has
	// announced 15-17 balanced, so the values are theirs: the intervention is
	// not a contract proposal but an act of obstruction, and it has to be paid
	// for in playing tricks. A five-card suit at the two level in front of a
	// hand that knows its own strength to two points is what a penalty double
	// is waiting for -- and the sixth card is exactly what turns the same
	// honours into the tricks that survive it.
	minLen := 5
	if last == bid(1, SNoTrump) && e.openCall == bid(1, SNoTrump) &&
		sideOf(lastSeat) != sideOf(p.seat) {
		minLen = 6
	}
	bestVal := ""
	if bestLen > 0 {
		bestVal = cards(h, best)
	}
	if tr.check(bestLen >= minLen,
		fmt.Sprintf("belle couleur de %d cartes et plus, libre → intervention à la couleur", minLen),
		fmt.Sprintf("good %d+ card suit, still free → suit overcall", minLen), bestVal) {
		c := e.cheapestCall(best.Strain())
		tr.in()
		switch {
		// At the one level the honours must carry the bid [I-5]: 8 H at
		// least, on top of the good suit GoodSuit already demands (two
		// honours, the ten included). Length points alone do not make a
		// sound overcall -- KJT76 T3 Q87654 - reaches 9 HL on 6 H and has no
		// business entering over 1C.
		case c.Level == 1 && tr.check(hl >= 9 && hp >= 8, "palier de 1 : 8 H et 9 HL → intervention [I-5]",
			"one level: 8 H and 9 HL → overcall [I-5]", pts(hp, "H")+", "+pts(hl, "HL")):
			tr.out()
			return c, m(9, 18, "intervention au palier de 1, 5 cartes et plus", "one-level overcall, 5+ cards").withLen(best, 5)
		case c.Level == 2 && tr.check(hl >= 11 && (bestLen >= 6 || hl >= 14),
			"palier de 2 : 11 HL avec six cartes, ou 14 HL → intervention",
			"two level: 11 HL with six cards, or 14 HL → overcall", bestVal+", "+pts(hl, "HL")):
			tr.out()
			return c, m(11, 18, "intervention au palier de 2", "two-level overcall").withLen(best, max(bestLen, 5))
		case c.Level == 3 && tr.check(hl >= 12 && bestLen >= 6,
			"palier de 3 : six cartes et 12 HL → intervention",
			"three level: six cards and 12 HL → overcall", bestVal+", "+pts(hl, "HL")):
			tr.out()
			return c, m(12, 18, "intervention au palier de 3", "three-level overcall").withLen(best, 6)
		}
		tr.out()
	}
	// Takeout double, from 12H behind a one-level suit opening; the shape
	// conditions depend on the opened suit (docs/addon_1.md). Over a minor the
	// floor counts the distribution (12 HL), over a major the honours alone
	// (12 H): the double of a major promises a precise four-card holding in the
	// other one, so length elsewhere buys nothing.
	if hp <= 17 && hasOppSuit && e.legal(p.seat, doubleCall) {
		floorReached := hl >= 12
		if oppSuit.IsMajor() {
			floorReached = hp >= 12
		}
		floorFR, floorEN, floorVal := "12 HL", "12 HL", pts(hl, "HL")
		if oppSuit.IsMajor() {
			floorFR, floorEN, floorVal = "12 H", "12 H", pts(hp, "H")
		}
		if last.Level == 1 && tr.check(floorReached && takeoutShape(h, oppSuit),
			floorFR+" et la forme du contre (court dans leur couleur, les autres majeures) → contre d'appel",
			floorEN+" and takeout shape (short in their suit, the other majors) → takeout double",
			floorVal+", "+shape(h)) {
			mn := m(12, 17, "contre d'appel, 12H et plus", "takeout double, 12+")
			for s := Clubs; s <= Spades; s++ {
				if s.IsMajor() && s != oppSuit {
					n := 3
					if oppSuit.IsMajor() {
						n = 4
					}
					mn = mn.withLen(s, n)
				}
			}
			return doubleCall, mn
		}
		// Over a two-level bid, keep the generic short-in-their-suit rule.
		if last.Level == 2 && tr.check(hp >= 12 && hp <= 17 && h.Len(oppSuit) <= 2,
			"sur un palier de 2 : 12-17 H, deux cartes au plus dans leur couleur → contre d'appel envisagé",
			"over a two-level bid: 12-17 H, two cards at most in their suit → takeout double considered",
			pts(hp, "H")+", "+cards(h, oppSuit)) {
			ok := true
			for s := Clubs; s <= Spades; s++ {
				if s != oppSuit && h.Len(s) < 3 {
					ok = false
				}
			}
			if tr.check(ok, "au moins 3 cartes dans chaque autre couleur → contre",
				"at least three cards in every other suit → double", shape(h)) {
				return doubleCall, m(12, 17, "contre d'appel", "takeout double")
			}
		}
	}
	// [I-5c] Five cards and an opening, at the one level, without the two
	// honours [I-5] asks for. The "belle couleur" is what pays for a bid made
	// on distribution alone; a hand that holds 12 H owns its share of the deal
	// and cannot be made to pass for want of a jack. The takeout double has
	// already had its turn above and, with five cards in a major, [I-6]
	// forbids it outright -- so the choice here is not between two
	// descriptions but between naming the suit and staying silent with an
	// opening. Only the one level: higher, the sixth card and the suit
	// quality are what make the bid safe, and neither is promised here.
	if last.Level == 1 && hasOppSuit && hp >= 12 {
		var alt Suit
		altLen := 0
		for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
			if oppNamed[s] || (s == Clubs && landySeat) {
				continue
			}
			if h.Len(s) >= 5 && h.Len(s) > altLen {
				alt, altLen = s, h.Len(s)
			}
		}
		altVal := ""
		if altLen > 0 {
			altVal = cards(h, alt)
		}
		if tr.check(altLen >= 5, "12 H et une cinquième, même moins belle → intervention au palier de 1 [I-5c]",
			"12 H and a five-card suit, even a poorer one → one-level overcall [I-5c]", altVal) {
			if c := e.cheapestCall(alt.Strain()); c.Level == 1 && e.legal(p.seat, c) {
				return c, m(12, 17, "intervention au palier de 1, 5 cartes et l'ouverture",
					"one-level overcall, five cards and opening values").withLen(alt, 5)
			}
		}
	}
	tr.note("aucune intervention ne s'applique → Passe", "no overcall applies → Pass")
	return passCall, m(0, 14, "", "")
}

// ---------- balancing seat (réveil) ----------

// isClassicReopen reports the classic balancing position: the last two calls
// are passes and the auction holds a single bid, the opponents' opening.
func (e *Engine) isClassicReopen() bool {
	n := len(e.calls)
	if n < 3 || e.calls[n-1].Call.Kind != KindPass || e.calls[n-2].Call.Kind != KindPass {
		return false
	}
	acts := 0
	for _, sc := range e.calls {
		if sc.Call.Kind != KindPass {
			acts++
		}
	}
	return acts == 1
}

// reopenBid chooses the balancing call over the opponents' dying one-level
// opening (docs/regles_moteur.md §9.3). Three families share the seat and none
// of them means what the same call would mean in the direct seat: a suit, with
// or without a jump, denies opening values; the notrump bids are limited
// (9-13 HL for 1SA, 17-19 for 2SA) and ask for a hold in the opened suit; and
// the double covers three hands with nothing in common -- the takeout shape
// short in their suit, where 8 H is enough, opening values with no notrump bid
// available, and the balanced 14-16 that follows its double with the cheapest
// notrump.
func (e *Engine) reopenBid(p *playerState, opp Suit) (Call, meaning) {
	h := p.hand
	hp, hl := h.H(), h.HL()
	balanced := h.IsRegular() || h.IsSemiRegular()

	// Two-suited reopenings first, as in the direct seat [V-1].
	if c, mn, ok := e.reopenTwoSuiter(p, opp); ok {
		return c, mn
	}
	// 2SA is natural in the balancing seat: 17-19 HL with a stopper [V-2].
	if hl >= 17 && hl <= 19 && balanced && h.Stopper(opp) {
		return bid(2, SNoTrump), m(17, 19, "réveil de 2SA naturel, 17-19HL régulier avec arrêt", "natural 2NT reopening, 17-19 HL balanced with a stopper").withStopper(opp).asReopen()
	}
	// From 14 HL the hand holds an opening, and every suit reopening denies
	// one: the double is compulsory [V-3]. The balanced 14-16 with a hold
	// plans the notrump rebid that alone pins its zone -- the double on its
	// own would leave partner reading the 8 H takeout shape.
	if hl >= 14 && e.legal(p.seat, doubleCall) {
		if c, mn, ok := e.reopenLongSuit(p, opp); ok {
			return c, mn
		}
		if hp >= 14 && hp <= 16 && balanced && h.reopenHold(opp) {
			p.planned = func() (Call, meaning) { return e.reopenDoubleNTRebid(p, opp) }
			return doubleCall, m(14, 16, "contre de réveil, jeu régulier de 14-16H : sera suivi d'une enchère à Sans-Atout", "reopening double, 14-16 balanced: a notrump bid will follow").asReopen()
		}
		return doubleCall, m(14, 40, "contre de réveil, la valeur de l'ouverture et pas d'enchère à Sans-Atout possible", "reopening double, opening values and no notrump bid available").asReopen()
	}
	// Jump reopenings [V-5] before the plain ones: with six cards in a major
	// the barrage is the whole point of the seat.
	if c, mn, ok := e.reopenJump(p, opp); ok {
		return c, mn
	}
	// A good five-card suit biddable at the one level outranks the notrump
	// bids: a suit partner can raise is worth more than a limited 1SA.
	if c, mn, ok := e.reopenSuit(p, opp, 1); ok {
		return c, mn
	}
	// 1SA de réveil: 9-13 HL with a stopper or three small cards [V-4].
	if hl >= 9 && hl <= 13 && balanced && h.reopenHold(opp) {
		if c := e.cheapestCall(SNoTrump); c.Level == 1 {
			mn := m(9, 13, "réveil de 1SA, 9-13HL, arrêt (ou trois petites cartes) dans la couleur d'ouverture", "1NT reopening, 9-13 HL, a stopper (or three small cards) in the opened suit").asReopen()
			if h.Stopper(opp) {
				mn = mn.withStopper(opp)
			}
			return c, mn
		}
	}
	// The same five-card suit, one rung higher: below 1SA, since the two-level
	// reopening buys less and costs more.
	if c, mn, ok := e.reopenSuit(p, opp, 2); ok {
		return c, mn
	}
	// Reopening double on the takeout shape: short in the opened suit, three
	// cards or more everywhere else, and 8 H is enough [V-7].
	if hp >= 8 && e.legal(p.seat, doubleCall) && takeoutReopenShape(h, opp) {
		return doubleCall, m(8, 13, "contre d'appel en réveil, dès 8H, court dans la couleur d'ouverture", "reopening takeout double, from 8 H, short in the opened suit").asReopen()
	}
	// Exceptionally, a strong four-card suit biddable at the one level [V-8].
	if hp >= 9 {
		for _, s := range []Suit{Spades, Hearts, Diamonds} {
			if s == opp || h.Len(s) != 4 || h.SuitH(s) < 5 {
				continue
			}
			if c := e.cheapestCall(s.Strain()); c.Level == 1 && !reopenReserved(c, opp) {
				return c, m(9, 13, "réveil par une belle couleur quatrième au palier de un", "one-level reopening in a strong four-card suit").withLen(s, 4).asReopen()
			}
		}
	}
	return passCall, m(0, 13, "", "")
}

// reopenHold reports the holding the balancing notrump bids ask for in the
// opened suit: a real stopper, or three small cards -- length enough that the
// opener has to spend his own suit before it runs, which is all the balancing
// 1SA needs when partner's values sit behind him.
func (h *Hand) reopenHold(opp Suit) bool {
	return h.Stopper(opp) || h.Len(opp) >= 3
}

// takeoutReopenShape reports the ideal shape for the balancing takeout double
// [V-7]: at most two cards in the opened suit, three or more everywhere else.
func takeoutReopenShape(h *Hand, opp Suit) bool {
	if h.Len(opp) > 2 {
		return false
	}
	for s := Clubs; s <= Spades; s++ {
		if s != opp && h.Len(s) < 3 {
			return false
		}
	}
	return true
}

// reopenReserved reports whether the balancing two-suiter scheme [V-1] has
// already booked a call: the cue-bid always, plus 3T over a major opening and
// 2K over a minor one. A natural reopening must never borrow one of them --
// partner would read the two-suiter and bid a suit nobody holds.
func reopenReserved(c Call, opp Suit) bool {
	if c.Strain > SSpades {
		return false
	}
	s := Suit(c.Strain)
	switch {
	case s == opp:
		return true
	case opp.IsMajor():
		return c.Level == 3 && s == Clubs
	default:
		return c.Level == 2 && s == Diamonds
	}
}

// reopenSuit names a good five-card suit without a jump: 8-13 HL, denying an
// opening [V-6]. level selects the rung the call must land on, so the
// one-level reopening can outrank the notrump bids while the two-level one
// stays below them.
func (e *Engine) reopenSuit(p *playerState, opp Suit, level int) (Call, meaning, bool) {
	h := p.hand
	if hl := h.HL(); hl < 8 || hl > 13 {
		return Call{}, meaning{}, false
	}
	var best Suit
	bestLen := 0
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if s == opp || h.Len(s) < 5 || !h.GoodSuit(s) {
			continue
		}
		if h.Len(s) > bestLen {
			best, bestLen = s, h.Len(s)
		}
	}
	if bestLen == 0 {
		return Call{}, meaning{}, false
	}
	c := e.cheapestCall(best.Strain())
	if c.Level != level || reopenReserved(c, opp) || !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	mn := m(8, 13, "réveil par une couleur, 8-13HL et 5 cartes, dénie l'ouverture", "suit reopening, 8-13 HL and five cards, denies an opening hand").withLen(best, 5).asReopen()
	return c, mn, true
}

// reopenLongSuit names a good six-card side suit at the two level or lower
// when the hand has reached the 14 HL that makes the double compulsory [V-3].
// The double asks partner to pick a suit; a hand with six of its own has
// nothing to ask, and understating the strength is a far cheaper lie than
// promising support in three suits it does not hold.
func (e *Engine) reopenLongSuit(p *playerState, opp Suit) (Call, meaning, bool) {
	h := p.hand
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if s == opp || h.Len(s) < 6 || !h.GoodSuit(s) {
			continue
		}
		c := e.cheapestCall(s.Strain())
		if c.Level > 2 || reopenReserved(c, opp) || !e.legal(p.seat, c) {
			continue
		}
		mn := m(14, 18, "réveil par une couleur sixième, la valeur de l'ouverture sans la forme du contre", "six-card suit reopening, opening values without the shape for a double").withLen(s, 6).asReopen()
		return c, mn, true
	}
	return Call{}, meaning{}, false
}

// reopenJump handles the jump reopenings [V-5], and the two families it covers
// pull in opposite directions. In a major the jump is a barrage and nothing
// else: six cards to the two level (a good weak two), seven to the three level
// (a three-level opening), the honour strength sitting in the suit itself. In
// a minor there is no barrage to make -- the jump shows a good six-card suit
// at the very limit of an opening, 11 HL and up. Calls the two-suiter scheme
// has already booked [V-1] are skipped rather than borrowed.
func (e *Engine) reopenJump(p *playerState, opp Suit) (Call, meaning, bool) {
	h := p.hand
	hp, hl := h.H(), h.HL()
	preemptive := hp <= 11 && hl <= 12
	for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
		if s == opp || h.Len(s) < 6 || !h.GoodSuit(s) {
			continue
		}
		base := e.cheapestCall(s.Strain())
		var c Call
		var mn meaning
		switch {
		case s.IsMajor():
			if !preemptive || h.SuitH(s)*2 < hp {
				continue
			}
			if h.Len(s) >= 7 {
				c = bid(3, s.Strain())
				mn = m(5, 12, "réveil à saut au palier de 3, barrage, 7 cartes", "three-level jump reopening, preempt, seven cards").withLen(s, 7)
			} else {
				c = bid(2, s.Strain())
				mn = m(5, 12, "réveil à saut au palier de 2, barrage, 6 cartes, l'équivalent d'un beau 2 faible", "two-level jump reopening, preempt, six cards, the equivalent of a good weak two").withLen(s, 6)
			}
		default:
			if hl < 11 {
				continue
			}
			c = bid(base.Level+1, s.Strain())
			mn = m(11, 13, "réveil à saut en mineure, belle couleur sixième à la limite de l'ouverture", "minor-suit jump reopening, good six-card suit at the limit of an opening").withLen(s, 6)
		}
		if c.Level <= base.Level || reopenReserved(c, opp) || !e.legal(p.seat, c) {
			continue
		}
		return c, mn.asReopen(), true
	}
	return Call{}, meaning{}, false
}

// reopenDoubleNTRebid is the second half of the balanced 14-16 reopening
// double [V-3]: over partner's minimum answer the doubler names the cheapest
// notrump, which is what tells this hand apart from the 8 H takeout shape the
// double also covers. Facing anything better than a minimum the description is
// already out of date, and the generic endgame takes over.
func (e *Engine) reopenDoubleNTRebid(p *playerState, opp Suit) (Call, meaning) {
	partner := e.ps[partnerOf(p.seat)]
	c := e.cheapestCall(SNoTrump)
	if partner.lastM != nil && partner.lastM.maxPts >= 0 && partner.lastM.maxPts <= 10 &&
		c.Level <= 2 && e.legal(p.seat, c) {
		mn := m(14, 16, "Sans-Atout sur le contre de réveil : jeu régulier de 14-16H", "notrump over the reopening double: balanced 14-16")
		if p.hand.Stopper(opp) {
			mn = mn.withStopper(opp)
		}
		return c, mn
	}
	return e.conclude(p)
}

// reopenTwoSuiter looks for a two-suited reopening [V-1]. In
// the balancing seat 2NT is natural (17-18), so the direct-seat Michaels
// scheme shifts: over 1C the cue-bid shows the two cheapest suits
// (diamonds-hearts) and 2D both majors; over 1D, 2D shows both majors (a
// spade two-suiter is otherwise bid naturally); over a major, the cue-bid
// shows the other major and clubs, and 3C the other major and diamonds.
func (e *Engine) reopenTwoSuiter(p *playerState, opened Suit) (Call, meaning, bool) {
	h := p.hand
	mk := func(c Call, s1, s2 Suit, fr, en string) (Call, meaning, bool) {
		if !e.legal(p.seat, c) {
			return Call{}, meaning{}, false
		}
		mn := m(9, 40, fr, en).withLen(s1, 5).withLen(s2, 5).asForcing().asMichaels(s1, s2).asReopen()
		return c, mn, true
	}
	if opened.IsMajor() {
		otherMajor := Spades
		if opened == Spades {
			otherMajor = Hearts
		}
		switch {
		case h.michaelsCandidate(otherMajor, Clubs):
			return mk(bidSuit(2, opened), otherMajor, Clubs,
				"réveil bicolore : cue-bid, 5+ cartes dans l'autre majeure et 5+ à Trèfle (alerte)",
				"two-suited reopening: cue-bid, 5+ cards in the other major and 5+ clubs (alertable)")
		case h.michaelsCandidate(otherMajor, Diamonds):
			return mk(bid(3, SClubs), otherMajor, Diamonds,
				"réveil bicolore : 3T, 5+ cartes dans l'autre majeure et 5+ à Carreau (alerte)",
				"two-suited reopening: 3C, 5+ cards in the other major and 5+ diamonds (alertable)")
		}
		return Call{}, meaning{}, false
	}
	switch {
	case h.michaelsCandidate(Spades, Hearts):
		return mk(bid(2, SDiamonds), Spades, Hearts,
			"réveil bicolore : 2K, 5+ cartes dans les deux majeures (alerte)",
			"two-suited reopening: 2D, 5+ cards in both majors (alertable)")
	case opened == Clubs && h.michaelsCandidate(Diamonds, Hearts):
		return mk(bid(2, SClubs), Diamonds, Hearts,
			"réveil bicolore : cue-bid, 5+ cartes à Carreau et à Cœur (alerte)",
			"two-suited reopening: cue-bid, 5+ cards in diamonds and hearts (alertable)")
	}
	return Call{}, meaning{}, false
}

// ---------- specified Michaels cue-bid ----------

// michaelsCandidate reports whether the hand qualifies for a specified
// Michaels two-suiter (docs/addon_3.md): always 5+-5+, never 5-4, both
// suits of playing quality, 9 HL and up with no upper limit.
func (h *Hand) michaelsCandidate(s1, s2 Suit) bool {
	return h.Len(s1) >= 5 && h.Len(s2) >= 5 && h.GoodSuit(s1) && h.GoodSuit(s2) && h.HL() >= 9
}

// michaelsShape looks for a specified Michaels two-suiter over a one-level
// opening. The pair of suits shown depends on the suit opened:
//   - over a major, the cue-bid shows the other major and clubs, the jump to
//     3C shows the other major and diamonds, and 2NT shows both minors;
//   - over a minor, 2D always shows both majors, and 2NT shows the other
//     minor and hearts (a spade two-suiter is bid naturally in that case).
func (e *Engine) michaelsShape(p *playerState, opened Suit) (Call, meaning, bool) {
	h := p.hand
	mk := func(c Call, s1, s2 Suit, fr, en string) (Call, meaning, bool) {
		if !e.legal(p.seat, c) {
			return Call{}, meaning{}, false
		}
		mn := m(9, 40, fr, en).withLen(s1, 5).withLen(s2, 5).asForcing().asMichaels(s1, s2)
		return c, mn, true
	}
	if opened.IsMajor() {
		otherMajor := Spades
		if opened == Spades {
			otherMajor = Hearts
		}
		switch {
		case h.michaelsCandidate(otherMajor, Clubs):
			return mk(bidSuit(2, opened), otherMajor, Clubs,
				"cue-bid Michaël précisé, 5+ cartes dans l'autre majeure et 5+ à Trèfle (alerte)",
				"specified Michaels cue-bid, 5+ cards in the other major and 5+ clubs (alertable)")
		case h.michaelsCandidate(otherMajor, Diamonds):
			return mk(bid(3, SClubs), otherMajor, Diamonds,
				"saut Michaël précisé, 5+ cartes dans l'autre majeure et 5+ à Carreau (alerte)",
				"Michaels jump to 3C, 5+ cards in the other major and 5+ diamonds (alertable)")
		case h.michaelsCandidate(Clubs, Diamonds):
			return mk(bid(2, SNoTrump), Clubs, Diamonds,
				"2SA Michaël précisé, 5+ cartes dans les deux mineures (alerte)",
				"2NT Michaels, 5+ cards in both minors (alertable)")
		}
		return Call{}, meaning{}, false
	}
	otherMinor := Diamonds
	if opened == Diamonds {
		otherMinor = Clubs
	}
	switch {
	case h.michaelsCandidate(Spades, Hearts):
		return mk(bid(2, SDiamonds), Spades, Hearts,
			"2K Michaël précisé, 5+ cartes dans les deux majeures (alerte)",
			"2D Michaels, 5+ cards in both majors (alertable)")
	case h.michaelsCandidate(otherMinor, Hearts):
		return mk(bid(2, SNoTrump), otherMinor, Hearts,
			"2SA Michaël précisé, 5+ cartes à Cœur et dans l'autre mineure (alerte)",
			"2NT Michaels, 5+ cards in hearts and the other minor (alertable)")
	}
	return Call{}, meaning{}, false
}

// advanceMichaels answers partner's specified Michaels two-suiter: picks the
// longer of the two shown suits (ties favour the major), then bids to the
// level supported by the law of total tricks. A double misfit once the
// opponents have shown their own fit is the one case where staying low (or
// doubling) is safer than guessing a suit (docs/addon_3.md).
func (e *Engine) advanceMichaels(p *playerState, pm *meaning) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	s1, s2 := pm.mSuits[0], pm.mSuits[1]
	best, worst := s1, s2
	if h.Len(s2) > h.Len(s1) || (h.Len(s2) == h.Len(s1) && s2.IsMajor() && !s1.IsMajor()) {
		best, worst = s2, s1
	}
	lBest, lWorst := h.Len(best), h.Len(worst)
	n3Bid := len(e.calls) > 0 && e.calls[len(e.calls)-1].Call.Kind != KindPass

	if lBest <= 2 && lWorst <= 2 && n3Bid {
		return passCall, m(-1, -1, "aucun fit dans les couleurs annoncées, sécurité distributionnelle", "no fit in either suit shown, staying safe")
	}
	if n3Bid && lBest == 3 && hl < 8 {
		return passCall, m(0, 7, "3 cartes seulement dans une des couleurs, pas assez de points pour parler", "only three-card support, not enough points to compete")
	}

	level := 2
	trumps := lBest + 5
	switch {
	case hl >= 13 || trumps >= 10:
		level = 4
	case hl >= 10 || trumps >= 9:
		level = 3
	}
	c := e.cheapestCall(best.Strain())
	if level > c.Level {
		c = bid(level, best.Strain())
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	mn := m(-1, -1, "soutien de la couleur Michaël la plus longue, loi des atouts", "raise of the longer Michaels suit, law of total tricks").withLen(best, lBest)
	if hl >= 13 {
		mn = mn.asForcing()
	}
	return c, mn
}

// ---------- Landy (intervention sur 1SA) ----------

// inLandySeat reports whether the auction stands at the one seat the Landy
// convention lives in: the call about to be made is the one immediately over
// an opposing 1NT opening, before it has been answered. Whose call it is
// follows from that -- the engine only ever decides in turn -- so the test is
// about the auction, not about the hand. It gates the convention itself and,
// just as importantly, the natural 2C the convention takes away: the two must
// agree, or the pair would be playing two systems at once.
func (e *Engine) inLandySeat() bool {
	n := len(e.calls)
	if n == 0 {
		return false
	}
	prev := e.calls[n-1]
	return prev.Seat == e.opener && prev.Call == bid(1, SNoTrump)
}

// landyShape looks for the Landy 2C overcall. It is the intervention of the
// player seated right behind the 1NT opener: one single, cheap bid describes
// a major two-suiter -- at least 5-4 in the majors -- from about ten points
// up, where no natural call could show both suits below the opponents' game.
// The bid is only available in that seat, over that opening: further round
// the table the 1NT has already been answered and the auction is no longer
// the one the convention addresses.
func (e *Engine) landyShape(p *playerState) (Call, meaning, bool) {
	if !e.inLandySeat() {
		return Call{}, meaning{}, false
	}
	h := p.hand
	lh, ls := h.Len(Hearts), h.Len(Spades)
	// At least 5-4: four cards in each major, nine in the two together.
	if lh < 4 || ls < 4 || lh+ls < 9 {
		return Call{}, meaning{}, false
	}
	if h.HL() < 10 {
		return Call{}, meaning{}, false
	}
	c := bid(2, SClubs)
	if !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	p.planned = func() (Call, meaning) { return e.afterLandy(p) }
	mn := m(10, 18,
		"2T Landy : bicolore majeur au moins 5-4, une dizaine de points (alerte)",
		"2C Landy: both majors, at least 5-4, about ten points (alertable)").
		withLen(Hearts, 4).withLen(Spades, 4).asForcing().asLandy()
	return c, mn, true
}

// advanceLandy answers partner's Landy 2C: name the longer major, hearts at
// equal length so the 5-4 hand with the spades can still correct at the two
// level (see afterLandy). Only a four-card major is promised in each suit,
// so the law of total tricks counts one trump less than after a Michaels
// 5-5, and the answer records the length actually held -- that is what tells
// partner whether the hearts are a preference or a default.
func (e *Engine) advanceLandy(p *playerState) (Call, meaning) {
	h := p.hand
	lh, ls := h.Len(Hearts), h.Len(Spades)
	// Neither major and real clubs: 2C is playable as it stands, and the
	// seven-card fit the correction would find is worth less than passing.
	if lh <= 2 && ls <= 2 && h.Len(Clubs) >= 5 {
		return passCall, m(-1, -1,
			"aucune majeure et de beaux Trèfles : 2T se joue tel quel",
			"no major and real clubs: 2C plays as it stands")
	}
	best := Hearts
	if ls > lh {
		best = Spades
	}
	lBest := h.Len(best)
	hld := h.HLD(best)
	// The law of total tricks may push the level, but the zone announced
	// stays the one actually held: a jump made on trump length alone must
	// not read as an invitation partner would accept.
	level := 2
	trumps := lBest + 4
	switch {
	case hld >= 13 || trumps >= 10:
		level = 4
	case hld >= 10 || trumps >= 9:
		level = 3
	}
	zoneMin, zoneMax := 0, 9
	switch {
	case hld >= 13:
		zoneMin, zoneMax = 13, 40
	case hld >= 10:
		zoneMin, zoneMax = 10, 12
	}
	c := e.cheapestCall(best.Strain())
	if level > c.Level {
		c = bid(level, best.Strain())
	}
	// The level computed above is what the fit and the zone pay for. When the
	// opponents have already climbed past it, the cheapest call in the major
	// is no longer that bid: choosing the longer major is an obligation of
	// the convention, not a licence to buy any level the auction has reached.
	// Answering 3SA with four of a major on a doubleton -- the law counting a
	// fit that is six cards long, not eight -- is a phantom sacrifice, and a
	// contract nobody's hand ever promised.
	if c.Level > level {
		return passCall, m(-1, -1,
			"le palier dépasse ce que le fit et la zone paient : on les laisse jouer",
			"the level has climbed past what the fit and the zone pay for: let them play")
	}
	if !e.legal(p.seat, c) {
		return passCall, noInfo()
	}
	return c, m(zoneMin, zoneMax,
		"choix de la majeure la plus longue sur le Landy, loi des atouts",
		"picking the longer major over the Landy overcall, law of total tricks").withLen(best, lBest)
}

// afterLandy is the Landy bidder's own follow-up, and the reason hearts win
// the ties: a 2H answer that shows no more than three of them is a choice
// made for want of anything better, so the hand with five spades and only
// four hearts corrects to 2S -- an eight-card fit for the price of a
// seven-card one. A 2H answer holding four hearts is a genuine preference
// and is left alone. Anything else goes back to the generic decision.
func (e *Engine) afterLandy(p *playerState) (Call, meaning) {
	h := p.hand
	partner := e.ps[partnerOf(p.seat)]
	if psc, ok := e.lastCallBy(partner.seat); ok && psc.Call == bidSuit(2, Hearts) &&
		partner.shownLens[Hearts] <= 3 && h.Len(Spades) >= 5 && h.Len(Hearts) == 4 {
		c := bidSuit(2, Spades)
		if e.legal(p.seat, c) {
			return c, m(-1, -1,
				"rectification à 2P : 5 cartes à Pique, 4 seulement à Cœur",
				"correction to 2S: five spades, only four hearts").withLen(Spades, 5)
		}
	}
	return e.conclude(p)
}

// takeoutShape checks the distribution conditions for a takeout double of a
// one-level suit opening (docs/addon_1.md). Over a minor: seven or eight cards
// in the majors, i.e. 4-3 or 4-4 (never two, never five cards in a major); the
// 4-3-3-3 keeps the double, only real length in the opened minor rules it out.
// Over a major: shortness -- at most two cards -- in the opened suit and four
// cards in the other major, five hearts being tolerated over 1S since the good
// ones have already been overcalled naturally.
//
// The four-card holding in the other major is not negotiable: doubling with a
// short one because the hand holds opening values is exactly what lands the
// partner in a contract that goes down. Passing on a strong hand costs far
// less than promising a shape one does not have, and the pass is not final --
// the balancing seat may still speak.
func takeoutShape(h *Hand, opp Suit) bool {
	lh, ls := h.Len(Hearts), h.Len(Spades)
	switch opp {
	case Clubs, Diamonds:
		return h.Len(opp) <= 3 && lh >= 3 && lh <= 4 && ls >= 3 && ls <= 4 && lh+ls >= 7
	case Hearts:
		return lh <= 2 && ls >= 4
	default: // Spades
		return ls <= 2 && lh >= 4 && lh <= 5
	}
}

// advanceOwnSuit picks the suit the advance may name naturally over partner's
// overcall: the longest good five-card-plus suit that is neither partner's --
// where a bid would read as a preference rather than as news -- nor one the
// opponents have named, which is theirs to play and never our contract.
// Reading the longest suit alone left the hand mute whenever that suit
// happened to be one of those two.
func (e *Engine) advanceOwnSuit(p *playerState, partnerSuit Suit) (Suit, bool) {
	h := p.hand
	theirs := e.opponentSuits(p)
	best, n := Clubs, 0
	for _, s := range []Suit{Clubs, Diamonds, Hearts, Spades} {
		// Ties go to the higher-ranking suit, as Hand.Longest reads them.
		if s == partnerSuit || h.Len(s) < n || !h.GoodSuit(s) {
			continue
		}
		if slices.Contains(theirs, s) {
			continue
		}
		best, n = s, h.Len(s)
	}
	return best, n >= 5
}

func (e *Engine) advance(p *playerState) (Call, meaning) {
	h := p.hand
	hl := h.HL()
	partner := e.ps[partnerOf(p.seat)]
	psc, _ := e.lastCallBy(partner.seat)
	tr := e.tr

	// The opponents' last bid is a game they are driving to: before the
	// generic advances (which never compete past the three level), weigh a
	// surenchère or a law-of-total-tricks sacrifice on the scoring table --
	// the silent advancer never reaches conclude, so the shared evaluation
	// is called from here.
	if ctx := e.concludeContext(p); ctx.hasBid && !ctx.ours && ctx.hasFit &&
		isGame(ctx.last) && partner.bids > 0 && !e.bw[ctx.side].asked {
		if c, mn, ok := e.competitiveSacrifice(ctx); ok {
			return e.untraced(c, mn)
		}
	}
	// After partner's takeout double, p must answer unless its right-hand
	// opponent has bid over the double (the call immediately before p's turn),
	// which frees p to pass. The opening bid that provoked the double is not
	// such a bid: reading it from lastBid() wrongly frees a forced advancer.
	freePosition := false
	if n := len(e.calls); n > 0 {
		rho := e.calls[n-1]
		freePosition = rho.Call.IsBid() && sideOf(rho.Seat) != sideOf(p.seat)
	}

	if psc.Call.Kind == KindDouble {
		// The suit taken out is whatever partner's double followed -- the
		// original opening for an immediate double, but a later reopening or
		// balancing bid when the double answers that instead (docs/addon_1.md,
		// réveil section): e.openCall would wrongly keep pointing at the
		// original opening in that second case.
		oppSuit, _ := e.doubledSuit(partner.seat)
		// A balancing double is not the direct-seat one: it may hold 8 H, and
		// the three zones of §9.2 would overbid it. Its own ladder [§9.5]
		// answers as though partner were weak.
		if psc.M.reopen {
			return e.untraced(e.answerReopenDouble(p, oppSuit, !freePosition))
		}
		// The answer to a takeout double is obligatory only while the
		// right-hand opponent stays silent. Once it has bid over the double
		// (freePosition), the advance is free: a hand worth no more than the
		// minimum answer passes instead of being forced to describe.
		fr, en := callSym(bidSuit(1, oppSuit))
		tr.note("le partenaire a contré l'ouverture ("+fr+"…) : réponse au contre d'appel",
			"partner doubled the opening ("+en+"…): answering the takeout double")
		return e.answerDouble(p, oppSuit, !freePosition)
	}
	// Partner's Landy 2C: like the Michaels cue-bid below, the strain bid is
	// not the suit shown, so it is resolved before the generic branches.
	if partner.lastM != nil && partner.lastM.landy {
		return e.untraced(e.advanceLandy(p))
	}
	// Partner's specified Michaels two-suiter: the call's own strain is not
	// partner's suit, so this must be resolved before the generic branches
	// below (docs/addon_3.md).
	if partner.lastM != nil && partner.lastM.michaels {
		return e.untraced(e.advanceMichaels(p, partner.lastM))
	}
	// Partner reopened in a suit: the balancing zones are not the direct-seat
	// ones, and neither are the answers [§9.5]. The notrump réveils (1SA
	// 9-13 HL, 2SA 17-19) keep the generic route just below, which is exactly
	// "les mêmes principes que sur l'intervention par 1SA": partner's zone is
	// recorded, and the endgame adds it up.
	if psc.M.reopen && psc.Call.IsBid() && psc.Call.Strain <= SSpades {
		return e.untraced(e.advanceReopenSuit(p, Suit(psc.Call.Strain)))
	}
	// Partner overcalled in notrump (15-17 balanced): show a running suit
	// first, otherwise add up and conclude.
	if psc.Call.IsBid() && psc.Call.Strain == SNoTrump {
		tr.note("le partenaire est intervenu à Sans-Atout", "partner overcalled in notrump")
		if c, mn, ok := e.advanceNotrumpLongSuit(p); tr.check(ok,
			"une longue couleur qui rapporte des levées → la montrer",
			"a long suit bringing tricks → show it", shape(h)) {
			return c, mn
		}
		return e.untraced(e.conclude(p))
	}
	if psc.Call.IsBid() && psc.Call.Strain <= SSpades {
		s := Suit(psc.Call.Strain)
		sup := h.Len(s)
		// « Il faut entre 7/8 et 12 HLD pour un soutien simple, avec des
		// points *utiles* » : les trois zones de [A-5] et [A-6] se comptent
		// contre les enchères adverses. L'ouverture adverse a nommé une
		// couleur, et le singleton à honneur qu'on y détient est le seul
		// point que HLD paie deux fois pour rien — l'honneur tombe sous
		// leur As, et la coupe a déjà été payée par l'honneur. Sans ce
		// retrait, une Dame sèche dans leur couleur vaut quatre points, de
		// quoi faire franchir à la main la barre du cue-bid de force.
		hld := e.hldAgainstTheirBidding(p, s)
		fr, en := callSym(psc.Call)
		tr.note("le partenaire est intervenu ("+fr+") : soutenir, nommer sa couleur, ou passer",
			"partner overcalled ("+en+"): raise, name a suit, or pass")
		// advance() only runs while p has made no bid, so any earlier call
		// of his is a pass -- and that pass has already limited the hand.
		_, alreadyPassed := e.lastCallBy(p.seat)
		// Rencontre: jump in a new suit, 4+ support for partner's
		// intervention, a good 5+ card suit, 8-11 HCP (docs/addon_6.md,
		// section IV).
		if sup >= 4 {
			for _, cand := range []Suit{Clubs, Diamonds, Hearts, Spades} {
				if cand == s {
					continue
				}
				if e.openCall.Strain <= SSpades && cand == Suit(e.openCall.Strain) {
					continue // the opponents' own suit
				}
				if c, mn, ok := e.tryRencontre(p, s, cand, 4, 1); ok {
					tr.check(true, "4 atouts et belle cinquième à "+suitSymbol[cand]+", 8-11 H → enchère de rencontre",
						"four trumps and a good five cards in "+suitSymbol[cand]+", 8-11 H → meeting bid", cards(h, cand))
					return c, mn
				}
			}
		}
		// The fit, measured rather than assumed. [A-5] asks for "3 atouts et
		// plus", a count taken against the five cards a one-level overcall
		// promises; at the two and three levels the overcall promises six
		// ([I-5]), where a doubleton makes the very same eight-card fit.
		fitLen := sup + partner.shownLens[s]
		// Too strong for the capped raises below: ask which end of the overcall's
		// range partner holds [A-6]. The fit is what makes the question worth
		// asking -- and what makes the answer safe, since partner may repeat
		// his suit.
		//
		// The two raise bands further down keep asking for three real trumps:
		// they are competitive bids whose safety is the trump length itself,
		// and on a doubleton they would buy a level the law of total tricks
		// does not pay for. This band is not a raise but a statement of
		// values -- opposite the six cards and 12 HL a three-level overcall
		// promises, the side owns two openings whatever the trump split -- so
		// the eight-card fit is enough to move.
		// Two ways in, and they measure different things. 13 HLD is playing
		// strength: the trumps and the shortness will bring the tricks in.
		// 11 H is the honour count the cue-bid itself announces -- "j'ai un
		// fit avec toi et une main forte" -- and a hand that holds it owns
		// half the deck's honours opposite a bid that may hold anything from
		// 9 to 18 HL. Guessing which end of that range partner has is not a
		// decision the advance should be making on its own, and the jump
		// raise makes it: it names the game zone and caps itself there.
		//
		// The H count is taken in useful points, exactly as the HLD one is
		// [A-5]: a bare king or queen in the suit they opened is not worth
		// its pips, and without that discount the very hand the "points
		// utiles" rule was written for -- K74 AJ32 Q J8765, eleven printed
		// honour points -- would walk straight back into the cue-bid.
		usefulH := e.hAgainstTheirBidding(p, s)
		if tr.check(fitLen >= 8 && (hld >= 13 || usefulH >= 11),
			"fit de 8 cartes et main forte (13 HLD ou 11 H utiles) → cue-bid de force [A-6]",
			"eight-card fit and a strong hand (13 HLD or 11 useful H) → strength cue-bid [A-6]",
			fmt.Sprintf("%d cartes, %s, %d H", fitLen, pts(hld, "HLD"), usefulH)) {
			floor := 13
			if hld < 13 {
				floor = 11 // announce what the hand holds, not the other band's
			}
			if c, mn, ok := e.overcallStrengthAsk(p, floor); ok {
				return c, mn.withLen(s, sup)
			}
			return e.untraced(e.conclude(p))
		}
		if tr.check(sup >= 3, "3 atouts et plus → soutien de l'intervention [A-5]",
			"three or more trumps → raise the overcall [A-5]", cards(h, s)) {
			tr.in()
			switch {
			case tr.check(hld >= 11, "11-12 HLD → soutien à saut (au palier le plus bas si déjà passé)",
				"11-12 HLD → jump raise (cheapest level if already passed)", pts(hld, "HLD")):
				tr.out()
				c := e.cheapestCall(s.Strain())
				// The jump is an invitation, and an invitation is a promise
				// of values. A hand that has already passed -- over the
				// opening, or as dealer -- has denied them: whatever the
				// fit adds afterwards, it supports at the cheapest level and
				// leaves partner free to pass. Jumping there asks partner to
				// bid a game on points the pass said were not there.
				if c.Level+1 <= 3 && !alreadyPassed {
					return bid(c.Level+1, s.Strain()), m(11, 12, "soutien à saut de l'intervention", "jump raise of the overcall").withLen(s, sup).asInvite()
				}
				return c, m(11, 12, "soutien de l'intervention", "raise of the overcall").withLen(s, sup)
			case tr.check(hld >= 7, "7-10 HLD → soutien simple", "7-10 HLD → single raise", pts(hld, "HLD")):
				c := e.cheapestCall(s.Strain())
				if c.Level <= 3 {
					tr.out()
					return c, m(7, 10, "soutien simple de l'intervention", "single raise of the overcall").withLen(s, sup)
				}
			}
			tr.out()
		}
		own, ownOK := e.advanceOwnSuit(p, s)
		ownVal := ""
		if ownOK {
			ownVal = cards(h, own) + ", "
		}
		if tr.check(ownOK && hl >= 10, "une couleur à soi et 10 HL → la nommer, non forcing",
			"a suit of one's own and 10 HL → name it, not forcing", ownVal+pts(hl, "HL")) {
			c := e.cheapestCall(own.Strain())
			// The level has to be paid for in cards. A fifth card names a
			// partscore partner is free to pass, and stops at the two level;
			// the three level asks him to play a contract he did not choose
			// and that his own suit has to be abandoned for, so it takes a
			// sixth card. Above that the hand is worth a real bid, not this
			// limited one.
			switch {
			case c.Level <= 2:
				return c, m(10, 17, "couleur propre, non forcing", "own suit, not forcing").withLen(own, 5)
			case c.Level == 3 && h.Len(own) >= 6:
				return c, m(10, 17, "couleur propre sixième au palier de 3, non forcing", "own six-card suit at the three level, not forcing").withLen(own, 6)
			}
		}
		// A stopper in every suit the opponents have named is required both for
		// the natural notrump advance and for the strong cue-bid below: the
		// responder may have introduced (and the opener raised) a second suit
		// that would be run against a notrump contract.
		if oppSuits := e.opponentSuits(p); len(oppSuits) > 0 {
			stopped := true
			for _, os := range oppSuits {
				if !h.Stopper(os) {
					stopped = false
					break
				}
			}
			// Too strong for the limited 1SA advance (capped at 10HL) but with
			// the same stopped, no-fit, no-biddable-suit profile: cue-bid the
			// opener's suit as a forcing probe that asks partner to describe
			// further (11HL and up).
			//
			// This one aims at 3NT rather than at partner's suit, so it keeps
			// the old three-level cap instead of the game ceiling the ask uses
			// with a fit: a cue-bid at the four level has already climbed past
			// the contract it was probing for, and the coded return would land
			// the side in a suit nobody here has support for.
			if e.openCall.Strain <= SSpades && tr.check(stopped && hl >= 11,
				"leurs couleurs arrêtées, 11 HL, sans fit ni couleur → cue-bid forcing",
				"their suits stopped, 11 HL, no fit and no suit → forcing cue-bid", pts(hl, "HL")) {
				if c, mn, ok := e.overcallStrengthAsk(p, 11); ok && c.Level <= 3 {
					mn.fr = "cue-bid, main forte sans fit ni couleur propre, forcing"
					mn.en = "cue-bid, strong hand without a fit or biddable suit, forcing"
					return c, mn
				}
			}
			// Natural notrump advance: regular or semi-regular values with a
			// stopper in the opponents' suit(s), 8-10HL (docs/bidings.md).
			if tr.check(stopped && hl >= 8 && hl <= 10, "leurs couleurs arrêtées, 8-10 HL → Sans-Atout naturel",
				"their suits stopped, 8-10 HL → natural notrump", pts(hl, "HL")) {
				c := e.cheapestCall(SNoTrump)
				if c.Level <= 2 {
					return c, m(8, 10, "Sans-Atout naturel avec arrêt", "natural notrump with a stopper")
				}
			}
		}
	}
	tr.note("aucune réponse ne s'applique → Passe", "no advance applies → Pass")
	return passCall, m(0, 7, "", "")
}

// ---------- answering the balancing seat (réponses au réveil) ----------

// advanceReopenSuit answers a suit réveil (docs/regles_moteur.md §9.5). The
// balancing bid has already denied an opening and capped itself at 13 HL, so
// this is not the advance made over a direct overcall: there is no strength
// left to ask about, the notrump answers carry their own — higher — zones
// because partner's may be as little as 8 HL, and the cue-bid of the opener's
// suit is a game try rather than a question.
func (e *Engine) advanceReopenSuit(p *playerState, s Suit) (Call, meaning) {
	h := p.hand
	hp, hl := h.H(), h.HL()
	sup, hld := h.Len(s), h.HLD(s)
	opp, hasOpp := Suit(e.openCall.Strain), e.openCall.Strain <= SSpades

	// cue bids the opener's suit: "espoir de manche, l'ouverture et plus".
	// Unlike the cue-bid of [A-6] it asks nothing about partner's strength —
	// the réveil has already announced it — it announces our own.
	cue := func(min int, fr, en string) (Call, meaning, bool) {
		if !hasOpp {
			return Call{}, meaning{}, false
		}
		c := e.cheapestCall(opp.Strain())
		if c.Level > 3 || !e.legal(p.seat, c) {
			return Call{}, meaning{}, false
		}
		mn := m(min, 40, fr, en).asForcing()
		mn.cuebid, mn.reopenAsk = true, true
		return c, mn, true
	}

	// Rencontre: jump in a new suit with four-card support, as over a direct
	// overcall [A-5].
	if sup >= 4 {
		for _, cand := range []Suit{Clubs, Diamonds, Hearts, Spades} {
			if cand == s || (hasOpp && cand == opp) {
				continue
			}
			if c, mn, ok := e.tryRencontre(p, s, cand, 4, 1); ok {
				return c, mn
			}
		}
	}
	if sup >= 3 {
		switch {
		case hld >= 13:
			if c, mn, ok := cue(13, "cue-bid sur le réveil : espoir de manche avec le fit, l'ouverture et plus, forcing", "cue-bid over the reopening: game hope with a fit, opening values and up, forcing"); ok {
				return c, mn.withLen(s, sup)
			}
		case hld >= 11:
			c := e.cheapestCall(s.Strain())
			// Unlike [A-5], the jump raise survives the advancer's earlier
			// pass: in the balancing seat that pass is not a denial of
			// anything, it is the very reason partner had to reopen, and the
			// réveil it answers is capped at 13 HL.
			if c.Level+1 <= 3 {
				return bid(c.Level+1, s.Strain()), m(11, 12, "soutien à saut du réveil", "jump raise of the reopening").withLen(s, sup).asInvite()
			}
			if c.Level <= 3 {
				return c, m(11, 12, "soutien du réveil", "raise of the reopening").withLen(s, sup)
			}
		case hld >= 7:
			if c := e.cheapestCall(s.Strain()); c.Level <= 3 {
				return c, m(7, 10, "soutien simple du réveil", "single raise of the reopening").withLen(s, sup)
			}
		}
	}
	// Natural notrump answers on their own zones: 9-12 H for 1SA, 13-15 for
	// 2SA, both asking for a stopper or three small cards in the opened suit.
	if hasOpp && h.reopenHold(opp) {
		withHold := func(mn meaning) meaning {
			if h.Stopper(opp) {
				return mn.withStopper(opp)
			}
			return mn
		}
		cheap := e.cheapestCall(SNoTrump)
		// 1SA carries the 9-12 zone, so 2SA is a jump over it -- and stays
		// 2SA when partner's réveil has already used up the one level.
		strong := cheap
		if strong.Level == 1 {
			strong = bid(2, SNoTrump)
		}
		switch {
		case hp >= 13 && hp <= 15 && strong.Level == 2 && e.legal(p.seat, strong):
			mn := withHold(m(13, 15, "2SA sur le réveil, 13-15H avec l'arrêt (ou trois petites cartes)", "2NT over the reopening, 13-15 with a stopper (or three small cards)"))
			mn.reopenNTInvite = true
			return strong, mn
		case hp >= 9 && hp <= 12 && cheap.Level == 1 && e.legal(p.seat, cheap):
			return cheap, withHold(m(9, 12, "1SA sur le réveil, 9-12H avec l'arrêt (ou trois petites cartes)", "1NT over the reopening, 9-12 with a stopper (or three small cards)"))
		}
	}
	// Opening values and nothing obvious to say: the cue-bid is the only game
	// try left, and it is forcing.
	if hl >= 12 {
		if c, mn, ok := cue(12, "cue-bid sur le réveil : espoir de manche, l'ouverture et plus, forcing", "cue-bid over the reopening: game hope, opening values and up, forcing"); ok {
			return c, mn
		}
	}
	// A new suit over a réveil is a misfit call, not forcing: partner is
	// limited and free to pass it.
	if own := h.Longest(); own != s && h.Len(own) >= 5 && h.GoodSuit(own) && hl >= 10 && !(hasOpp && own == opp) {
		if c := e.cheapestCall(own.Strain()); c.Level <= 2 && e.legal(p.seat, c) {
			return c, m(10, 17, "nouvelle couleur sur le réveil : misfit, non forcing", "new suit over the reopening: misfit, not forcing").withLen(own, 5)
		}
	}
	return passCall, m(0, 8, "", "")
}

// answerReopenDouble answers the balancing double (docs/regles_moteur.md
// §9.5). The double can be as little as 8 H, so the direct-seat zones of §9.2
// would overbid it on nearly every hand: one answers as though partner were
// weak, because the doubler who really holds an opening is the one who must
// speak again. The ladder is read from the cheapest call in the suit — no jump
// with 0-10 H, one jump for a four-card suit with 11-12 H or a five-card suit
// with 8-10 H, two jumps for five cards with 11-12 H — with 1SA (9-12 H) and
// 2SA (13-14 H) for the hands holding the opened suit and no four-card major,
// and the cue-bid above all of it for opening values with nothing obvious.
func (e *Engine) answerReopenDouble(p *playerState, oppSuit Suit, forced bool) (Call, meaning) {
	h := p.hand
	hp := h.H()
	hold := h.reopenHold(oppSuit)

	// The suit to name: longest outside theirs, and a four-card major ahead of
	// any minor — the double promises support there, and a major partscore is
	// worth more than a minor one.
	var best Suit
	bestLen := 0
	for _, s := range []Suit{Hearts, Spades, Diamonds, Clubs} {
		if s == oppSuit {
			continue
		}
		if n := h.Len(s); n > bestLen {
			best, bestLen = s, n
		}
	}
	for _, s := range []Suit{Hearts, Spades} {
		if s != oppSuit && h.Len(s) >= 4 && !best.IsMajor() {
			best, bestLen = s, h.Len(s)
			break
		}
	}
	hasMajor4 := best.IsMajor() && bestLen >= 4

	// 2SA, then the cue-bid: from 13 H the hand is past every rung of the
	// ladder, and the cue-bid is what keeps the auction alive when no natural
	// bid describes it.
	if hp >= 13 {
		if hold && !hasMajor4 && (h.IsRegular() || h.IsSemiRegular()) && hp <= 14 {
			// 1SA answers the 9-12 zone, so this one is a jump to 2SA.
			c := e.cheapestCall(SNoTrump)
			if c.Level == 1 {
				c = bid(2, SNoTrump)
			}
			if c.Level == 2 && e.legal(p.seat, c) {
				mn := m(13, 14, "2SA sur le contre de réveil, 13-14H et l'arrêt", "2NT over the reopening double, 13-14 with the stopper")
				if h.Stopper(oppSuit) {
					mn = mn.withStopper(oppSuit)
				}
				return c, mn
			}
		}
		c := e.cheapestCall(oppSuit.Strain())
		if c.Level <= 3 && e.legal(p.seat, c) {
			mn := m(13, 40, "cue-bid sur le contre de réveil : la valeur de l'ouverture et pas d'enchère évidente, forcing", "cue-bid over the reopening double: opening values and no obvious bid, forcing").asForcing()
			mn.cuebid = true
			return c, mn
		}
	}
	// The suit ladder.
	base := e.cheapestCall(best.Strain())
	jumps, lo, hi := 0, 0, 10
	switch {
	case bestLen >= 5 && hp >= 11:
		jumps, lo, hi = 2, 11, 12
	case bestLen >= 5 && hp >= 8:
		jumps, lo, hi = 1, 8, 10
	case bestLen >= 4 && hp >= 11:
		jumps, lo, hi = 1, 11, 12
	}
	// 1SA sits between the jumps and the minimum answer: a hand with a suit
	// worth a jump names the suit, a hand without one falls back on notrump
	// when it holds the opened suit, and only then comes the cheapest bid.
	if jumps == 0 && hp >= 9 && hp <= 12 && hold && !hasMajor4 {
		if c := e.cheapestCall(SNoTrump); c.Level == 1 && e.legal(p.seat, c) {
			mn := m(9, 12, "1SA sur le contre de réveil, 9-12H et l'arrêt (ou trois petites cartes)", "1NT over the reopening double, 9-12 with the stopper (or three small cards)")
			if h.Stopper(oppSuit) {
				mn = mn.withStopper(oppSuit)
			}
			return c, mn
		}
	}
	if jumps == 0 && !forced && hp < 8 {
		// Right-hand opponent has bid over the double: a hand with nothing to
		// add is free to stay silent rather than name a suit it does not own.
		return passCall, m(0, 7, "", "")
	}
	c := bid(min(base.Level+jumps, 3), best.Strain())
	if c.Level <= base.Level {
		// No room left for the jump the zone calls for (2♥ behind 1♠ is the
		// classic case): the bid falls back on the cheapest call, and covers
		// everything from the weak zone up to what the jump would have shown.
		c, lo = base, 0
	}
	if !e.legal(p.seat, c) {
		return passCall, m(0, 7, "", "")
	}
	fr, en := "réponse au contre de réveil au palier le plus bas, 0-10H", "minimum answer to the reopening double, 0-10"
	switch jumps {
	case 1:
		fr, en = "saut sur le contre de réveil", "jump answer to the reopening double"
	case 2:
		fr, en = "double saut sur le contre de réveil, 5 cartes et 11-12H", "double jump over the reopening double, five cards and 11-12"
	}
	return c, m(lo, hi, fr, en).withLen(best, min(bestLen, 5))
}

// advanceNotrumpLongSuit names a running six-card suit facing partner's 15-17
// notrump overcall [I-4].
//
// The generic endgame [A-4] values this hand on honours alone -- the rule for
// notrump, where "a long side suit may not run". A six-card suit headed by two
// of the top three honours is exactly the suit that does run, and opposite a
// balanced 15-17 it is worth the four or five tricks the honour count never
// sees. Passing there buries a game.
//
// The advancer names the suit rather than notrump because he is usually the
// one who cannot bid it: the opponents have bid, he has no stopper of his own,
// and it is the overcaller -- who guaranteed one, and holds the balanced hand
// behind it -- who can place the contract. So this bid describes and hands
// back the decision; it never concludes.
func (e *Engine) advanceNotrumpLongSuit(p *playerState) (Call, meaning, bool) {
	h := p.hand
	partner := e.ps[partnerOf(p.seat)]
	var oppNamed [4]bool
	for _, os := range e.opponentSuits(p) {
		oppNamed[os] = true
	}
	best, bestLen := Clubs, 0
	for s := Clubs; s <= Spades; s++ {
		if oppNamed[s] || h.Len(s) < 6 || !h.GoodSuit(s) {
			continue
		}
		if h.Len(s) > bestLen {
			best, bestLen = s, h.Len(s)
		}
	}
	if bestLen == 0 {
		return Call{}, meaning{}, false
	}
	// The suit's own length points are the whole argument for bidding it, so
	// they are what the count uses: HL, not the bare honours the notrump rule
	// would apply. Below the game zone opposite partner's floor the hand is
	// not worth pushing the auction one level higher than it already sits.
	if h.HL()+partner.shownMin < 25 {
		return Call{}, meaning{}, false
	}
	c := e.cheapestCall(best.Strain())
	if c.Level > 3 || !e.legal(p.seat, c) {
		return Call{}, meaning{}, false
	}
	return c, m(h.HL(), 40,
		"couleur longue et belle face à l'intervention à 1SA, forcing",
		"long running suit facing the 1NT overcall, forcing").
		withLen(best, bestLen).asForcing(), true
}

// overcallStrengthAsk cue-bids the opener's suit over partner's one-level suit
// overcall. The overcall promises 9-18 HL, far too wide for the advancer to
// place the contract on his own, so he asks which end of it partner holds.
// The answer is coded -- without opening values the overcaller returns to his
// own suit, with them he describes his hand [A-6].
//
// Asking takes a fit: at least three cards in the overcall suit. Without one,
// every coded answer -- the sign-off return to the suit included -- lands the
// side in a contract nobody has support for, and the question was never worth
// its price. The one exception is the notrump-oriented probe of [A-5], which
// carries a stopper in every enemy suit and is aiming at 3NT rather than at
// partner's suit; the call site there overrides the wording.
func (e *Engine) overcallStrengthAsk(p *playerState, minPts int) (Call, meaning, bool) {
	if e.openCall.Strain > SSpades {
		return Call{}, meaning{}, false
	}
	// The question is "have you got an opening hand?", and the coded answer
	// splits the 9-18 HL of a direct overcall in two. A balancing bid has
	// already answered it -- it denies the opening [V-6] -- so there is
	// nothing left to ask, and the sign-off reply would promise 9 points a
	// six-point hand does not hold.
	if e.ps[partnerOf(p.seat)].shownMax <= 12 {
		return Call{}, meaning{}, false
	}
	cue := e.cheapestCall(Suit(e.openCall.Strain).Strain())
	if !e.legal(p.seat, cue) {
		return Call{}, meaning{}, false
	}
	// The question costs whatever its worst answer costs: the coded sign-off,
	// partner's cheapest return to his own suit. Below the game that return is
	// a partscore he is free to pass, so the question is free; the game itself
	// is still affordable, being where a fit and 13 HLD were heading anyway.
	// Past the game the answer would buy a level nobody bid, and the question
	// stops being worth its price.
	//
	// A blanket "never above the three level" stood here before, and it refused
	// the ask over a three-level overcall -- exactly the auction where partner
	// has promised six cards and 12 HL, and where his return is the game and
	// nothing more.
	if psc, ok := e.lastCallBy(partnerOf(p.seat)); ok && psc.Call.IsBid() && psc.Call.Strain <= SSpades {
		suit := Suit(psc.Call.Strain)
		signoff := bid(cue.Level, suit.Strain())
		if !signoff.higherThan(cue) {
			signoff = bid(cue.Level+1, suit.Strain())
		}
		if signoff.higherThan(gameOfTrump(suit)) {
			return Call{}, meaning{}, false
		}
	} else if cue.Level > 3 {
		return Call{}, meaning{}, false
	}
	mn := m(minPts, 40, "cue-bid avec le fit, demande le niveau de l'intervention, forcing", "cue-bid with a fit, asking how strong the overcall is, forcing").asForcing()
	mn.cuebid = true
	mn.overcallAsk = true
	return cue, mn, true
}

// answerDouble applies the rule of the three zones to the takeout double
// (docs/addon_1.md): 0-7H minimum suit answer, 8-10H jump answers, and from
// 11H game-forcing developments through the cue-bid.
func (e *Engine) answerDouble(p *playerState, oppSuit Suit, forced bool) (Call, meaning) {
	h := p.hand
	hp := h.H()

	// Longest four-card or longer major, hearts preferred on a tie (the
	// cheaper answer, as over an opening bid).
	var maj Suit
	majLen := 0
	for _, s := range []Suit{Hearts, Spades} {
		if s != oppSuit && h.Len(s) >= 4 && h.Len(s) > majLen {
			maj, majLen = s, h.Len(s)
		}
	}
	bothMajors := !oppSuit.IsMajor() && h.Len(Hearts) >= 4 && h.Len(Spades) >= 4
	stopper := h.Stopper(oppSuit) && (h.IsRegular() || h.IsSemiRegular())
	tr := e.tr
	majVal := cards(h, Hearts) + ", " + cards(h, Spades)

	// A seven-card (or longer) major applies the Law of Total Tricks: the
	// takeout double promises at least four cards in every unbid major, so a
	// seventh trump assures an eleven-card fit. The number of trumps and the
	// distribution — not the honour count — command the bid, so game is bid
	// even on a near-bust hand.
	if tr.check(majLen >= 7, "majeure septième → manche (loi des levées totales)",
		"seven-card major → game (law of total tricks)", majVal) {
		c := bidSuit(4, maj)
		if e.legal(p.seat, c) {
			return c, m(0, 40, "saut à la manche, majeure septième : loi des levées totales, le contre garantit 3 atouts, soit un fit dixième", "jump to game, seven-card major: law of total tricks, the double guarantees three trumps, a ten-card fit").withLen(maj, 7)
		}
	}
	// A six-card major reaches game from the middle zone: naming a major game
	// facing the double takes six cards there, five in the strong zone.
	if tr.check(majLen >= 6 && hp >= 8, "majeure sixième et 8 H → manche", "six-card major and 8 H → game",
		majVal+", "+pts(hp, "H")) {
		return bidSuit(4, maj), m(8, 40, "saut à la manche, majeure sixième, zone moyenne", "jump to game, six-card major, middle zone").withLen(maj, 6)
	}
	// 11+H: game-forcing zone.
	if tr.check(hp >= 11, "11 H et plus → zone forte", "11+ H → strong zone", pts(hp, "H")) {
		tr.in()
		if tr.check(majLen >= 5, "majeure cinquième → manche", "five-card major → game", majVal) {
			tr.out()
			return bidSuit(4, maj), m(11, 40, "saut à la manche, majeure cinquième, 11H et plus", "jump to game, five-card major, 11+").withLen(maj, 5)
		}
		// No four-card major: look for notrumps. Every notrump answer is
		// positive and its zone is precise, so each level has its own price
		// in stoppers -- two of them for 3NT, a stopper and a half for 2NT.
		// The strong zone stops at 14: above it the hand is worth more than
		// any of these limited answers and goes through the cue-bid.
		if tr.check(majLen == 0 && (h.IsRegular() || h.IsSemiRegular()) && hp <= 14,
			"sans majeure quatrième, régulière, 11-14 H → Sans-Atout envisagé",
			"no four-card major, balanced, 11-14 H → notrump considered", shape(h)) {
			if tr.check(hp >= 12 && h.DoubleStopper(oppSuit), "12-14 H et deux arrêts → 3SA",
				"12-14 H and two stoppers → 3NT", pts(hp, "H")) {
				c := bid(3, SNoTrump)
				if e.legal(p.seat, c) {
					tr.out()
					return c, m(12, 14, "3SA, 12-14H, deux arrêts", "3NT, 12-14, two stoppers")
				}
			}
			if tr.check(hp <= 12 && h.StopperAndHalf(oppSuit), "11-12 H et un arrêt et demi → 2SA",
				"11-12 H and a stopper and a half → 2NT", pts(hp, "H")) {
				c := bid(2, SNoTrump)
				if e.legal(p.seat, c) {
					tr.out()
					return c, m(11, 12, "2SA, 11H, arrêt et demi", "2NT, 11, a stopper and a half").asInvite()
				}
			}
		}
		tr.check(true, "sinon → cue-bid, forcing", "otherwise → cue-bid, forcing", "")
		tr.out()
		return e.cueBid(p, oppSuit, 11)
	}
	// 8-10H: positive answers, all with a jump.
	if tr.check(hp >= 8, "8-10 H → réponse positive, avec saut", "8-10 H → positive answer, with a jump", pts(hp, "H")) {
		tr.in()
		if tr.check(bothMajors, "les deux majeures quatrièmes → cue-bid", "both four-card majors → cue-bid", majVal) {
			// Both four-card majors over a minor opening: the cue-bid finds
			// the 4-4 fit without guessing.
			return e.cueBid(p, oppSuit, 8)
		}
		if tr.check(majLen >= 4, "majeure quatrième → la nommer avec saut", "four-card major → name it with a jump", majVal) {
			// The ladder is anchored on game, not on the cheapest bid: four
			// cards land at the two level, five at the three level, six at
			// game. Behind 1C, 1D or 1H that reads as the simple, double and
			// triple jump the middle zone promises; behind 1S one step is
			// missing, so 2H is no jump at all and carries the whole 0-10
			// zone, 3H showing five hearts and 4H six.
			n := min(majLen, 6)
			c := bid(n-2, maj.Strain())
			base := e.cheapestCall(maj.Strain())
			if !c.higherThan(base) && c != base {
				c = base
			}
			fr, en := "réponse à saut au contre, 4 cartes, 8-10H", "jump answer to the double, four cards, 8-10"
			lo := 8
			switch {
			case n == 5:
				fr, en = "saut, 5 cartes, 8-10H", "jump, five cards, 8-10"
				if c.Level-base.Level >= 2 {
					fr, en = "double saut, 5 cartes, 8-10H", "double jump, five cards, 8-10"
				}
			case c == base:
				// No jump available: the bid spans the weak and middle zones.
				fr, en = "réponse au contre, 4 cartes, 0-10H", "answer to the double, four cards, 0-10"
				lo = 0
			}
			mn := m(lo, 10, fr, en).withLen(maj, n)
			if lo == 0 {
				mn.wideDouble, mn.wideDoubleSuit = true, maj
			}
			return c, mn
		}
		if tr.check(stopper, "arrêt dans leur couleur, main régulière → 1SA",
			"stopper in their suit, balanced → 1NT", cards(h, oppSuit)) {
			c := e.cheapestCall(SNoTrump)
			if c.Level == 1 {
				return c, m(8, 10, "1SA, 8-10H, dénie une majeure quatrième", "1NT, 8-10, denies a four-card major")
			}
		}
		// Jump in a five-card or longer minor, not forcing.
		var mi Suit
		miLen := 0
		for _, s := range []Suit{Diamonds, Clubs} {
			if s != oppSuit && h.Len(s) >= 5 && h.Len(s) > miLen {
				mi, miLen = s, h.Len(s)
			}
		}
		miVal := cards(h, Diamonds) + ", " + cards(h, Clubs)
		if tr.check(miLen >= 5, "mineure cinquième → saut, non forcing", "five-card minor → jump, not forcing", miVal) {
			c := e.cheapestCall(mi.Strain())
			if c.Level+1 <= 3 {
				return bid(c.Level+1, mi.Strain()), m(8, 10, "saut en mineure, 5 cartes et plus, non forcing", "minor-suit jump, five-plus cards, not forcing").withLen(mi, 5)
			}
		}
		if c := e.cheapestCall(SNoTrump); c.Level == 1 {
			tr.note("sinon → 1SA, le moins mauvais mensonge", "otherwise → 1NT, the least bad lie")
			return c, m(8, 10, "1SA, le moins mauvais mensonge", "1NT, the least available lie")
		}
		tr.out()
	}
	// 0-7H: a suit without a jump, four-card major before five-card minor.
	// This is the purely obligatory answer; if the opponents have bid over the
	// double the advancer is free and a hand this weak simply passes.
	tr.check(true, "0-7 H → réponse la moins chère", "0-7 H → cheapest answer", pts(hp, "H"))
	if tr.check(!forced, "l'adversaire a parlé après le contre : la réponse n'est plus obligatoire → Passe",
		"the opponents bid over the double: the answer is no longer forced → Pass", "") {
		return passCall, m(0, 7, "", "")
	}
	best := maj
	if majLen == 0 {
		bestLen := 0
		for _, s := range []Suit{Spades, Hearts, Diamonds, Clubs} {
			if s == oppSuit {
				continue
			}
			if h.Len(s) > bestLen {
				best, bestLen = s, h.Len(s)
			}
		}
	}
	c := e.cheapestCall(best.Strain())
	n := 3
	if majLen >= 4 {
		n = 4
	}
	// Over a double of an artificial opening (e.g. a strong 2C) the "enemy
	// suit" exclusion can leave only sub-minimum holdings: record what the
	// hand really has rather than the theoretical promise.
	if h.Len(best) < n {
		n = h.Len(best)
	}
	tr.note("majeure quatrième d'abord, sinon la couleur la plus longue", "a four-card major first, otherwise the longest suit")
	return c, m(0, 7, "réponse au contre au palier le plus bas, 0-7H", "minimum answer to the takeout double, 0-7").withLen(best, n)
}

// cueBid bids the opponents' suit, asking the doubler to describe his hand,
// starting with his cheapest four-card major.
func (e *Engine) cueBid(p *playerState, oppSuit Suit, minPts int) (Call, meaning) {
	c := e.cheapestCall(oppSuit.Strain())
	if c.Level <= 3 && e.legal(p.seat, c) {
		mn := m(minPts, 40, "cue-bid, demande la majeure quatrième la moins chère, forcing", "cue-bid, asking for the cheapest four-card major, forcing").asForcing()
		mn.cuebid = true
		return c, mn
	}
	// No room left for the cue-bid: fall back on the generic engine. Going
	// through the value-based endgame directly (rather than conclude, which
	// would still see partner's double as its own last call and route right
	// back into answerDouble/cueBid) avoids looping forever between the two.
	return e.concludeGameDecision(e.concludeContext(p))
}
