package engine

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// tracedOpening runs opening() for a hand in the given position (0 = dealer,
// 2 = third seat...), with a tracer, as Run does.
func tracedOpening(h *Hand, passes int) (Call, []traceStep) {
	e := &Engine{opener: -1}
	for seat := range 4 {
		e.ps[seat] = &playerState{seat: seat, hand: hand("", "", "", ""), shownMax: 40}
	}
	seat := passes % 4
	e.ps[seat].hand = h
	for i := range passes {
		e.calls = append(e.calls, SeatCall{Seat: i, Call: passCall})
	}
	e.tr = &tracer{}
	c, _ := e.opening(e.ps[seat])
	return c, e.tr.steps
}

// lastTrue returns the label (French) of the last test that held.
func lastTrue(steps []traceStep) string {
	for i := len(steps) - 1; i >= 0; i-- {
		if steps[i].ok && !steps[i].note {
			return steps[i].fr
		}
	}
	return ""
}

func TestOpeningTrace(t *testing.T) {
	cases := []struct {
		name     string
		h        *Hand
		passes   int
		want     string // call, French notation
		lastTest string // substring of the last test that held
		falseFor []string
	}{
		{"mineures 4-4, 13 H → 1♦", hand("K52", "Q7", "AJ84", "K963"), 0, "1K",
			"4-4 ou 5-5 → 1♦", []string{"[O-1]", "[O-2]", "[O-3]", "[O-4]", "[O-5]", "5 ♠ ou plus", "5 ♥ ou plus"}},
		{"5♠-5♣ avec 14 H → 1♣", hand("AKJ84", "54", "7", "AQ852"), 0, "1T",
			"5♠-5♣ avec 14 H", nil},
		{"2 majeur faible", hand("84", "KQJ964", "Q73", "52"), 0, "2C",
			"2 majeur faible", []string{"[O-6]"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, steps := tracedOpening(tc.h, tc.passes)
			if got := c.Format("fr"); got != tc.want {
				t.Fatalf("ouverture = %s, attendu %s", got, tc.want)
			}
			if got := lastTrue(steps); !strings.Contains(got, tc.lastTest) {
				t.Fatalf("dernier test vrai = %q, attendu %q\n%v", got, tc.lastTest, steps)
			}
			for _, want := range tc.falseFor {
				found := false
				for _, s := range steps {
					if strings.Contains(s.fr, want) {
						found = true
						if s.ok {
							t.Errorf("le test %q ne devait pas tenir", s.fr)
						}
					}
				}
				if !found {
					t.Errorf("test %q absent de la trace", want)
				}
			}
			for _, s := range steps {
				if s.fr == "" || s.en == "" {
					t.Errorf("étape sans libellé dans les deux langues : %+v", s)
				}
			}
		})
	}
}

// TestOpeningTracePass: 9 H scattered — the preempt gate [O-7] opens, the
// concentration test then refuses, and the trace ends on the [O-8] note.
func TestOpeningTracePass(t *testing.T) {
	c, steps := tracedOpening(hand("K52", "Q73", "J984", "963"), 0)
	if c.Kind != KindPass {
		t.Fatalf("ouverture = %s, attendu Passe", c.Format("fr"))
	}
	last := steps[len(steps)-1]
	if !last.note || !strings.Contains(last.fr, "[O-8]") {
		t.Fatalf("dernière étape = %+v, attendu la note [O-8]", last)
	}
	if got := lastTrue(steps); !strings.Contains(got, "[O-7]") {
		t.Fatalf("dernier test vrai = %q, attendu la porte des barrages [O-7]", got)
	}
	for _, s := range steps {
		if strings.Contains(s.fr, "moitié des points") {
			if s.ok || s.depth != 1 {
				t.Fatalf("test de concentration = %+v, attendu faux et imbriqué", s)
			}
			return
		}
	}
	t.Fatal("test de concentration absent de la trace")
}

// TestNilTracer: an untraced decision records nothing and decides the same.
func TestNilTracer(t *testing.T) {
	h := hand("K52", "Q7", "AJ84", "K963")
	c1, _ := openingWith(h, 0)
	c2, steps := tracedOpening(h, 0)
	if c1 != c2 {
		t.Fatalf("avec trace %s, sans trace %s", c2.Format("fr"), c1.Format("fr"))
	}
	if len(steps) == 0 {
		t.Fatal("la trace de l'ouverture est vide")
	}
	var nilTracer *tracer
	if !nilTracer.check(true, "a", "b", "") || nilTracer.check(false, "a", "b", "") {
		t.Fatal("un traceur nil doit rendre le booléen tel quel")
	}
	nilTracer.note("a", "b")
	nilTracer.in()
	nilTracer.out()
}

// TestBidJSONTrace: the opening carries its trace in the JSON, in the
// requested language; a call of a situation not yet traced carries none.
func TestBidJSONTrace(t *testing.T) {
	pbn, err := os.ReadFile("testdata/d1.pbn")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	for _, lang := range []string{"fr", "en"} {
		data, err := BidJSON(pbn, lang)
		if err != nil {
			t.Fatalf("BidJSON: %v", err)
		}
		var resp bidResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		open := resp.Auction[0]
		if open.Bid != "2SA" && open.Bid != "2NT" {
			t.Fatalf("ouverture = %s, attendu 2SA/2NT", open.Bid)
		}
		if len(open.Trace) == 0 {
			t.Fatalf("[%s] l'ouverture n'a pas de trace", lang)
		}
		held := ""
		for _, s := range open.Trace {
			if s.Ok && !s.Note {
				held = s.Label
			}
		}
		want := map[string]string{"fr": "→ 2SA", "en": "→ 2NT"}[lang]
		if !strings.Contains(held, want) {
			t.Fatalf("[%s] test retenu = %q, attendu %q", lang, held, want)
		}
		// East's answer is an overcall situation, not traced yet.
		if len(resp.Auction[1].Trace) != 0 {
			t.Fatalf("[%s] l'enchère d'Est ne devait pas porter de trace : %+v", lang, resp.Auction[1].Trace)
		}
	}
}
