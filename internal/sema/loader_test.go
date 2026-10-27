package sema

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/parser"
)

// MapLoader resolves imports from an in-memory tree, keyed by the path as
// written.
type MapLoader map[string]string

func (m MapLoader) Load(_, path string) (string, string, error) {
	src, ok := m[path]
	if !ok {
		return path, "", os.ErrNotExist
	}
	return path, src, nil
}

// A recording loader keeps every file lowering read, transitive imports
// included, once each.
func TestRecordingKeepsEveryFileRead(t *testing.T) {
	loader := &Recording{Loader: MapLoader{
		"b.tdl": "package b\n\nimport \"c.tdl\" as c\nimport \"d.tdl\" as d\n",
		"c.tdl": "package c\n\nimport \"d.tdl\" as d\n",
		"d.tdl": "package d\n",
	}}
	file, err := parser.Parse("a.tdl", strings.NewReader("package a\n\nimport \"b.tdl\" as b\n"))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if _, diags := Lower(file, WithLoader(loader)); len(diags) > 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	want := []string{"b.tdl", "c.tdl", "d.tdl"}
	if got := loader.Names(); !slices.Equal(got, want) {
		t.Errorf("Names() = %v, want %v", got, want)
	}
}

// A file that could not be read is not one lowering read.
func TestRecordingSkipsAFailedLoad(t *testing.T) {
	loader := &Recording{Loader: MapLoader{}}
	if _, _, err := loader.Load("a.tdl", "missing.tdl"); err == nil {
		t.Fatal("expected an error loading a missing file")
	}
	if got := loader.Names(); len(got) != 0 {
		t.Errorf("Names() = %v, want none", got)
	}
}
