package engine

import "testing"

// TestThreeNTOverPreempt: North opens 3D, East holds K854 AT5 KQJ42 A -- 17
// H, the diamonds stopped several times over, no void. Nothing in the old
// overcall table fitted (the double wanted 18 H, the suit overcall a suit of
// theirs) and East passed with game in hand. East bids a natural 3NT [I-4b].
func TestThreeNTOverPreempt(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "N"]
[Deal "N:32.J.AT98653.QT9 K854.AT5.KQJ42.A AT6.K98643.7.532 QJ97.Q72..KJ8764"]`)
	calls := NewEngine(d).Run()
	const north, east = 0, 1
	wantCall(t, calls, north, 1, "3K")
	wantCall(t, calls, east, 1, "3SA")
}

// TestTakeoutDoubleOfPreempt: over 3H, 16 H with a singleton heart and three
// cards or more in every other suit doubles for takeout [I-6b].
func TestTakeoutDoubleOfPreempt(t *testing.T) {
	const north, east, south, west = 0, 1, 2, 3
	d := dealWith(north, map[int]*Hand{
		north: hand("75", "KQJT862", "83", "92"),
		east:  hand("AQ96", "4", "KJ72", "AJ5"),
		south: hand("KT83", "A93", "Q954", "86"),
		west:  hand("J42", "75", "AT6", "KQT743"),
	})
	calls := NewEngine(d).Run()
	wantCall(t, calls, north, 1, "3C")
	wantCall(t, calls, east, 1, "Contre")
}

// TestMajorGameOnHonours: 1C - 1H - 2H, and East holds A842 AQ94 J6 QT6. The
// 27 HLD of a major game found 14 + 12 and only invited, West declined, and
// 25 honour points with an eight-card fit played 3H. The honours alone reach
// game [E-9c]: East bids 4H.
func TestMajorGameOnHonours(t *testing.T) {
	d := mustParsePBN(t, `[Dealer "S"]
[Deal "N:T965.JT8.A982.72 A842.AQ94.J6.QT6 Q3.K3.QT743.K843 KJ7.7652.K5.AJ95"]`)
	calls := NewEngine(d).Run()
	const east, west = 1, 3
	wantCall(t, calls, west, 2, "2C")
	wantCall(t, calls, east, 2, "4C")
}
