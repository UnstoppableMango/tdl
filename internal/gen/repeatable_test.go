package gen_test

import (
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
