package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

func TestRenderModelFormats(t *testing.T) {
	model := lowerFixture(t, `
package shop

primitive string

entity Order {
  key id: string
  tags: {string}
}
`)

	text, err := renderModel(model, "text")
	if err != nil {
		t.Fatalf("text: %v", err)
	}
	for _, want := range []string{"Model shop", "entity Order", "field key id", "Set<string>"} {
		if !strings.Contains(text, want) {
			t.Errorf("text output missing %q:\n%s", want, text)
		}
	}

	raw, err := renderModel(model, "json")
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("json output does not parse: %v", err)
	}
	if out["package"] != "shop" {
		t.Errorf("json package = %v", out["package"])
	}
}

func TestRenderModelUnknownFormat(t *testing.T) {
	model := lowerFixture(t, `primitive string`)
	if _, err := renderModel(model, "yaml"); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}

func lowerFixture(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	model, _ := sema.Lower(file)
	return model
}
