package main

import (
	"strings"
	"testing"
)

// TestAnswerOpenersReopeningDouble checks that a partner's forcing reopening
// double is not left to die a penalty pass. North opens 1S, East and South
// pass, West balances with a "réveil" 2C (6-12H, denies an opening), and
// North -- 18H, "toutes distributions" -- reopens with a strong forcing
// double (docs/addon_1.md, "contre de réveil, jeu fort"). With East silent
// over the double, South (a bust, 0H) must still describe its hand rather
// than pass and convert the double into a penalty pass at 2C: conclude had
// no notion of a double from partner at all, since that only ever happens on
// the opener's side via this reopening double or the (non-forcing) penalty
// double of a sacrifice, and fell back to a generic value-based endgame that
// simply passed for lack of a known fit.
func TestAnswerOpenersReopeningDouble(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("A9873", "AKJ6", "QJ", "K8"), // 18H: opens 1S, later doubles 2C
		west:  hand("K", "Q5", "8652", "AQJT65"), // 12H: balances with 2C
		south: hand("", "9873", "9743", "97432"), // 0H, four hearts: forced to answer
	})
	e := NewEngine(d)

	c, mn := e.decide(north)
	if got := c.Format("fr"); got != "1P" {
		t.Fatalf("North's opening = %s (%s), want 1P", got, mn.fr)
	}
	e.record(north, c, mn)

	e.record(east, passCall, noInfo())

	c, mn = e.decide(south)
	if c.Kind != KindPass {
		t.Fatalf("South's first call = %s (%s), want Passe", c.Format("fr"), mn.fr)
	}
	e.record(south, c, mn)

	c, mn = e.decide(west)
	if got := c.Format("fr"); got != "2T" || !strings.Contains(mn.fr, "réveil") {
		t.Fatalf("West's reopening = %s (%s), want 2T réveil", got, mn.fr)
	}
	e.record(west, c, mn)

	c, mn = e.decide(north)
	if c.Kind != KindDouble || !mn.forcing {
		t.Fatalf("North's reopening call = %s (%s, forcing=%v), want a forcing Contre", c.Format("fr"), mn.fr, mn.forcing)
	}
	e.record(north, c, mn)

	e.record(east, passCall, noInfo())

	c, mn = e.decide(south)
	if got := c.Format("fr"); got != "2C" {
		t.Fatalf("South's answer to the double = %s (%s), want 2C (four hearts, forced minimum answer)\ngot Passe instead: the forcing double was read as a penalty pass", got, mn.fr)
	}
	if !strings.Contains(mn.fr, "réponse au contre") {
		t.Fatalf("comment %q does not describe an answer to the double", mn.fr)
	}
}
