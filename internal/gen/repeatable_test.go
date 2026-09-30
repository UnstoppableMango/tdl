package gen_test

import (
	"os"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// twoReserved sets reserved twice at one specificity: both entries are in
// the Widget scope.
const twoReserved = `package p

type Widget { id: string }

target t for p {
  Widget {
    reserved(1)
    reserved(2)
  }
}
`

func lowerClean(t *testing.T, src string) *ir.Model {
	t.Helper()
	file, err := parser.Parse("test.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v; whether a directive may repeat is the backend's to say", diags)
	}
	return model
}

func reservedSpec(repeatable bool) plugin.Description {
	return plugin.Description{
		Name: "t",
		Directives: []*plugin.DirectiveSpec{
			{Name: "reserved", MinArgs: 1, MaxArgs: -1, Repeatable: repeatable},
		},
	}
}

func TestRepeatableDirectiveKeepsEveryEntry(t *testing.T) {
	model := lowerClean(t, twoReserved)

	if problems := gen.CheckDirectives("t", model, reservedSpec(true)); len(problems) != 0 {
		t.Errorf("problems = %v, want none for a repeatable directive", problems)
	}

	widget, _, ok := model.FindDecl("Widget")
	if !ok {
		t.Fatal("no Widget in the model")
	}
	got := plugin.Directives("t", widget.GetDirectives())
	if len(got) != 2 {
		t.Fatalf("Widget directives = %v, want reserved(1) then reserved(2)", got)
	}
	for i, want := range []string{"1", "2"} {
		if got[i].GetName() != "reserved" || len(got[i].GetArgs()) != 1 || got[i].GetArgs()[0].GetText() != want {
			t.Errorf("directive %d = %v, want reserved(%s)", i, got[i], want)
		}
	}
}

func TestUnrepeatableDirectiveTiedIsAnError(t *testing.T) {
	model := lowerClean(t, twoReserved)

	problems := gen.CheckDirectives("t", model, reservedSpec(false))
	if !gen.Fatal(problems) {
		t.Fatalf("problems = %v, want an error", problems)
	}
	const want = "two entries at the same specificity set reserved; one of them has to go"
	found := false
	for _, p := range problems {
		if p.GetSeverity() == plugin.Severity_SEVERITY_ERROR && p.GetMessage() == want {
			found = true
		}
	}
	if !found {
		t.Errorf("problems = %v, want an error %q", problems, want)
	}
}

// twoClassRules reaches Ent through two classes it satisfies at the same
// distance, so both entries tie.
const twoClassRules = `package p

class One { x: string }
class Two { y: string }
type Ent: Entity, One, Two { id: string x: string y: string }

target t for p {
  One => rule("a")
  Two => rule("b")
}
`

func hasProblem(problems []*plugin.Diagnostic, severity plugin.Severity, message string) bool {
	for _, p := range problems {
		if p.GetSeverity() == severity && p.GetMessage() == message {
			return true
		}
	}
	return false
}

func TestUnrepeatableDirectiveTiedThroughClassesIsAnError(t *testing.T) {
	model := lowerClean(t, twoClassRules)
	desc := plugin.Description{
		Name:       "t",
		Directives: []*plugin.DirectiveSpec{{Name: "rule", MinArgs: 1, MaxArgs: 1}},
	}

	problems := gen.CheckDirectives("t", model, desc)
	const want = "two entries at the same specificity set rule; one of them has to go"
	if !hasProblem(problems, plugin.Severity_SEVERITY_ERROR, want) {
		t.Errorf("problems = %v, want an error %q", problems, want)
	}
}

// A directive the backend does not declare cannot be repeatable, so a tie
// on it is an error on top of the undeclared warning.
func TestUndeclaredDirectiveTiedIsAnError(t *testing.T) {
	model := lowerClean(t, twoReserved)

	problems := gen.CheckDirectives("t", model, plugin.Description{Name: "t"})
	const tie = "two entries at the same specificity set reserved; one of them has to go"
	if !hasProblem(problems, plugin.Severity_SEVERITY_ERROR, tie) {
		t.Errorf("problems = %v, want an error %q", problems, tie)
	}
	const undeclared = "t does not declare the directive reserved"
	if !hasProblem(problems, plugin.Severity_SEVERITY_WARNING, undeclared) {
		t.Errorf("problems = %v, want a warning %q", problems, undeclared)
	}
}

// depLoader serves the one dependency twoExternRules imports.
type depLoader struct{}

func (depLoader) Load(_, path string) (string, string, error) {
	if path != "dep.tdl" {
		return path, "", os.ErrNotExist
	}
	return path, "package dep\ntype Money { units: int }\n", nil
}

// twoExternRules sets rule twice on a declaration a `_` import merged in,
// both at the specificity of a path naming it.
const twoExternRules = `package p

import "dep.tdl" as _

type Price { amount: Money }

target t for p {
  Money => rule("a")
  Money => rule("b")
}
`

// An extern's directives follow the same rule as a declaration's: lowering
// keeps both tied entries, and the backend's spec decides whether a tie is
// an error.
func TestTiedDirectiveOnAnExtern(t *testing.T) {
	file, err := parser.Parse("test.tdl", strings.NewReader(twoExternRules))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	model, diags := sema.Lower(file, sema.WithLoader(depLoader{}))
	if len(diags) > 0 {
		t.Fatalf("lowering reported %v; whether a directive may repeat is the backend's to say", diags)
	}
	if n := len(model.GetExterns()); n != 1 {
		t.Fatalf("externs = %d, want 1: %v", n, model.GetExterns())
	}
	if got := plugin.Directives("t", model.GetExterns()[0].GetDirectives()); len(got) != 2 {
		t.Fatalf("Money directives = %v, want rule(a) then rule(b)", got)
	}

	spec := func(repeatable bool) plugin.Description {
		return plugin.Description{
			Name:       "t",
			Directives: []*plugin.DirectiveSpec{{Name: "rule", MinArgs: 1, MaxArgs: 1, Repeatable: repeatable}},
		}
	}
	if problems := gen.CheckDirectives("t", model, spec(true)); len(problems) != 0 {
		t.Errorf("problems = %v, want none for a repeatable directive", problems)
	}
	const tie = "two entries at the same specificity set rule; one of them has to go"
	if problems := gen.CheckDirectives("t", model, spec(false)); !hasProblem(problems, plugin.Severity_SEVERITY_ERROR, tie) {
		t.Errorf("problems = %v, want an error %q", problems, tie)
	}
}
