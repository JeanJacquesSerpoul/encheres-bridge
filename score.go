package main

// Duplicate-bridge scoring, as needed by the competitive sacrifice logic:
// the value of the opponents' contract if it makes exactly, against the
// penalty our own doubled sacrifice would concede.

// contractScore returns the score of an undoubled contract made exactly:
// trick score plus the partscore (+50), game (+300 / +500 vulnerable) and
// slam premiums (+500/+750 small, +1000/+1500 grand). Matches the classic
// table: 2S = 110, 3NT non-vul = 400, 4S vulnerable = 620, 5C vulnerable
// = 600, 6H non-vul = 980...
func contractScore(c Call, vul bool) int {
	if !c.IsBid() {
		return 0
	}
	var tricks int
	switch {
	case c.Strain == SNoTrump:
		tricks = 40 + 30*(c.Level-1)
	case c.Strain >= SHearts:
		tricks = 30 * c.Level
	default:
		tricks = 20 * c.Level
	}
	score := tricks
	if tricks >= 100 {
		if vul {
			score += 500
		} else {
			score += 300
		}
	} else {
		score += 50
	}
	if c.Level >= 6 {
		if vul {
			score += 750
		} else {
			score += 500
		}
	}
	if c.Level >= 7 {
		if vul {
			score += 750
		} else {
			score += 500
		}
	}
	return score
}

// doubledPenalty returns the points conceded for going down the given number
// of tricks, doubled: 100/300/500 then 300 apiece non-vulnerable, 200 then
// 300 apiece vulnerable (100, 300, 500, 800, 1100... / 200, 500, 800...).
func doubledPenalty(down int, vul bool) int {
	if down <= 0 {
		return 0
	}
	if vul {
		return 200 + 300*(down-1)
	}
	switch down {
	case 1:
		return 100
	case 2:
		return 300
	default:
		return 500 + 300*(down-3)
	}
}

// competitiveSacrifice weighs bidding over the opponents' game in a contested
// auction (each side runs its own evaluation). Two options, tried in order:
//   - bid on to make ("surenchère") when the combined count affords the extra
//     level(s): the game threshold plus about three points per level above
//     game -- judged on our own strength alone, so it applies over their
//     genuine game and over their sacrifice alike;
//   - otherwise sacrifice in the side's nine-card-plus fit when the doubled
//     penalty costs less than what the opponents would score for their
//     contract, vulnerability of each side taken into account. The expected
//     tricks come from the law of total tricks (combined trumps), minus one:
//     the law counts both camps' tricks together and the sacrificing side is
//     the one without the honours. A fit only buys one level this way.
//
// The caller has already checked the auction shape (their game as the last
// bid); this is shared by the conclude handler and by the silent advancer's
// path, which never reaches conclude.
func (e *Engine) competitiveSacrifice(ctx *concludeCtx) (Call, meaning, bool) {
	p, partner, fit, last, cMin := ctx.p, ctx.partner, ctx.fit, ctx.last, ctx.cMin
	sac := e.cheapestCall(fit.Strain())
	if sac.Level > 5 || !e.legal(p.seat, sac) {
		return Call{}, meaning{}, false
	}
	extra := sac.Level - gameOfTrump(fit).Level
	if extra < 0 {
		extra = 0
	}
	if cMin >= gameThreshold(fit, true)+3*extra {
		if extra == 0 {
			// Bidding our own game over theirs on real values is the
			// generic game decision's job, with its usual meaning -- not a
			// sacrifice.
			return Call{}, meaning{}, false
		}
		mn := m(-1, -1, "surenchère en compétition, le contrat reste probable", "competitive raise beyond game, the contract remains likely").withLen(fit, p.hand.Len(fit))
		return sac, mn, true
	}
	// The sacrifice, in contrast, only pays against a contract the opponents
	// are favourites to make: their combined shown floor must reach the game
	// zone.
	oppMin := e.ps[(p.seat+1)%4].shownMin + e.ps[(p.seat+3)%4].shownMin
	if oppMin < 23 {
		return Call{}, meaning{}, false
	}
	// The law gives one total for the deal, and this fit has already spent
	// its share: the same trumps cannot buy a second level. Without this the
	// side keeps re-pricing the same nine cards one rung higher each round,
	// until the doubled penalty passes the contract it was buying out.
	if e.lawAlreadySpent(p.seat, fit) {
		return Call{}, meaning{}, false
	}
	trumps := p.hand.Len(fit) + partner.shownLens[fit]
	if trumps < 9 {
		return Call{}, meaning{}, false
	}
	down := sac.Level + 6 - trumps
	if down <= 0 {
		// The law credits us with as many tricks as the contract needs:
		// compete on the fit alone.
		mn := m(-1, -1, "soutien loi des levées totales, le fit couvre le palier", "law of total tricks raise, the fit covers the level").withLen(fit, p.hand.Len(fit)).asLawBid(fit)
		return sac, mn, true
	}
	// The law counts the tricks of both camps together, and the side that
	// sacrifices is the side without the honours: its share of that total is
	// the smaller one, while the raw trump count credits it with the whole of
	// its fit. So the sacrifice is priced one trick worse than the count says
	// -- it must still cost less than their contract when it goes down one
	// more than the law promised.
	priced := down + 1
	if priced > 3 {
		return Call{}, meaning{}, false
	}
	penalty := doubledPenalty(priced, e.vul[ctx.side])
	oppScore := contractScore(last, e.vul[1-ctx.side])
	if penalty >= oppScore {
		return Call{}, meaning{}, false
	}
	mn := m(-1, -1, "sacrifice : la chute contrée coûte moins que leur contrat (loi des levées totales)", "sacrifice: the doubled penalty costs less than their contract (law of total tricks)").withLen(fit, p.hand.Len(fit)).asLawBid(fit)
	return sac, mn, true
}

// lawAlreadySpent reports whether this side has already bought a level in fit
// on the law of total tricks alone.
func (e *Engine) lawAlreadySpent(seat int, fit Suit) bool {
	for _, sc := range e.calls {
		if sideOf(sc.Seat) == sideOf(seat) && sc.M.lawBid && sc.M.lawSuit == fit {
			return true
		}
	}
	return false
}
