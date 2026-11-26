package salesforce_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/salesforce"
	"github.com/unstoppablemango/tdl/internal/analysis"
	"github.com/unstoppablemango/tdl/internal/sema"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/parser"
	"github.com/unstoppablemango/tdl/plugin"
)

// generateTDL generates src, failing on any diagnostic.
func generateTDL(t *testing.T, src string) []*plugin.File {
	t.Helper()
	file, err := parser.Parse("source.tdl", strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	model, diags := sema.Lower(file)
	if len(diags) > 0 {
		t.Fatal(diags)
	}
	b := salesforce.Backend{}
	req, problems := analysis.Request(model, b.Describe(), salesforce.Name, "out")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	resp, err := b.Generate(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.GetDiagnostics() {
		t.Errorf("generating: %s", d.GetMessage())
	}
	return resp.GetFiles()
}

// readBack imports files and prints the model as TDL.
func readBack(t *testing.T, files []*plugin.File) (string, []*plugin.Diagnostic) {
	t.Helper()
	resp, err := salesforce.ReadObjects(files, true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetModel() == nil {
		return "", resp.GetDiagnostics()
	}
	return ast.Fprint(unlower.File(resp.GetModel())), resp.GetDiagnostics()
}

// comesBack generates src and holds what it reads back to src.
func comesBack(t *testing.T, src string) {
	t.Helper()
	got, diags := readBack(t, generateTDL(t, src))
	for _, d := range diags {
		if !strings.Contains(d.GetMessage(), "Apex classes are not read yet") {
			t.Errorf("import warns: %s", d.GetMessage())
		}
	}
	if got != src {
		t.Errorf("reads back as\n%s\nwant\n%s", got, src)
	}
}

func hasDiag(diags []*plugin.Diagnostic, code, text string) bool {
	for _, d := range diags {
		if (code == "" || d.GetCode() == code) && strings.Contains(d.GetMessage(), text) {
			return true
		}
	}
	return false
}

// object is an object file as generating writes one for Thing, with extra
// elements.
func object(name, extra string) *plugin.File {
	return &plugin.File{Path: "objects/" + name + "/" + name + ".object-meta.xml", Content: []byte(`<CustomObject>
<deploymentStatus>Deployed</deploymentStatus>
<label>Thing</label>
<nameField><displayFormat>Thing-{0000}</displayFormat><label>Thing Number</label><type>AutoNumber</type></nameField>
<pluralLabel>Things</pluralLabel>
<sharingModel>ReadWrite</sharingModel>
` + extra + `</CustomObject>`)}
}

// field is a field file holding body.
func field(object, name, body string) *plugin.File {
	return &plugin.File{Path: "objects/" + object + "/fields/" + name + ".field-meta.xml",
		Content: []byte("<CustomField><fullName>" + name + "</fullName>" + body + "</CustomField>")}
}

// What generating writes for an object reads back as the source.
func TestObjectsReadBack(t *testing.T) {
	comesBack(t, `package salesforce

/// A customer.
///
/// It has two paragraphs.
type Customer: Entity { }

deprecated("use Customer")
type Client: Entity { }

type LineItem: Entity { }

target salesforce for salesforce {
  LineItem => name("Line_Item")
  LineItem => label("Line")
  LineItem => plural("Line items")
}
`)
}

// What has no TDL form warns under its code and is not read.
func TestObjectsWarn(t *testing.T) {
	files := []*plugin.File{
		object("Thing__c", "<enableHistory>true</enableHistory>\n<validationRules><fullName>R</fullName></validationRules>\n"),
		field("Account", "Tier__c", "<label>Tier</label><type>Text</type>"),
		field("acme__Thing__c", "Tier__c", "<label>Tier</label><type>Text</type>"),
		field("Lost__c", "Name__c", "<label>Name</label><type>Text</type>"),
		{Path: "objects/Thing__c/validationRules/Positive.validationRule-meta.xml", Content: []byte("<ValidationRule/>")},
		{Path: "objects/Thing__c/listViews/All.listView-meta.xml", Content: []byte("<ListView/>")},
		{Path: "layouts/Thing__c-Layout.layout-meta.xml", Content: []byte("<Layout/>")},
		{Path: "classes/Thing.cls", Content: []byte("public class Thing {}")},
	}
	got, diags := readBack(t, files)
	for _, want := range []struct{ code, text string }{
		{"lossy.constraint", "Thing__c has a validation rule, which"},
		{"lossy.constraint", "Positive.validationRule-meta.xml"},
		{"lossy.unsupported", "enableHistory, listViews"},
		{"lossy.unsupported", "Account is a standard name"},
		{"lossy.unsupported", "managed package"},
		{"lossy.unsupported", "no object file"},
		{"lossy.unsupported", "is not an object or a field"},
		{"", "Apex classes are not read yet"},
	} {
		if !hasDiag(diags, want.code, want.text) {
			t.Errorf("no %q warning containing %q in %v", want.code, want.text, diags)
		}
	}
	if len(diags) != 8 {
		t.Errorf("%d warnings, want 8: %v", len(diags), diags)
	}
	if want := "package salesforce\n\ntype Thing: Entity { }\n"; got != want {
		t.Errorf("reads as\n%s\nwant\n%s", got, want)
	}
}

// An object file that is not XML fails the import.
func TestObjectsRefuseBrokenXML(t *testing.T) {
	_, diags := readBack(t, []*plugin.File{{Path: "objects/A__c/A__c.object-meta.xml", Content: []byte("<CustomObject>")}})
	if len(diags) != 1 || diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), "is not object metadata") {
		t.Errorf("diagnostics = %v, want one error", diags)
	}
}
