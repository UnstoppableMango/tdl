package emit_test

import (
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/plugin"
)

// The loss codes are the ones docs/design/reverse.md tabulates, in its
// order.
func TestLossCodesMatchTheDesign(t *testing.T) {
	doc, err := os.ReadFile("../../../docs/design/reverse.md")
	if err != nil {
		t.Fatal(err)
	}
	var documented []string
	for _, m := range regexp.MustCompile("(?m)^\\| `(lossy\\.[a-z-]+)` \\|").FindAllSubmatch(doc, -1) {
		documented = append(documented, string(m[1]))
	}
	if !slices.Equal(documented, emit.LossCodes) {
		t.Errorf("reverse.md lists %v\nemit.LossCodes is %v", documented, emit.LossCodes)
	}
}

func TestLossyCarriesItsCode(t *testing.T) {
	s := &emit.Session{}
	s.Lossy(emit.LossCollection, nil, "%s is a set, written as a list", "tags")

	if len(s.Diags) != 1 {
		t.Fatalf("diagnostics = %v", s.Diags)
	}
	d := s.Diags[0]
	if d.GetCode() != "lossy.collection" || d.GetSeverity() != plugin.Severity_SEVERITY_WARNING || d.GetMessage() != "tags is a set, written as a list" {
		t.Errorf("diagnostic = %v", d)
	}
}
