package main

import (
	"strings"
	"testing"
)

// TestAceStepResponderCorrectsGameSignoff pins [S-10b]. Sud ouvre de 2K avec
// 24 H pile, Nord répond 2SA (pas d'As, 8 H et plus), Sud conclut à 3SA. Le
// plancher de l'échelle des As vaut 8, donc le capitaine compte 32 et s'arrête
// — alors que Nord en a 11, et que le camp en totalise 35. La description de
// l'ouvreur étant close, c'est à Nord de corriger.
func TestAceStepResponderCorrectsGameSignoff(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "N:J8.54.KQ84.KQ754 52.AJ873.T9632.9 AKQ.KQT.AJ5.AJ86 T97643.962.7.T32"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatal(err)
	}
	calls := NewEngine(d).Run()
	if got, comment := nthCallText(t, calls, 0, 1); got != "6SA" { // deuxième enchère de Nord
		t.Fatalf("enchère de Nord = %s (%s), attendu 6SA ; enchères : %s", got, comment, formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract.Format("fr") != "6SA" {
		t.Fatalf("contrat final = %s, attendu 6SA ; enchères : %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestAceStepResponderWaitsWhileOpenerDescribes pins la réserve de [S-10b] :
// tant que l'ouvreur nomme des couleurs sous la manche, il décrit encore, et
// la main qui a répondu aux As ne prend pas la main. Nord tient 12 H — de quoi
// atteindre 33 face au plancher de 24 — mais il pose 3SA. Sud, qui lit alors
// ce compte, appelle les Rois et trouve le grand chelem que le 6SA aurait
// enterré.
func TestAceStepResponderWaitsWhileOpenerDescribes(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "N:K54.Q764.T9.AK54 J9762.T83.7.QT73 A.AKJ95.AKQJ65.J QT83.2.8432.9862"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatal(err)
	}
	calls := NewEngine(d).Run()
	if got, comment := nthCallText(t, calls, 0, 1); got != "3SA" {
		t.Fatalf("enchère de Nord = %s (%s), attendu 3SA ; enchères : %s", got, comment, formatAuction(calls))
	}
	if got, comment := nthCallText(t, calls, 2, 2); got != "4SA" || !strings.Contains(comment, "Rois") {
		t.Fatalf("enchère de Sud = %s (%s), attendu 4SA appel aux Rois ; enchères : %s", got, comment, formatAuction(calls))
	}
	if contract, _, _ := finalContract(calls); contract.Format("fr") != "7SA" {
		t.Fatalf("contrat final = %s, attendu 7SA ; enchères : %s", contract.Format("fr"), formatAuction(calls))
	}
}
