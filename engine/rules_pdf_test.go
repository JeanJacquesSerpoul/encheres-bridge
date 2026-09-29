package engine

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestRulesPDFUpToDate: each rules file that cli/rules/index.json gives a PDF
// description must carry, in that PDF, the fingerprint of its current text
// (metadata Subject "sha256=…", written by tools/python_tools/rules_pdf.py).
// A rule changed without regenerating the PDF fails here, so the help the page
// opens never describes another system than the one the engine plays.
func TestRulesPDFUpToDate(t *testing.T) {
	const dir = "../cli/rules"
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index struct {
		Systems []struct {
			File string          `json:"file"`
			PDF  json.RawMessage `json:"pdf"`
		} `json:"systems"`
	}
	if err := json.Unmarshal(data, &index); err != nil {
		t.Fatal(err)
	}
	for _, sys := range index.Systems {
		if len(sys.PDF) == 0 {
			continue
		}
		var pdfs []string
		var one string
		var byLang map[string]string
		switch {
		case json.Unmarshal(sys.PDF, &one) == nil:
			pdfs = []string{one}
		case json.Unmarshal(sys.PDF, &byLang) == nil:
			for _, p := range byLang {
				pdfs = append(pdfs, p)
			}
		default:
			t.Fatalf("%s : champ pdf illisible", sys.File)
		}
		yaml, err := os.ReadFile(filepath.Join(dir, sys.File))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(bytes.ReplaceAll(yaml, []byte("\r\n"), []byte("\n")))
		want := []byte("sha256=" + hex.EncodeToString(sum[:]))
		for _, p := range pdfs {
			body, err := os.ReadFile(filepath.Join(dir, p))
			if err != nil {
				t.Errorf("%s : %v", p, err)
				continue
			}
			if !bytes.Contains(body, want) {
				t.Errorf("%s ne décrit plus %s : le régénérer (cd tools/python_tools && python rules_pdf.py ../../cli/rules/%s [--lang EN])",
					p, sys.File, sys.File)
			}
		}
	}
}
