package engine

import (
	"math/rand"
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

// formatAuction writes an auction as "N:1SA E:Passe ...", in French.
func formatAuction(calls []SeatCall) string {
	parts := make([]string, len(calls))
	for i, sc := range calls {
		parts[i] = seatNames[sc.Seat] + ":" + sc.Call.Format("fr")
	}
	return strings.Join(parts, " ")
}

func dealFrom(rng *rand.Rand) *Deal {
	cards := make([]int, 52)
	for i := range cards {
		cards[i] = i
	}
	rng.Shuffle(52, func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	var hands [4]*Hand
	for seat := 0; seat < 4; seat++ {
		h := &Hand{}
		var suits [4][]byte
		for _, c := range cards[seat*13 : seat*13+13] {
			suits[c/13] = append(suits[c/13], rankOrder[c%13])
		}
		for s := Clubs; s <= Spades; s++ {
			h.Suits[s] = sortRanks(string(suits[s]))
		}
		hands[seat] = h
	}
	return &Deal{Dealer: rng.Intn(4), Hands: hands}
}

// TestAuctionsTerminateAndAreLegal fuzzes the engine with random deals.
func TestAuctionsTerminateAndAreLegal(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	maxLen := 0
	for i := 0; i < 2000; i++ {
		deal := dealFrom(rng)
		calls := NewEngine(deal).Run()
		if len(calls) > maxLen {
			maxLen = len(calls)
		}
		if len(calls) >= maxCalls {
			t.Fatalf("deal %d: auction hit the safety cap (%d calls)", i, len(calls))
		}
		// Check seat rotation and call legality.
		var last Call
		hasBid := false
		lastBidSide := -1
		seat := deal.Dealer
		for k, sc := range calls {
			if sc.Seat != seat {
				t.Fatalf("deal %d call %d: expected seat %d, got %d", i, k, seat, sc.Seat)
			}
			switch sc.Call.Kind {
			case KindBid:
				if hasBid && !sc.Call.higherThan(last) {
					t.Fatalf("deal %d call %d: insufficient bid %v over %v", i, k, sc.Call, last)
				}
				last, hasBid, lastBidSide = sc.Call, true, sideOf(sc.Seat)
			case KindDouble:
				if !hasBid || lastBidSide == sideOf(sc.Seat) {
					t.Fatalf("deal %d call %d: illegal double", i, k)
				}
			}
			seat = (seat + 1) % 4
		}
		// Auction must end with three passes after a bid, or four passes.
		n := len(calls)
		if hasBid {
			if n < 4 || calls[n-1].Call.Kind != KindPass || calls[n-2].Call.Kind != KindPass || calls[n-3].Call.Kind != KindPass {
				t.Fatalf("deal %d: auction did not end with three passes", i)
			}
		} else if n != 4 {
			t.Fatalf("deal %d: passed-out auction has %d calls", i, n)
		}
	}
	t.Logf("longest auction over 2000 random deals: %d calls", maxLen)
}

func TestParsePBN(t *testing.T) {
	good := `[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`
	d, err := ParsePBN([]byte(good))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Dealer != 0 {
		t.Fatalf("dealer = %d, want 0 (N)", d.Dealer)
	}
	if d.Hands[0].Suits[Spades] != "AKQ" || newFeatures(d.Hands[0]).hcp != 20 {
		t.Fatalf("north hand parsed wrong: %+v", d.Hands[0])
	}
	// Rotation starting from another seat.
	rotated := `[Dealer "E"]
[Deal "W:T543.52.JT8.KT54 AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76"]`
	d2, err := ParsePBN([]byte(rotated))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d2.Hands[0].Suits[Spades] != "AKQ" {
		t.Fatalf("rotation broken: north spades = %q", d2.Hands[0].Suits[Spades])
	}
	// A void written as "-" instead of an empty suit.
	void := `[Board "1"]
[Dealer "N"]
[Declarer "E"]
[Vulnerable "None"]
[Deal "N:K94.9.Q72.J98742 QJ73.QJ.AJT4.Q65 A82.T654.85.AKT3 T65.AK8732.K963.-"]`
	d3, err := ParsePBN([]byte(void))
	if err != nil {
		t.Fatalf("unexpected error for '-' void: %v", err)
	}
	if d3.Hands[3].Len(Clubs) != 0 || d3.Hands[3].Len(Hearts) != 6 {
		t.Fatalf("west hand parsed wrong: %+v", d3.Hands[3])
	}
	for _, bad := range []string{
		`[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`, // no dealer
		`[Dealer "N"]`, // no deal
		`[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 J9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT5"]`, // 12 cards
		`[Dealer "N"]
[Deal "N:AKQ.KJ4.AQ54.J32 A9.AT63.K762.Q98 8762.Q987.93.A76 T543.52.JT8.KT54"]`, // duplicate ace
	} {
		if _, err := ParsePBN([]byte(bad)); err == nil {
			t.Fatalf("expected error for %s", strings.Split(bad, "\n")[0])
		}
	}
}

// TestThirdSuitRaiseWithoutFit: after 1♦ 1♠ 2♦ 2♥ 3♥, opener holds 5♦ and
// 4♥. Responder, void in hearts, must not pass 3♥: 3NT with the ♣ stopper.
func TestThirdSuitRaiseWithoutFit(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "S"]
[Vulnerable "NS"]
[Deal "S:32.KQ6.K86.QT753 Q9.A982.A5432.A6 K864.JT7543.Q.84 AJT75..JT97.KJ92"]`)
	calls := NewEngine(d).Run()
	const want = "S:Passe W:1K N:Passe E:1P S:Passe W:2K N:Passe E:2C S:Passe W:3C N:Passe E:3SA"
	if got := formatAuction(calls); !strings.HasPrefix(got, want) {
		t.Fatalf("enchères :\n %s\nattendu :\n %s …", got, want)
	}
}

// TestForcingBidsGetAnAnswer: after a forcing bid, partner no longer passes
// for want of a rule. 1♣ 1♠ 2♣ 2♥ 3♥ reaches 4♥; 1SA (2♠) 3♠ (stopper ask)
// gets its 3SA.
func TestForcingBidsGetAnAnswer(t *testing.T) {
	for _, c := range []struct{ pbn, want string }{
		{`[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:K.QT97.AK2.K7653 J654.J832.J9.QJ2 AT983.AK64.876.T Q72.5.QT543.A984"]`,
			"N:1T E:Passe S:1P W:Passe N:2T E:Passe S:2C W:Passe N:3C E:Passe S:4C"},
		{`[Dealer "N"]
[Vulnerable "All"]
[Deal "N:AK.852.Q6543.KQ6 Q76542.AK3.K8.87 J8.Q96.AJ92.AT95 T93.JT74.T7.J432"]`,
			"N:1SA E:2P S:3P W:Passe N:3SA"},
	} {
		got := formatAuction(NewEngine(mustParsePBN(t, c.pbn)).Run())
		if !strings.HasPrefix(got, c.want) {
			t.Errorf("enchères :\n %s\nattendu :\n %s …", got, c.want)
		}
	}
}

// TestNoThreeNTOverJumpRebidWithVoid: 1♠ 1SA 3♠, responder void in spades
// with 7 H: the spades will not run, no 3SA.
func TestNoThreeNTOverJumpRebidWithVoid(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:AK9832.92.J.AQ98 Q765.43.952.KJ62 .QT7.KQT7643.754 JT4.AKJ865.A8.T3"]`)
	got := formatAuction(NewEngine(d).Run())
	if strings.Contains(got, "3SA") || !strings.HasPrefix(got, "N:1P E:Passe S:1SA W:Passe N:3P") {
		t.Fatalf("enchères : %s", got)
	}
}

// TestSlamAfterStaymanFitOver2NT: 2SA 3♣ 3♥, responder has 4 hearts and
// 12 H: the line holds 31 H and the fit, 6♥.
func TestSlamAfterStaymanFitOver2NT(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "E"]
[Vulnerable "None"]
[Deal "N:QJ983.5.7653.J63 K4.KQJ64.J4.AKQ4 T62.932.KQ9.T972 A75.AT87.AT82.85"]`)
	const want = "E:2SA S:Passe W:3T N:Passe E:3C S:Passe W:6C"
	if got := formatAuction(NewEngine(d).Run()); !strings.HasPrefix(got, want) {
		t.Fatalf("enchères :\n %s\nattendu :\n %s …", got, want)
	}
}

// TestStrongHandsInCompetition: the strong advancer or doubler no longer
// passes below game. (1♣) 1♦ – : 2♣ cue-bid with 15 H and no fit, then 3SA;
// (1♦) P P X – 2♥ (8-10): the 16 H doubler with the stopper bids 3SA.
func TestStrongHandsInCompetition(t *testing.T) {
	for _, c := range []struct{ pbn, want string }{
		{`[Dealer "W"]
[Vulnerable "NS"]
[Deal "N:QT.A3.KJ542.QJ64 632.8542.T9873.5 AK97.QT6.AQ.T973 J854.KJ97.6.AK82"]`,
			"W:1T N:1K E:Passe S:2T W:Passe N:3SA"},
		{`[Dealer "E"]
[Vulnerable "All"]
[Deal "N:J98.A95.AK3.K432 AK42.KJ.J965.J87 65.8742.QT82.AQT QT73.QT63.74.965"]`,
			"E:1K S:Passe W:Passe N:X E:Passe S:2C W:Passe N:3SA"},
	} {
		got := formatAuction(NewEngine(mustParsePBN(t, c.pbn)).Run())
		if !strings.HasPrefix(got, c.want) {
			t.Errorf("enchères :\n %s\nattendu :\n %s …", got, c.want)
		}
	}
}

// TestLevy2NTOverWeakTwoDouble: (2♥) X – 2SA forcing de manche (Lévy),
// 3♣ contre banal, 3♥ cue-bid (4 piques sans tenue ♥), 4♠ par le contreur.
func TestLevy2NTOverWeakTwoDouble(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "N"]
[Vulnerable "NS"]
[Deal "N:T9864.A85.Q5.AJ5 7532.63.AK83.Q92 .KQJT92.J642.863 AKQJ.74.T97.KT74"]`)
	const want = "N:Passe E:Passe S:2C W:X N:Passe E:2SA S:Passe W:3T N:Passe E:3C S:Passe W:4P"
	if got := formatAuction(NewEngine(d).Run()); !strings.HasPrefix(got, want) {
		t.Fatalf("enchères :\n %s\nattendu :\n %s …", got, want)
	}
}

// TestDefenceAgainstPreempts: (3♣) X – 4♣ cue-bid with both majors 4-4,
// the doubler names his four-card major; (4♠) 4SA two-suiter, partner
// names the cheapest acceptable minor.
func TestDefenceAgainstPreempts(t *testing.T) {
	for _, c := range []struct{ pbn, want string }{
		{`[Dealer "N"]
[Vulnerable "EW"]
[Deal "N:95.T.632.AJ98752 AJT74.AJ532.AK7. 862.K96.T984.QT3 KQ3.Q874.QJ5.K64"]`,
			"N:3T E:X S:Passe W:4T N:Passe E:4C"},
		{`[Dealer "W"]
[Vulnerable "All"]
[Deal "N:.A7.AK984.AKJ862 9.KT9843.Q752.T5 K874.J52.JT63.93 AQJT6532.Q6..Q74"]`,
			"W:4P N:4SA E:Passe S:5K"},
	} {
		got := formatAuction(NewEngine(mustParsePBN(t, c.pbn)).Run())
		if !strings.HasPrefix(got, c.want) {
			t.Errorf("enchères :\n %s\nattendu :\n %s …", got, c.want)
		}
	}
}
