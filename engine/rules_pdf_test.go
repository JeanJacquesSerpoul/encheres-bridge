package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
)

// TestRulesPDFUpToDate: each system of cli/systems/index.json carries its
// description, rules.pdf and rules.en.pdf, and each must hold the fingerprint
// of the current rules.yaml (metadata Subject "sha256=…", written by
// tools/python_tools/rules_pdf.py). A rule changed without regenerating the
// PDF fails here, so the help the page opens never describes another system
// than the one the engine plays.
func TestRulesPDFUpToDate(t *testing.T) {
	forEachSystem(t, func(t *testing.T, s *testSystem) {
		yaml, err := os.ReadFile(s.file("rules.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(bytes.ReplaceAll(yaml, []byte("\r\n"), []byte("\n")))
		want := []byte("sha256=" + hex.EncodeToString(sum[:]))
		for _, name := range []string{"rules.pdf", "rules.en.pdf"} {
			body, err := os.ReadFile(s.file(name))
			if err != nil {
				t.Errorf("%v (générer : %s)", err, s.regenerateCmd())
				continue
			}
			if !bytes.Contains(body, want) {
				t.Errorf("%s ne décrit plus rules.yaml : le régénérer (%s)", s.file(name), s.regenerateCmd())
			}
		}
	})
}
