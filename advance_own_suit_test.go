package main

import "testing"

// TestAdvanceNamesSixthSuitAtThreeLevel: the advance may name its own suit
// above the two level when a sixth card pays for it (audit du par, donne
// 679). South opens 1S, West overcalls 2H, and East holds DAQJ8732 with 13 HL
// -- the cheapest diamond call is 3D, and the two-level ceiling left him with
// nothing to say at all. His side owns nine diamonds and the par is a diamond
// slam; staying silent on a seven-card suit is not a judgement, it is a hole.
func TestAdvanceNamesSixthSuitAtThreeLevel(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "All"]
[Deal "S:AK984.KT6.96.K76 QJT2.AQJ54.K5.A4 653.9873.T4.T953 7.2.AQJ8732.QJ82"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const east = 1
	for _, sc := range calls {
		if sc.Seat != east || !sc.Call.IsBid() {
			continue
		}
		if got := sc.Call.Format("fr"); got != "3K" {
			t.Fatalf("East advances with %s (%s), want 3K on the seven-card suit\nauction: %s",
				got, sc.M.fr, formatAuction(calls))
		}
		return
	}
	t.Fatalf("East never bid his seven diamonds\nauction: %s", formatAuction(calls))
}

// TestAdvanceNeverNamesTheirSuitNaturally: the advance's own suit must be one
// the opponents have not named (audit du par, donne 144). West opens 1D,
// North overcalls 1S, and South holds DKQJT2 -- his longest, and theirs. Bid
// naturally it read as a partscore proposal partner was free to pass, in the
// one suit the side can never own; the same call is the cue-bid, and partner
// answers it.
func TestAdvanceNeverNamesTheirSuitNaturally(t *testing.T) {
	pbn := `[Dealer "W"]
[Vulnerable "EW"]
[Deal "W:9872.AK8.A9875.K KQJ64.T764..AQJ7 53.QJ932.643.986 AT.5.KQJT2.T5432"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	for _, sc := range calls {
		if sc.Seat != south || !sc.Call.IsBid() || sc.Call.Strain != SDiamonds {
			continue
		}
		if !sc.M.cuebid {
			t.Fatalf("South bids %s naturally (%s): diamonds are West's suit\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		return
	}
}
