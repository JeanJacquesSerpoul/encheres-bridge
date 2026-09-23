package main

import (
	"strings"
	"testing"
)

func mustParsePBN(t *testing.T, pbn string) *Deal {
	t.Helper()
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("ParsePBN: %v", err)
	}
	return d
}

// TestRubensohlMinorTexasCompletionIsNotSuperAccept: after a Rubensohl
// transfer to a minor (here 2SA for clubs over 1SA-(2H)), the opener's 3C
// completion is mandatory -- there is no room for a super-accept jump.
// Responder must not misread it as one and push a partscore to 4C while
// calling it game: with a limited hand the auction stops at three of the
// minor.
//
// North is single-suited in hearts on purpose: a five-five in the majors
// there would produce the Landy 2C intervention [I-3b] instead, and there
// would be no natural 2H overcall left to Rubensohl over.
func TestRubensohlMinorTexasCompletionIsNotSuperAccept(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "S"]
[Deal "S:T95.JT6.9852.Q32 AK2.95.KQJ7.K974 Q73.AK87432.AT.5 J864.Q.643.AJT86"]`)
	calls := NewEngine(d).Run()
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "rectification à saut") {
			t.Fatalf("the mandatory 3C completion was read as a super-accept\nauction: %s", formatAuction(calls))
		}
	}
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SClubs) {
		t.Fatalf("contract = %s, want 3T (sign-off on the completed minor transfer)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestNoChangeOfSuitOnDoubleton: a responder with six-card support and both
// minors short (T93 KQT953 76 A4 facing 1H) has no genuine new suit to force
// with. It must not fall back on a doubleton -- which opener would raise with
// four trumps into a 4-2 "fit" -- but value the known fit directly.
func TestNoChangeOfSuitOnDoubleton(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "W"]
[Deal "W:K6542.J.Q82.J876 AQ87.A8762.AK93. J.4.JT54.KQT9532 T93.KQT953.76.A4"]`)
	calls := NewEngine(d).Run()
	contract, declarer, _ := finalContract(calls)
	if !contract.IsBid() || contract.Strain != SHearts {
		t.Fatalf("contract = %s, want a heart contract on the 11-card fit\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	combined := d.Hands[declarer].Len(Hearts) + d.Hands[partnerOf(declarer)].Len(Hearts)
	if combined < 7 {
		t.Fatalf("contract rests on a %d-card combined fit\nauction: %s", combined, formatAuction(calls))
	}
}

// TestNoForcedMinorOnShortSuit: over a 1D opening, a strong balanced-ish
// responder without a four-card side suit (AK8 A97 JT753 A5, 16H) must not
// force with 2C on a doubleton -- which opener raises with four clubs into a
// 4-2 game -- but bid the natural 3SA behind its stoppers.
func TestNoForcedMinorOnShortSuit(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "N"]
[Deal "N:542.K.AKQ86.QT63 JT3.JT542.92.K87 AK8.A97.JT753.A5 Q976.Q863.4.J942"]`)
	calls := NewEngine(d).Run()
	const south = 2
	got, comment := southsCall(calls, south, 0)
	if got != "3SA" {
		t.Fatalf("South's response = %s (%s), want 3SA\nauction: %s", got, comment, formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract != bid(3, SNoTrump) {
		t.Fatalf("contract = %s, want 3SA\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}
