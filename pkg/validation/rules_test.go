package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// onboardingRulesDir holds the cross-platform test vectors written with the
// product spec (docs/product/onboarding/rules/*.json). Every platform runs the
// same files, so the backend reads them as-is rather than copying the cases.
var onboardingRulesDir = filepath.Join("..", "..", "docs", "product", "onboarding", "rules")

// loadRuleCases decodes the "cases" array of a rules file into out.
func loadRuleCases(t *testing.T, file string, out any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(onboardingRulesDir, file))
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	var doc struct {
		Cases json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}
	if err := json.Unmarshal(doc.Cases, out); err != nil {
		t.Fatalf("decode %s cases: %v", file, err)
	}
}
