package acceptance

import (
	"strings"
	"testing"
)

func TestEnglishAssertionsRequireBusinessBehavior(t *testing.T) {
	for _, assertion := range []string{"element exists", "component renders successfully", "page opens"} {
		contract := submissionEvidenceContract()
		contract.RequiredTests[0].Assertions = []string{assertion}
		if err := Validate(contract); err == nil || !strings.Contains(err.Error(), "weak acceptance assertion") {
			t.Fatalf("weak English assertion %q was not rejected: %v", assertion, err)
		}
	}
}
