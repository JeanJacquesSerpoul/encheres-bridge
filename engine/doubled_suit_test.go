package engine

import (
	"strings"
	"testing"
)

// TestDoubledSuitSkipsPasses pins the suit a double takes out when the double
// does not sit next to the bid it doubles. A reopening double comes after two
// passes, and a pass carries the zero Strain -- Clubs -- so reading the
// neighbouring call named Clubs for every balancing double, whatever the
// opponents had actually opened.
func TestDoubledSuitSkipsPasses(t *testing.T) {
	cases := []struct {
		name  string
		calls []SeatCall
		want  Suit
		ok    bool
	}{
		{
			name: "contre direct",
			calls: []SeatCall{
				{Seat: 1, Call: bid(1, SHearts)},
				{Seat: 2, Call: doubleCall},
			},
			want: Hearts, ok: true,
		},
		{
			name: "contre de réveil, deux passes avant",
			calls: []SeatCall{
				{Seat: 1, Call: bid(2, SSpades)},
				{Seat: 2, Call: passCall},
				{Seat: 3, Call: passCall},
				{Seat: 0, Call: doubleCall},
			},
			want: Spades, ok: true,
		},
		{
			name: "réveil sur une ouverture mineure passée deux fois",
			calls: []SeatCall{
				{Seat: 0, Call: bid(1, SDiamonds)},
				{Seat: 1, Call: passCall},
				{Seat: 2, Call: passCall},
				{Seat: 3, Call: doubleCall},
			},
			want: Diamonds, ok: true,
		},
		{
			name: "rien à prendre : Sans-Atout",
			calls: []SeatCall{
				{Seat: 1, Call: bid(1, SNoTrump)},
				{Seat: 2, Call: passCall},
				{Seat: 3, Call: passCall},
				{Seat: 0, Call: doubleCall},
			},
			ok: false,
		},
		{
			name: "rien à prendre : personne n'a encore parlé",
			calls: []SeatCall{
				{Seat: 0, Call: passCall},
				{Seat: 1, Call: doubleCall},
			},
			ok: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doubler := tc.calls[len(tc.calls)-1].Seat
			e := &Engine{calls: tc.calls}
			got, ok := e.doubledSuit(doubler)
			if ok != tc.ok {
				t.Fatalf("doubledSuit ok = %v, want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("doubledSuit = %s, want %s", suitNameFR[got], suitNameFR[tc.want])
			}
		})
	}
}

// TestReopeningDoubleAnswerAvoidsTheirSuit is the auction the defect produced
// (audit du par, donne 362): East opens a weak 2S, two passes, North reopens
// with a takeout double, and South -- holding SA742 -- answered 3S, naming
// the very suit the double asked to take out. answerDouble does exclude the
// doubled suit; it had simply been told the double took out Clubs.
func TestReopeningDoubleAnswerAvoidsTheirSuit(t *testing.T) {
	pbn := `[Dealer "E"]
[Vulnerable "All"]
[Deal "E:KQ9653.32.T5.JT7 A742.654.9864.A6 J.QJT87.A32.K943 T8.AK9.KQJ7.Q852"]`
	d, err := ParsePBN([]byte(pbn))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()

	const south = 2
	doubled := false
	for _, sc := range calls {
		if sc.Call.Kind == KindDouble {
			doubled = true
			continue
		}
		if !doubled || sc.Seat != south || !sc.Call.IsBid() {
			continue
		}
		if sc.Call.Strain == SSpades {
			t.Fatalf("South answers the double with %s (%s): that is East's suit\nauction: %s",
				sc.Call.Format("fr"), sc.M.fr, formatAuction(calls))
		}
		return
	}
	if !doubled {
		t.Fatalf("North did not reopen with a double\nauction: %s", formatAuction(calls))
	}
	if !strings.Contains(formatAuction(calls), "S:") {
		t.Fatalf("South never called\nauction: %s", formatAuction(calls))
	}
}
