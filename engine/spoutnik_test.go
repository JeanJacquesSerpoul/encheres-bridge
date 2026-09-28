package engine

import (
	"strings"
	"testing"
)

// TestFiveCardMajorBeforeMinorSupport replays a reported auction: 1C-(1S),
// South holds 7.QT942.AQJ.AQ64 -- five hearts and 15 H. The engine raised
// clubs ("soutien en compétition, 11HLD et plus") and the auction died in
// 2C, the nine-card heart fit never found. A five-card major comes before
// any support [RC-1b].
func TestFiveCardMajorBeforeMinorSupport(t *testing.T) {
	pbn := `[Dealer "W"]
[Vulnerable "EW"]
[Deal "W:98.J73.KT72.9732 Q542.AK65.54.KT5 AKJT63.8.9863.J8 7.QT942.AQJ.AQ64"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	for _, sc := range calls {
		if sc.Seat != south || !sc.Call.IsBid() {
			continue
		}
		if got := sc.Call.Format("fr"); got != "2C" || !strings.Contains(sc.M.fr, "forcing") {
			t.Fatalf("South's first bid = %s (%q), want 2C (2H, forcing)\nauction: %s",
				got, sc.M.fr, formatAuction(calls))
		}
		break
	}
	contract, _, _ := finalContract(calls)
	if contract.Level != 4 || contract.Strain != SHearts {
		t.Fatalf("final contract = %s, want 4C (4H)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestSpoutnikDoubleShowsFourCardMajor checks the negative double [RC-6] on
// the same auction with exactly four hearts: South doubles instead of bidding,
// and opener names the major fit.
func TestSpoutnikDoubleShowsFourCardMajor(t *testing.T) {
	pbn := `[Dealer "W"]
[Deal "W:98.J732.KT7.9732 Q542.AK65.54.KT5 AKJT63.8.9863.J8 7.QT94.AQJ2.AQ64"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north, south = 0, 2
	var sDouble, nAnswer *SeatCall
	for i, sc := range calls {
		if sc.Seat == south && sc.Call.Kind == KindDouble && sDouble == nil {
			sDouble = &calls[i]
		}
		if sc.Seat == north && sDouble != nil && sc.Call.IsBid() && nAnswer == nil {
			nAnswer = &calls[i]
		}
	}
	if sDouble == nil || !sDouble.M.spoutnik || !strings.Contains(sDouble.M.fr, "Spoutnik") {
		t.Fatalf("South never made the Spoutnik double\nauction: %s", formatAuction(calls))
	}
	if !sDouble.M.spoutnikSuits[Hearts] || sDouble.M.spoutnikSuits[Spades] {
		t.Fatalf("the double should promise hearts only, got %v", sDouble.M.spoutnikSuits)
	}
	if nAnswer == nil || nAnswer.Call.Format("fr") != "2C" || !strings.Contains(nAnswer.M.fr, "Spoutnik") {
		t.Fatalf("North did not answer the double with 2C (2H)\nauction: %s", formatAuction(calls))
	}
	contract, _, _ := finalContract(calls)
	if contract.Level != 4 || contract.Strain != SHearts {
		t.Fatalf("final contract = %s, want 4C (4H)\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestSpoutnikAnswerNeverPasses checks opener's side of [RC-6]: the double is
// a bid, so opener describes his hand even with no fit for the major.
func TestSpoutnikAnswerNeverPasses(t *testing.T) {
	e := &Engine{opener: 0, openCall: bid(1, SClubs)}
	e.ps[0] = &playerState{seat: 0, hand: hand("A72", "5", "KQ96", "KQ954"), shownMin: 12, shownMax: 23}
	e.ps[1] = &playerState{seat: 1, hand: &Hand{}, shownMax: 40}
	e.ps[2] = &playerState{seat: 2, hand: &Hand{}, shownMax: 40}
	e.ps[3] = &playerState{seat: 3, hand: &Hand{}, shownMax: 40}
	e.calls = []SeatCall{
		{Seat: 0, Call: bid(1, SClubs)},
		{Seat: 1, Call: bid(1, SSpades)},
	}
	var majors [4]bool
	majors[Hearts] = true
	c, mn := e.answerSpoutnik(e.ps[0], majors)
	if c.Kind == KindPass {
		t.Fatalf("opener passed the negative double (%q)", mn.fr)
	}
	if !strings.Contains(mn.fr, "pas le fit majeur") {
		t.Fatalf("call = %s (%q), want a descriptive bid without the heart fit", c.Format("fr"), mn.fr)
	}
}

// TestSpoutnikDoubleNeedsEightHL replays a reported auction: 1C-(1D), North
// holds T8.T643.KQT654.T -- four hearts, 5 H and 7 HL with the diamond
// length. The Spoutnik double needs 8 HL [RC-6]: one point short.
func TestSpoutnikDoubleNeedsEightHL(t *testing.T) {
	pbn := `[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:KQ92.A7.J9.KQ873 A53.98.A8732.AJ6 T8.T643.KQT654.T J764.KQJ52..9542"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const north = 0
	for _, sc := range calls {
		if sc.Seat == north && sc.M.spoutnik {
			t.Fatalf("North made the Spoutnik double with 7 HL (%q)\nauction: %s",
				sc.M.fr, formatAuction(calls))
		}
	}
}

// TestSpoutnikDoubleCountsLengthPoints: the floor is 8 HL, not 8 H [RC-6].
// 1C-(1D), North holds KJ43.72.QJ9854.3 -- four spades, 7 H and 9 HL with the
// six diamonds: enough for the Spoutnik double.
func TestSpoutnikDoubleCountsLengthPoints(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(south, map[int]*Hand{
		north: hand("KJ43", "72", "QJ9854", "3"),
		south: hand("A92", "AK5", "7", "KQ8765"),
		west:  hand("Q7", "QJ3", "AKT62", "J42"),
	})
	calls := NewEngine(d).Run()

	for _, sc := range calls {
		if sc.Seat != north {
			continue
		}
		if !sc.M.spoutnik {
			t.Fatalf("North's first call = %s (%q), want the Spoutnik double\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		return
	}
	t.Fatalf("North never bid\nauction: %s", formatAuction(calls))
}
