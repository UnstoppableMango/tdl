package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/gen/echo"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

const shopSource = `package shop

/// An order.
type Order {
  id: string
  tags: {string}
}
`

// echoFile writes the model src lowers to as the echo backend's JSON, in a
// directory of its own.
func echoFile(t *testing.T, src string) string {
	t.Helper()
	file, err := parser.Parse("shop.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	return modelFile(t, model)
}

func modelFile(t *testing.T, model *ir.Model) string {
	t.Helper()
	data, err := protojson.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), echo.File)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func echoResolver(string) (plugin.Backend, error) { return echo.Backend{}, nil }

// Importing what echo wrote prints the source it came from, and the loss
// warning names its code.
func TestImportPrintsTDL(t *testing.T) {
	out, errOut, err := run(t, importCmd(echoResolver), "--from", "echo", echoFile(t, shopSource))
	if err != nil {
		t.Fatalf("import: %v\n%s", err, errOut)
	}
	if out != shopSource {
		t.Errorf("printed\n%s\nwant\n%s", out, shopSource)
	}
	if !strings.Contains(errOut, "warning: echo keeps the order JSON lists declarations in (lossy.order)") {
		t.Errorf("stderr = %q, want the lossy.order warning with its code", errOut)
	}
}

// An allowed code prints nothing, whether tdl.toml or the flag allows it.
func TestImportSilencesAllowedCodes(t *testing.T) {
	t.Run("tdl.toml", func(t *testing.T) {
		path := echoFile(t, shopSource)
		manifest := filepath.Join(filepath.Dir(path), config.FileName)
		if err := os.WriteFile(manifest, []byte("[lossy.echo]\nallow = [\"lossy.order\"]\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, errOut, err := run(t, importCmd(echoResolver), "--from", "echo", path)
		if err != nil || errOut != "" {
			t.Errorf("err = %v, stderr = %q, want neither", err, errOut)
		}
	})

	t.Run("flag", func(t *testing.T) {
		_, errOut, err := run(t, importCmd(echoResolver), "--from", "echo", "--allow-lossy", "lossy.doc,lossy.order", echoFile(t, shopSource))
		if err != nil || errOut != "" {
			t.Errorf("err = %v, stderr = %q, want neither", err, errOut)
		}
	})
}

func TestImportWritesAFile(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "imported.tdl")
	out, _, err := run(t, importCmd(echoResolver), "--from", "echo", "--package", "store", "-o", dest, echoFile(t, shopSource))
	if err != nil {
		t.Fatal(err)
	}
	if out != "" {
		t.Errorf("stdout = %q, want nothing with -o", out)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(shopSource, "package shop", "package store", 1); string(data) != want {
		t.Errorf("wrote\n%s\nwant\n%s", data, want)
	}
}

// A model that does not lower is reported with the source it printed, and
// nothing is written.
func TestImportRefusesAModelThatDoesNotLower(t *testing.T) {
	model := &ir.Model{
		Package: "shop",
		Decls: []*ir.Decl{{
			Meta: &ir.Meta{Name: "Order", Position: &ir.Position{Filename: "shop.json"}},
			Node: &ir.Decl_Structure{Structure: &ir.Struct{Fields: []*ir.Field{{
				Meta: &ir.Meta{Name: "total"},
				Type: &ir.ID{Index: -1, Name: "Money"},
			}}}},
		}},
	}
	dest := filepath.Join(t.TempDir(), "imported.tdl")
	_, _, err := run(t, importCmd(echoResolver), "--from", "echo", "-o", dest, modelFile(t, model))
	if err == nil || !strings.Contains(err.Error(), "does not lower") || !strings.Contains(err.Error(), "total: Money") {
		t.Errorf("err = %v, want a lowering failure showing the printed source", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("a model that does not lower was written")
	}
}

func TestImportNeedsAnImporter(t *testing.T) {
	_, _, err := run(t, newImportCmd(), "--from", "debug", "x.json")
	if err == nil || !strings.Contains(err.Error(), "debug does not import") {
		t.Errorf("err = %v, want one saying debug does not import", err)
	}

	_, _, err = run(t, newImportCmd(), "x.json")
	if err == nil || !strings.Contains(err.Error(), "--from") {
		t.Errorf("err = %v, want one asking for --from", err)
	}
}
