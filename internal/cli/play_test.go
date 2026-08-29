package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolveViewsCanonicalOrder(t *testing.T) {
	got, err := resolveViews([]string{"stats", "fmt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []string{"fmt", "stats"}; !slices.Equal(got, want) {
		t.Errorf("resolveViews = %v, want %v", got, want)
	}
}

func TestResolveViewsAll(t *testing.T) {
	got, err := resolveViews([]string{"all"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(got, playViews) {
		t.Errorf("resolveViews(all) = %v, want %v", got, playViews)
	}
}

func TestResolveViewsRejectsUnknown(t *testing.T) {
	if _, err := resolveViews([]string{"nope"}); err == nil {
		t.Fatal("expected an error for an unknown view")
	}
}

func TestSeedScratchCreatesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scratch.tdl")
	if err := seedScratch(path, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != scratchTemplate {
		t.Error("seeded file does not match the template")
	}
}

func TestSeedScratchRejectsMissingExplicitFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.tdl")
	if err := seedScratch(path, true); err == nil {
		t.Fatal("expected an error for an explicitly named missing file")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("an explicitly named file should not be created")
	}
}

func TestRenderReportsErrorsWithCaret(t *testing.T) {
	var out bytes.Buffer
	render(&out, "bad.tdl", "type Bad {\n  id string\n}\n", []string{"fmt"})

	got := out.String()
	if !strings.Contains(got, "2:6: expected :, got IDENT") {
		t.Errorf("missing error line in output:\n%s", got)
	}
	if !strings.Contains(got, "\n       ^\n") {
		t.Errorf("missing caret under column 6:\n%s", got)
	}
}

func TestRenderStatsCountsShape(t *testing.T) {
	var out bytes.Buffer
	render(&out, "t.tdl", scratchTemplate, []string{"stats"})

	for _, want := range []string{"types          1", "fields         5", "optional       1", "variants       2"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("stats missing %q:\n%s", want, out.String())
		}
	}
}
