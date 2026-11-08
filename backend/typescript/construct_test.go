package typescript_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/plugin"
)

const constructSource = `package shop

type Secret: string
type Email: string
type WorkEmail: Email
type Grade: string where { oneOf("a", "b") }
type Level: int where { min(1) max(5) }
type Code: string where { length(2..4) }
type Age: int where { min(0) max(150) }
type Ratio: float64 where { min(0) }
type Tags: [string] where { length(..3) unique }
type Pin: string where { matches(/^[0-9]+$/) }
type Maybe: string | null where { length(1..) }
type Login { user: string }

target typescript for shop {
  construct
}
`

// With construct on the block every newtype has a constructor named for
// it, checking only what its type cannot state.
func TestConstructOnTheBlock(t *testing.T) {
	resp := generateRaw(t, lowerSource(t, "shop.tdl", constructSource, nil))
	src := check(t, resp)
	contains(t, src,
		"export type Secret = string;",
		"export const Secret = (v: string): Secret => v as Secret;",
		"export const WorkEmail = (v: Email): WorkEmail => v as WorkEmail;",
		`export const Grade = (v: "a" | "b"): Grade => v as Grade;`,
		"export const Level = (v: 1 | 2 | 3 | 4 | 5): Level => v as Level;",
		"export const Code = (v: string): Code => {",
		`throw new RangeError("Code: length(2..4)");`,
		`throw new RangeError("Age: min(0)");`,
		`throw new RangeError("Age: max(150)");`,
		`throw new RangeError("Ratio: min(0)");`,
		`throw new RangeError("Tags: length(..3)");`,
		`throw new RangeError("Tags: unique");`,
		`throw new RangeError("Maybe: length(1..)");`,
		"return v as Code;",
	)
	if strings.Contains(src, "export const Login") {
		t.Errorf("a struct has a constructor:\n%s", src)
	}
	if strings.Contains(src, "Pin: matches") {
		t.Errorf("matches is checked without patterns:\n%s", src)
	}

	// Every checked constraint stops warning; matches keeps its warning.
	var pin bool
	for _, d := range resp.GetDiagnostics() {
		if d.GetCode() != emit.LossConstraint {
			continue
		}
		if strings.Contains(d.GetMessage(), "Pin") {
			pin = true
			continue
		}
		t.Errorf("unexpected constraint warning: %s", d.GetMessage())
	}
	if !pin {
		t.Errorf("diagnostics = %v, want Pin's matches to warn", resp.GetDiagnostics())
	}

	runs(t, src, map[string]bool{
		`Code("ab")`:                 true,
		`Code("a")`:                  false,
		`Code("ééé")`:                true,
		`Code("abcde")`:              false,
		`Age(0)`:                     true,
		`Age(-1)`:                    false,
		`Age(151)`:                   false,
		`Ratio(0.5)`:                 true,
		`Ratio(-0.5)`:                false,
		`Tags(["a", "b"])`:           true,
		`Tags(["a", "a"])`:           false,
		`Tags(["a", "b", "c", "d"])`: false,
		`Maybe(null)`:                true,
		`Maybe("x")`:                 true,
		`Maybe("")`:                  false,
		`Secret("hunter2")`:          true,
		`Pin("abc")`:                 true,
	})
}

// A newtype over a branded newtype takes it, which only its constructor
// makes, so it does not check again what it inherits.
func TestConstructTrustsABrandedBase(t *testing.T) {
	src := check(t, generateModel(t, lowerSource(t, "shop.tdl", `package shop

type Email: string where { length(3..) }
type WorkEmail: Email where { length(..64) }

target typescript for shop {
  brand
}
`, nil)))
	contains(t, src, `throw new RangeError("Email: length(3..)");`, `throw new RangeError("WorkEmail: length(..64)");`)
	if strings.Count(src, "length(3..)") != 1 {
		t.Errorf("WorkEmail checks Email's length again:\n%s", src)
	}
}

// construct on a newtype covers that one alone.
func TestConstructOnANewtype(t *testing.T) {
	src := check(t, generateModel(t, lowerSource(t, "shop.tdl", `package shop

type Secret: string
type Username: string

target typescript for shop {
  Secret => construct
}
`, nil)))
	contains(t, src, "export const Secret = (v: string): Secret => v as Secret;")
	if strings.Contains(src, "export const Username") {
		t.Errorf("Username has a constructor:\n%s", src)
	}
}

// brand turns construct on, on the block and on a newtype, and the
// constructor casts to the brand.
func TestBrandConstructs(t *testing.T) {
	src := check(t, generateModel(t, lowerSource(t, "shop.tdl", brandSource, nil)))
	contains(t, src,
		"export const Secret = (v: string): Secret => v as Secret;",
		"export const WorkEmail = (v: Email): WorkEmail => v as WorkEmail;",
		`export const Grade = (v: "a" | "b"): Grade => v as Grade;`,
		"export const Maybe = (v: string | null): Maybe => v as Maybe;",
	)
	compiles(t, src, `export const s: Secret = Secret("hunter2"); export const w: WorkEmail = WorkEmail(Email("a@b"));`, true)
	compiles(t, src, `export const u: Username = Secret("hunter2");`, false)

	one := check(t, generateModel(t, lowerSource(t, "shop.tdl", `package shop

type Secret: string
type Username: string

target typescript for shop {
  Secret => brand
}
`, nil)))
	contains(t, one, "export const Secret = (v: string): Secret => v as Secret;")
	if strings.Contains(one, "export const Username") {
		t.Errorf("Username has a constructor:\n%s", one)
	}
}

// Under patterns, matches is checked with the pattern ECMAScript reads as
// RE2 does; a shape that backtracks keeps warning.
func TestConstructChecksPatterns(t *testing.T) {
	resp := generateRaw(t, lowerSource(t, "shop.tdl", `package shop

type Pin: string where { matches(/^[0-9]+$/) }
type Slow: string where { matches(/^(a+)+$/) }

target typescript for shop {
  construct
  patterns
}
`, nil))
	src := check(t, resp)
	contains(t, src, `throw new RangeError("Pin: matches(/^[0-9]+$/)");`)
	if strings.Contains(src, `"Slow: matches`) {
		t.Errorf("a backtracking pattern is checked:\n%s", src)
	}
	var slow bool
	for _, d := range resp.GetDiagnostics() {
		if d.GetCode() == emit.LossConstraint && strings.Contains(d.GetMessage(), "Slow") && strings.Contains(d.GetMessage(), "backtracks") {
			slow = true
		} else if d.GetCode() == emit.LossConstraint {
			t.Errorf("unexpected constraint warning: %s", d.GetMessage())
		}
	}
	if !slow {
		t.Errorf("diagnostics = %v, want Slow's pattern to warn", resp.GetDiagnostics())
	}
	runs(t, src, map[string]bool{`Pin("123")`: true, `Pin("12a")`: false, `Slow("aaa")`: true})
}

// construct anywhere but a newtype warns, as brand does, with no loss code.
func TestConstructOnAStructWarns(t *testing.T) {
	resp := generateModel(t, lowerSource(t, "shop.tdl", `package shop

type Login { user: string }

target typescript for shop {
  Login => construct
}
`, nil))
	diags := resp.GetDiagnostics()
	if len(diags) != 1 || diags[0].GetCode() != "" || diags[0].GetSeverity() != plugin.Severity_SEVERITY_WARNING ||
		!strings.Contains(diags[0].GetMessage(), "Login is not a newtype, so its construct directive") {
		t.Errorf("diagnostics = %v", diags)
	}
}

// Under readback, a checked constraint does not read back, and the
// block's construct reads back on each newtype.
func TestConstructUnderReadback(t *testing.T) {
	resp := generateRaw(t, lowerSource(t, "shop.tdl", `package shop

type Code: string where { length(2..4) }

target typescript for shop {
  construct
  readback
}
`, nil))
	var checked, block bool
	for _, d := range resp.GetDiagnostics() {
		switch {
		case d.GetCode() == emit.LossConstraint && strings.Contains(d.GetMessage(), "checked by its constructor"):
			checked = true
		case d.GetCode() == emit.LossName && strings.Contains(d.GetMessage(), "construct"):
			block = true
		}
	}
	if !checked || !block {
		t.Errorf("diagnostics = %v, want read-back loss for the check and the block's construct", resp.GetDiagnostics())
	}
}

// Constructors read back as construct directives, and a branded alias's as
// nothing more than its brand.
func TestImportReadsConstructors(t *testing.T) {
	src, diags := importTS(t, map[string]string{"shop.ts": `
export type Secret = string;
export const Secret = (v: string): Secret => v as Secret;
export type Token = string & { readonly __brand: { readonly "shop.Token": true } };
export const Token = (v: string): Token => v as Token;
export type Code = string;
export const Code = (v: string): Code => {
  if ([...v].length < 2) {
    throw new RangeError("Code: length(2..)");
  }
  return v as Code;
};
export type Plain = string;
export type Tier = "free" | "paid";
export const Tier = (v: "free" | "paid"): Tier => v as Tier;
`})
	for _, s := range []string{"Secret => construct", "Code => construct", "Token => brand", "Tier => construct", `type Tier: string where { oneOf("free", "paid") }`} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
	for _, s := range []string{"Token => construct", "Plain => construct"} {
		if strings.Contains(src, s) {
			t.Errorf("import writes %q:\n%s", s, src)
		}
	}
	var code bool
	for _, d := range diags {
		switch {
		case d.GetCode() == emit.LossConstraint && strings.Contains(d.GetMessage(), "Code"):
			code = true
		case strings.Contains(d.GetMessage(), "no TDL form"):
			t.Errorf("a constructor warns: %s", d.GetMessage())
		}
	}
	if !code {
		t.Errorf("diagnostics = %v, want a warning that Code's checks are not read", diags)
	}
}

// runs runs each call against the module under node, and fails unless it
// returns exactly when its value is set and throws a RangeError otherwise.
func runs(t *testing.T, module string, calls map[string]bool) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs node")
	}
	tsc, err := exec.LookPath("tsc")
	if err != nil {
		t.Skip("tsc is not on PATH")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on PATH")
	}
	var b strings.Builder
	b.WriteString(module)
	b.WriteString("\nconst calls: [string, boolean, () => unknown][] = [\n")
	for call, ok := range calls {
		b.WriteString("  [" + quoteJS(call) + ", " + map[bool]string{true: "true", false: "false"}[ok] + ", () => " + call + "],\n")
	}
	b.WriteString(`];
let failed = false;
for (const [call, ok, f] of calls) {
  let threw: unknown = undefined;
  try {
    f();
  } catch (e) {
    threw = e;
  }
  if (ok && threw !== undefined) {
    console.log(call + " throws " + String(threw));
    failed = true;
  } else if (!ok && !(threw instanceof RangeError)) {
    console.log(call + " does not throw a RangeError");
    failed = true;
  }
}
if (failed) {
  throw new Error("constructor checks failed");
}
`)
	dir := t.TempDir()
	path := filepath.Join(dir, "run.ts")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(tsc, "--strict", "--target", "es2022", "--module", "commonjs", "--outDir", dir, path).CombinedOutput(); err != nil {
		t.Fatalf("tsc: %v\n%s\n%s", err, out, b.String())
	}
	if out, err := exec.Command(node, filepath.Join(dir, "run.js")).CombinedOutput(); err != nil {
		t.Errorf("node: %v\n%s", err, out)
	}
}

func quoteJS(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
