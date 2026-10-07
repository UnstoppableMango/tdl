package ir

import (
	"strings"
	"testing"
)

func TestDumpPackageDoc(t *testing.T) {
	got := Dump(&Model{Package: "p", Doc: []string{"first", "second"}})

	if want := "Model p\n├── Doc (2 lines)\n"; !strings.HasPrefix(got, want) {
		t.Errorf("dump does not start with %q:\n%s", want, got)
	}
}

func TestDumpWithoutPackageDoc(t *testing.T) {
	if got := Dump(&Model{Package: "p"}); strings.Contains(got, "Doc") {
		t.Errorf("dump of an undocumented package has a Doc line:\n%s", got)
	}
}
