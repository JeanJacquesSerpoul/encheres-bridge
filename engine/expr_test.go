package engine

import "testing"

// The hand every expression below is evaluated on: AKQ.KJ4.AQ54.J32
// (spades AKQ, hearts KJ4, diamonds AQ54, clubs J32) -- 20 H, 4-3-3-3.
func exprHand() *features { return newFeatures(parseRuleHand("AKQ.KJ4.AQ54.J32")) }

func TestExprEval(t *testing.T) {
	f := exprHand()
	cases := []struct {
		src  string
		want string // Python's repr of the result
	}{
		{"hcp", "20"},
		{"20 <= hcp <= 21", "True"},
		{"9 <= hcp <= 12", "False"},
		{"1 < 2 < 2", "False"}, // chained: both links must hold
		{"hcp >= 20 and S >= 3", "True"},
		{"hcp > 25 or D", "4"}, // or returns the operand
		{"0 and hcp", "0"},     // and returns the first falsy operand
		{"balanced + 1", "2"},  // a bool counts as 1
		{"ace('S') + king('S')", "2"},
		{"keycards('S')", "3"},
		{"shape", "'4333'"},
		{"shape in ('4333', '4432')", "True"},
		{"shape not in ('4333',)", "False"},
		{"'33' in shape", "True"},
		{"H in (3, 4)", "True"},
		{"True in (1, 2)", "True"}, // 1 == True
		{"ptricks >= 8.5", "False"},
		{"qtricks", "4.0"},
		{"-S + 5", "2"},
		{"not hcp", "False"},
		{"not hcp == 3", "True"}, // not binds looser than ==
		{"min(S, H, D, C)", "3"},
		{"max((S, H, D))", "4"},
		{"true", "True"},
		{"false or ()", "()"},
		{"stop('C') and not stop('H') == false", "False"},
	}
	for _, c := range cases {
		n, err := compileCond(c.src)
		if err != nil {
			t.Errorf("%s : %v", c.src, err)
			continue
		}
		v, err := n.eval(f)
		if err != nil {
			t.Errorf("%s : %v", c.src, err)
			continue
		}
		if got := v.String(); got != c.want {
			t.Errorf("%s = %s, attendu %s", c.src, got, c.want)
		}
	}
}

// TestExprContext: the auction context names (setContext).
func TestExprContext(t *testing.T) {
	f := exprHand()
	f.lvl, f.pSuit, f.pLen, f.pForcing = 2, "H", 5, true
	for src, want := range map[string]string{
		"lvl":                        "2",
		"p_suit":                     "'H'",
		"p_len":                      "5",
		"fit":                        "8", // three hearts in hand + five promised
		"p_forcing":                  "True",
		"p_suit == 'H' and fit >= 8": "True",
		"fit - 6 == 2 and lvl < 3":   "True",
		"p_suit in ('S', 'H') and H": "3",
	} {
		n, err := compileCond(src)
		if err != nil {
			t.Errorf("%s : %v", src, err)
			continue
		}
		v, err := n.eval(f)
		if err != nil {
			t.Errorf("%s : %v", src, err)
			continue
		}
		if got := v.String(); got != want {
			t.Errorf("%s = %s, attendu %s", src, got, want)
		}
	}
	f.pSuit = ""
	if v, _ := mustEval(t, "fit", f); v != "0" {
		t.Errorf("fit sans couleur du partenaire = %s, attendu 0", v)
	}
}

func mustEval(t *testing.T, src string, f *features) (string, error) {
	t.Helper()
	n, err := compileCond(src)
	if err != nil {
		t.Fatal(err)
	}
	v, err := n.eval(f)
	return v.String(), err
}

func TestExprRejects(t *testing.T) {
	for _, src := range []string{
		"hcp * 2",         // no multiplication
		"foo > 1",         // unknown name
		"hcp(",            // incomplete
		"hcp is 3",        // no identity test
		"+hcp",            // no unary plus
		"hcp('S')",        // not a function
		"ace['S']",        // no subscript
		"x.y",             // no attribute
		"hcp >= 12 hcp",   // trailing garbage
		"'unterminated",   // string never closed
		"lambda: 1",       // keyword
		"hcp if S else H", // no ternary
	} {
		if _, err := compileCond(src); err == nil {
			t.Errorf("%q : accepté, devait être refusé", src)
		}
	}
}

func TestExprRuntimeErrors(t *testing.T) {
	f := exprHand()
	for _, src := range []string{"shape < 3", "ace('Z')", "ace(1)", "ace()", "hcp in 3"} {
		n, err := compileCond(src)
		if err != nil {
			t.Errorf("%q : %v", src, err)
			continue
		}
		if _, err := n.eval(f); err == nil {
			t.Errorf("%q : évalué sans erreur", src)
		}
	}
}
