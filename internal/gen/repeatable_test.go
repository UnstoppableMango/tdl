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
