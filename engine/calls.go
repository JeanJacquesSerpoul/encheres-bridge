package engine

import "fmt"

type CallKind int

const (
	KindPass CallKind = iota
	KindBid
	KindDouble
	KindRedouble
)

// Call is one action in the auction.
type Call struct {
	Kind   CallKind
	Level  int
	Strain Strain
}

var passCall = Call{Kind: KindPass}
var doubleCall = Call{Kind: KindDouble}

func bid(level int, s Strain) Call { return Call{Kind: KindBid, Level: level, Strain: s} }

func (c Call) IsBid() bool { return c.Kind == KindBid }

// steps ranks bids on the auction ladder (1C=0 ... 7NT=34).
func (c Call) steps() int { return (c.Level-1)*5 + int(c.Strain) }

func (c Call) higherThan(o Call) bool { return c.steps() > o.steps() }

var strainEN = [5]string{"C", "D", "H", "S", "NT"}
var strainFR = [5]string{"T", "K", "C", "P", "SA"}

// Format renders a call in the requested language ("en" or "fr").
func (c Call) Format(lang string) string {
	fr := lang == "fr"
	switch c.Kind {
	case KindPass:
		if fr {
			return "Passe"
		}
		return "Pass"
	case KindDouble:
		// X et XX dans les deux langues, comme sur les boîtes à enchères.
		return "X"
	case KindRedouble:
		return "XX"
	}
	names := strainEN
	if fr {
		names = strainFR
	}
	return fmt.Sprintf("%d%s", c.Level, names[c.Strain])
}
