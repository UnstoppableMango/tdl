package unlower_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
)

// TestCorpusRoundTrips unlowers every conformance case, prints it, and
// checks the printed file lowers to the same model, positions aside, and
// is already canonical.
func TestCorpusRoundTrips(t *testing.T) {
	dirs, err := filepath.Glob("../../testdata/conformance/*")
	if err != nil || len(dirs) == 0 {
		t.Fatalf("no corpus found: %v", err)
	}

	for _, dir := range dirs {
		t.Run(filepath.Base(dir), func(t *testing.T) {
			loader := caseLoader(t, dir)
			want := lower(t, loader, read(t, filepath.Join(dir, "source.tdl")))

			printed := ast.Fprint(unlower.File(want))
			got := lower(t, loader, printed)

			if a, b := ir.WithoutPositions(want), ir.WithoutPositions(got); !proto.Equal(a, b) {
				t.Errorf("the printed file lowers to a different model\n--- printed ---\n%s--- got ---\n%s--- want ---\n%s",
					printed, ir.Dump(b), ir.Dump(a))
			}

			file, err := parser.Parse("source.tdl", strings.NewReader(printed))
			if err != nil {
				t.Fatalf("parsing the printed file: %v", err)
			}
			if again := ast.Fprint(file); again != printed {
				t.Errorf("the printed file is not canonical\n--- printed ---\n%s--- formatted ---\n%s", printed, again)
			}
		})
	}
}

func lower(t *testing.T, loader mapLoader, src string) *ir.Model {
	t.Helper()

	file, err := parser.Parse("source.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parsing:\n%s\n%v", src, err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(loader))
	for _, d := range diags {
		t.Errorf("lowering:\n%s\n%s", src, d.Error())
	}
	return model
}

// caseLoader serves the other .tdl files in a case directory by their base
// name, as the sema corpus test does.
func caseLoader(t *testing.T, dir string) mapLoader {
	t.Helper()

	paths, err := filepath.Glob(filepath.Join(dir, "*.tdl"))
	if err != nil {
		t.Fatalf("globbing %s: %v", dir, err)
	}
	m := mapLoader{}
	for _, p := range paths {
		if filepath.Base(p) != "source.tdl" {
			m[filepath.Base(p)] = read(t, p)
		}
	}
	return m
}

// mapLoader serves imports from memory by path.
type mapLoader map[string]string

func (m mapLoader) Load(_, path string) (string, string, error) {
	src, ok := m[path]
	if !ok {
		return path, "", os.ErrNotExist
	}
	return path, src, nil
}

func read(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// TestRoundTrips covers what the conformance corpus does not: directives
// reaching variants, variant fields, externs, and class fields; blocks
// sharing a target; overridden class directives; and mixins including
// mixins.
func TestRoundTrips(t *testing.T) {
	loader := mapLoader{"money.tdl": "package money\n\ntype Money {\n  cents: int\n}\n"}
	cases := map[string]string{
		"directives": `package shop

import "money.tdl" as _

class Audited {
  createdAt: instant
}

type Order: Entity, Audited {
  id: string
  createdAt: instant
  total: Money
}

type Line: Audited {
  createdAt: instant
}

enum Payment {
  Card {
    /// The last four digits.
    last4: string
  }
  Cash
}

target go for shop {
  Audited => embed("audit")
  Audited.createdAt => tag("created")
  Order => embed("own")
  Payment.Card => name("CardPayment")
  Payment.Card.last4 => tag("last4")
  Money => foreign("example.com/money.Money")
}

/// A second block for the same target.
target go for shop {
  Order.id => tag("id")
}
`,
		"mixins": `package shop

mixin Stamped {
  createdAt: instant
}

mixin Tracked {
  include Stamped
  by: string
}

type Order {
  include Tracked
  id: string
}
`,
		"sugar": `package shop

type Holder {
  deprecated("use b") a: string? | null
  b: [string]? = []
  c: {string -> {int}} | null
  d: int where { between(-3, 7) } = 4
  e: Option<string | null>
}
`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			want := lower(t, loader, src)
			printed := ast.Fprint(unlower.File(want))
			got := lower(t, loader, printed)
			if a, b := ir.WithoutPositions(want), ir.WithoutPositions(got); !proto.Equal(a, b) {
				t.Errorf("the printed file lowers to a different model\n--- printed ---\n%s--- got ---\n%s--- want ---\n%s",
					printed, ir.Dump(b), ir.Dump(a))
			}
		})
	}
}
