package engine

import (
	"testing"
)

// TestSlamExploredWhenNoKeycardIsMissing is the reported deal. South opens 2D
// on AK8632 AK A65 AK -- 25 honour points, all four aces and three kings --
// North answers the ace step, South names the spades and the pair cue-bids
// 3D (North's king) and 4C (South's ace). North then signs off in 4S, having
// no further control to show, and South passed: 29 HLD opposite a floor of 3
// counts 32, one short of the slam zone.
//
// The count is what fails here, not the hands. A king shown by a cue-bid
// raises partner's floor by three, so a pair holding every ace and three
// kings still adds up to 32. South can see that no keycard is missing -- his
// own five -- so the slam turns on the trump queen and the kings, not on the
// total, and the ask cannot come back short. Partner's sign-off says he has
// no more controls, not that there is no slam.
func TestSlamExploredWhenNoKeycardIsMissing(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "EW"]
[Deal "S:AK8632.AK.A65.AK QT5.T7432.J43.98 J97.965.KQ72.T42 4.QJ8.T98.QJ7653"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract.Level < 6 || contract.Strain != SSpades {
		t.Fatalf("final contract = %s, want a spade slam: the side holds every keycard\nauction: %s",
			contract.Format("fr"), formatAuction(calls))
	}
}

// TestSlamNotExploredWhenAKeycardIsMissing is the guard on the other side, and
// it is the deal of TestNoKeycardAskBelowTheSlamZoneInAMajor: East holds four
// keycards and the fifth -- the club ace -- is with the opponents. Its club
// control is a king, second round, which the control exchange accepts but the
// keycard count does not. Below the slam zone that hand must still stop in
// game rather than buy the five level for an answer it already knows.
func TestSlamNotExploredWhenAKeycardIsMissing(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Deal "S:J2.Q9754.94.A983 64.KT32.K863.Q72 T73.J86.QJT7.JT5 AKQ985.A.A52.K64"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(4, SSpades) {
		t.Fatalf("final contract = %s, want 4P\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestNoKeycardAskFarBelowTheSlamZone is the reported deal. South opens 2D on
// AT43 AK532 AK AQ, West overcalls 2S, North passes (under 5H), South shows
// the hearts and North, with J T987 763 98652, signs off in 4H. South holds
// all five keycards, so the shortcut above used to fire: 4NT - 5C - 6H, on a
// combined floor of 30 with nothing from partner but four small trumps.
//
// The answer was known before it was asked -- "0 or 3" facing five keycards
// can only be 0 -- and the slam never came within a point of the zone. The
// shortcut makes up for one missing point, not three: South passes 4H.
func TestNoKeycardAskFarBelowTheSlamZone(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "N"]
[Vulnerable "EW"]
[Deal "S:AT43.AK532.AK.AQ KQ9865.64.QJ842. J.T987.763.98652 72.QJ.T95.KJT743"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(4, SHearts) {
		t.Fatalf("final contract = %s, want 4C\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}

// TestKeycardAskWhenPartnerDeniedTheMissingAce is the reported deal. South
// opens 1S on T9543 AQJ72 A7 5, North (AKQJ86 - 84 QJT84) bids 2C, South
// shows the hearts, North sets the trump with 3S, South answers 3NT "yes but"
// (the club singleton) and North cue-bids the heart void, skipping 4C and 4D.
// South used to sign off in 4S on 29 combined, one short of the 33 an ask
// normally needs in a major, with twelve tricks on top: only the club ace is
// lost.
//
// The skip is what the count misses. North has no club control, so the club
// ace is the opponents'; South holds the other two side aces and controls the
// clubs with his singleton. Partner's keycards can only be the trump ace and
// king: two of them means the club ace is the one missing, and the slam is
// bid; fewer stops in five.
func TestKeycardAskWhenPartnerDeniedTheMissingAce(t *testing.T) {
	d, err := ParsePBN([]byte(`[Dealer "S"]
[Vulnerable "EW"]
[Deal "S:T9543.AQJ72.A7.5 .T8654.JT9632.K9 AKQJ86..84.QJT84 72.K93.KQ5.A7632"]`))
	if err != nil {
		t.Fatalf("bad deal: %v", err)
	}
	calls := NewEngine(d).Run()
	contract, _, _ := finalContract(calls)
	if contract != bid(6, SSpades) {
		t.Fatalf("final contract = %s, want 6P\nauction: %s", contract.Format("fr"), formatAuction(calls))
	}
}
