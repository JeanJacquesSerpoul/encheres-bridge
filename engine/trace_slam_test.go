package engine

import (
	"math/rand"
	"strings"
	"testing"
)

// TestBlackwoodAndInvitationTrace: over random deals, every answer to a
// keycard or ace ask, and every answer to a game invitation, carries a trace
// whose conclusion is the call actually made.
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
			}
		}
	}
	for _, k := range []string{"réponse Blackwood", "réponse à la proposition", "suite de Blackwood"} {
		if seen[k] == 0 {
			t.Errorf("aucune %s tracée sur l'échantillon", k)
		}
	}
	t.Logf("%v", seen)
}
