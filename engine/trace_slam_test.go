package engine

import (
	"math/rand"
	"strings"
	"testing"
)

// TestBlackwoodAndInvitationTrace: over random deals, every answer to a
// keycard or ace ask, every answer to a game invitation and every step of a
// slam exploration carries a trace whose conclusion is the call actually made.
func TestBlackwoodAndInvitationTrace(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	seen := map[string]int{}
	for i := 0; i < 1500; i++ {
		for _, sc := range NewEngine(randomDeal(rng)).Run() {
			if len(sc.Trace) == 0 {
				continue
			}
			first := sc.Trace[0].fr
			callFR, _ := callSym(sc.Call)
			var lastHeld string
			for _, s := range sc.Trace {
				if !s.note && s.ok {
					lastHeld = s.fr
				}
				if s.fr == "" || s.en == "" {
					t.Fatalf("étape sans libellé dans les deux langues : %+v", s)
				}
			}
			end := sc.Trace[len(sc.Trace)-1]
			switch {
			case strings.Contains(first, "demandé les clefs"):
				seen["réponse Blackwood"]++
				if sc.Call.IsBid() && !strings.HasSuffix(lastHeld, "→ "+callFR) {
					t.Errorf("réponse %s, dernier test retenu %q\n%v", callFR, lastHeld, sc.Trace)
				}
			case strings.Contains(first, "propose la manche"):
				seen["réponse à la proposition"]++
				switch {
				case sc.Call.Kind == KindPass:
					if !end.note || !strings.Contains(end.fr, "Passe") {
						t.Errorf("passe sans conclusion « Passe »\n%v", sc.Trace)
					}
				case strings.HasPrefix(lastHeld, "→ la manche"):
					if !strings.HasSuffix(lastHeld, callFR) {
						t.Errorf("manche %s, dernier test retenu %q", callFR, lastHeld)
					}
				case !strings.HasPrefix(lastHeld, "refuser") && !strings.HasPrefix(lastHeld, "chicane"):
					t.Errorf("%s sans test retenu cohérent : %q\n%v", callFR, lastHeld, sc.Trace)
				}
			case strings.Contains(first, "demande de clefs a reçu sa réponse"):
				seen["suite de Blackwood"]++
			case strings.Contains(first, "sacrifier") || strings.Contains(first, "contrat sérieux") ||
				strings.Contains(first, "couleur d'aide") || strings.Contains(first, "Checkback") ||
				strings.Contains(first, "Roudi") || strings.Contains(first, "décidée dès le tour précédent") &&
				len(sc.Trace) > 1 && (strings.Contains(sc.Trace[1].fr, "Checkback") || strings.Contains(sc.Trace[1].fr, "Roudi")):
				seen["sacrifice, essai, Checkback, Roudi"]++
				if !namesCall(sc.Trace, callFR) {
					t.Errorf("%s : la trace ne conclut pas sur l'enchère\n%v", callFR, sc.Trace)
				}
			case strings.Contains(first, "contrôle") || strings.Contains(first, "zone de chelem") ||
				strings.Contains(first, "le soutenir d'abord"):
				seen["exploration du chelem"]++
				// The call is named by the last test that held or by the
				// closing note (a Blackwood ask, a control, the game).
				named := strings.HasSuffix(lastHeld, "→ "+callFR) || strings.HasSuffix(lastHeld, ": "+callFR) ||
					strings.Contains(lastHeld, "Blackwood") && callFR == "4SA" ||
					strings.Contains(lastHeld, "relais contrôle") ||
					end.note && (strings.HasSuffix(end.fr, callFR) || strings.Contains(end.fr, "4SA") && callFR == "4SA")
				if sc.Call.IsBid() && !named {
					t.Errorf("%s : la trace ne conclut pas sur l'enchère\n%v", callFR, sc.Trace)
				}
			}
		}
	}
	for _, k := range []string{"réponse Blackwood", "réponse à la proposition", "suite de Blackwood", "exploration du chelem", "sacrifice, essai, Checkback, Roudi"} {
		if seen[k] == 0 {
			t.Errorf("aucune %s tracée sur l'échantillon", k)
		}
	}
	t.Logf("%v", seen)
}

// namesCall reports whether a held test or a note of the trace concludes on
// the call: the call follows the last arrow of its label.
func namesCall(trace []traceStep, callFR string) bool {
	for _, s := range trace {
		if !s.ok && !s.note {
			continue
		}
		if i := strings.LastIndex(s.fr, "→"); i >= 0 &&
			strings.Contains(strings.ToLower(s.fr[i:]), strings.ToLower(callFR)) {
			return true
		}
	}
	return false
}
