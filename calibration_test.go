package main

import (
	"strings"
	"testing"
)

// TestStrongHandBidsGameOverPreempt: facing partner's 3D preempt (5-10, seven
// cards), West holds 17H with a stopper in every side suit. The raw combined
// count understates seven playing tricks: West must bid 3SA, not pass out a
// game (audit find: ~100 such missed games per 20k deals).
func TestStrongHandBidsGameOverPreempt(t *testing.T) {
	pbn := `[Dealer "N"]
[Deal "N:QJ6.97.KJ7.T8642 942.T2.AQT9542.7 AT753.A643.3.QJ9 K8.KQJ85.86.AK53"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if got := contract.Format("fr"); got != "3SA" {
		t.Fatalf("final contract = %s, want 3SA over the preempt\nauction: %s", got, formatAuction(calls))
	}
}

// TestKeycardSignoffNotBumpedIntoSlam: North asks for keycards holding two;
// South's 5H answer shows two more without the trump queen, so a keycard is
// missing. The old code, finding five of the trump (the sign-off spot)
// already taken by the answer, bumped the "stop" to six. The answer must be
// passed instead.
//
// The deal originally used here (W:A2.K87.J9865.K98 KQT3.J9.Q7.Q7542
// 9754.T652.T43.JT J86.AQ43.AK2.A63) no longer reaches the ask at all: North
// held nothing but a five-card club suit opposite an 18-19 notrump rebid, and
// [S-0] now sends that hand to 3NT instead of opening a fitless slam probe.
func TestKeycardSignoffNotBumpedIntoSlam(t *testing.T) {
	pbn := `[Dealer "S"]
[Deal "S:AQ954.973.AJ84.6 T732.AKT.762.T73 KJ.QJ8652.K9.AKQ 86.4.QT53.J98542"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract.Level >= 6 {
		t.Fatalf("final contract = %s: slam bid missing two keycards on ~25 points\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
	found := false
	for _, sc := range calls {
		if strings.Contains(sc.M.fr, "arrêt sur la réponse") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the ask to pass the 5-level answer\nauction: %s", formatAuction(calls))
	}
}
