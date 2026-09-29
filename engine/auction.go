package engine

// The auction: the four hands bid in turn from the dealer, each call the first
// rule that applies to the sequence its pair sees -- a port of
// generate_auction() and _legal() in tools/python_tools/pbn_auction.py.

import "strings"

// maxCalls caps an auction, as pbn_auction.py does; three passes end any real
// one long before.
const maxCalls = 60

// fallback says why a call was not given by a rule.
type fallback int

const (
	byRule    fallback = iota
	noRule             // no rule covers the sequence: pass by default
	illegalBy          // the rule found gives an illegal call: pass by default
)

// SeatCall is a call as recorded in the auction.
type SeatCall struct {
	Seat   int
	Call   Call
	Rule   *Rule    // the rule that gave the call, nil for a default pass
	Why    fallback // why there is no rule
	Denied *Rule    // illegalBy: the rule whose call was illegal
	Seq    []string // the sequence the pair saw, in rule notation
	Trace  []traceStep
}

// Engine bids one deal with a rule set.
type Engine struct {
	deal    *Deal
	rules   *RuleSet
	options map[string]bool
}

// NewEngine prepares the auction of a deal with the rules loaded by SetRules
// (an empty set if none: every call is then a default pass).
func NewEngine(deal *Deal) *Engine {
	rs := activeRules.Load()
	if rs == nil {
		rs = &RuleSet{}
	}
	return &Engine{deal: deal, rules: rs, options: map[string]bool{}}
}

func sideOf(seat int) int { return seat % 2 }

// ruleToken writes a call the way the rules do: P, X, XX, 1C ... 7NT.
func ruleToken(c Call) string {
	switch c.Kind {
	case KindPass:
		return "P"
	case KindDouble:
		return "X"
	case KindRedouble:
		return "XX"
	}
	return c.Format("en")
}

// pairSequence is the auction as the pair of `seat` reads it: its own calls,
// passes included, and the opponents' calls in parentheses, their passes left
// out.
func pairSequence(history []SeatCall, seat int) []string {
	seq := []string{}
	for _, sc := range history {
		switch {
		case sideOf(sc.Seat) == sideOf(seat):
			seq = append(seq, ruleToken(sc.Call))
		case sc.Call.Kind != KindPass:
			seq = append(seq, "("+ruleToken(sc.Call)+")")
		}
	}
	return seq
}

// legalCall reports whether seat may make call after history.
func legalCall(call Call, history []SeatCall, seat int) bool {
	if call.Kind == KindPass {
		return true
	}
	var last *SeatCall
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Call.Kind != KindPass {
			last = &history[i]
			break
		}
	}
	if call.Kind == KindDouble || call.Kind == KindRedouble {
		if last == nil || sideOf(last.Seat) == sideOf(seat) {
			return false
		}
		if call.Kind == KindDouble {
			return last.Call.Kind == KindBid
		}
		return last.Call.Kind == KindDouble
	}
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Call.IsBid() {
			return call.higherThan(history[i].Call)
		}
	}
	return true
}

func auctionOver(history []SeatCall) bool {
	n := len(history)
	anyBid := false
	for _, sc := range history {
		if sc.Call.Kind != KindPass {
			anyBid = true
			break
		}
	}
	if !anyBid {
		return n >= 4
	}
	return n >= 3 && history[n-1].Call.Kind == KindPass && history[n-2].Call.Kind == KindPass &&
		history[n-3].Call.Kind == KindPass
}

// Run bids the deal from the dealer until the auction ends.
func (e *Engine) Run() []SeatCall {
	var feats [4]*features
	for seat, h := range e.deal.Hands {
		feats[seat] = newFeatures(h)
	}
	trump := [2]string{} // agreed suit per pair (N/S = 0, E/W = 1)
	var history []SeatCall
	for i := range maxCalls {
		seat := (e.deal.Dealer + i) % 4
		seq := pairSequence(history, seat)
		sc := SeatCall{Seat: seat, Call: passCall, Seq: seq}
		r := e.rules.choose(seq, feats[seat], e.options, trump[sideOf(seat)])
		switch {
		case r == nil:
			sc.Why = noRule
		case !legalCall(r.Call, history, seat):
			sc.Why, sc.Denied = illegalBy, r
		default:
			sc.Rule, sc.Call = r, r.Call
			if r.Trump != "" {
				trump[sideOf(seat)] = r.Trump
			}
		}
		sc.Trace = traceCall(&sc, feats[seat])
		history = append(history, sc)
		if auctionOver(history) {
			break
		}
	}
	return history
}

// comment is the call's explanation in the response language: the rule's
// meaning, or why the pass was a default one.
func (sc *SeatCall) comment(lang string) string {
	fr := lang == "fr"
	switch {
	case sc.Rule != nil && fr:
		return sc.Rule.Meaning
	case sc.Rule != nil:
		return sc.Rule.MeaningEN
	case sc.Why == illegalBy && fr:
		return "passe par défaut (la règle " + sc.Denied.ID + " donnerait " + sc.Denied.CallText + ", illégal)"
	case sc.Why == illegalBy:
		return "default pass (rule " + sc.Denied.ID + " would give " + sc.Denied.CallText + ", which is illegal)"
	case fr:
		return "passe par défaut (aucune règle pour cette séquence)"
	}
	return "default pass (no rule for this sequence)"
}

// formatSeq writes a pair's sequence with the calls of the response language,
// the opponents' still in parentheses.
func formatSeq(seq []string, lang string) string {
	out := make([]string, len(seq))
	for i, tok := range seq {
		inner := strings.Trim(tok, "()")
		c := parseRuleCall(inner)
		if inner != tok {
			out[i] = "(" + c.Format(lang) + ")"
		} else {
			out[i] = c.Format(lang)
		}
	}
	return strings.Join(out, " ")
}
