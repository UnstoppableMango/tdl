package graphql_test

import (
	"context"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/ast"
	"github.com/unstoppablemango/tdl/backend/graphql"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/internal/unlower"
	"github.com/unstoppablemango/tdl/plugin"
)

// importGraphQL imports files named shop.graphql, and returns the TDL
// printed from the model, with the response's diagnostics.
func importGraphQL(t *testing.T, files ...string) (string, []*plugin.Diagnostic) {
	t.Helper()
	return importNamed(t, "shop.graphql", files...)
}

func importNamed(t *testing.T, path string, files ...string) (string, []*plugin.Diagnostic) {
	t.Helper()
	req := &plugin.ImportRequest{Target: graphql.Name}
	for _, src := range files {
		req.Files = append(req.Files, &plugin.File{Path: path, Content: []byte(src)})
	}
	resp, err := graphql.Backend{}.Import(context.Background(), req)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if resp.GetModel() == nil {
		return "", resp.GetDiagnostics()
	}
	return ast.Fprint(unlower.File(resp.GetModel())), resp.GetDiagnostics()
}

func failed(t *testing.T, diags []*plugin.Diagnostic, want string) {
	t.Helper()
	if len(diags) != 1 || diags[0].GetSeverity() != plugin.Severity_SEVERITY_ERROR || !strings.Contains(diags[0].GetMessage(), want) {
		t.Errorf("diagnostics = %v, want one error containing %q", diags, want)
	}
}

func TestImportRefusesWhatItCannotRead(t *testing.T) {
	_, diags := importGraphQL(t, "type A { a: Int }", "type B { b: Int }")
	failed(t, diags, "one .graphql file")

	_, diags = importGraphQL(t, "type A {")
	failed(t, diags, "does not parse")

	_, diags = importGraphQL(t, "type A { b: B }")
	failed(t, diags, "not a valid schema")

	_, diags = importGraphQL(t, `directive @tdl(conforms: String) on OBJECT
type A @tdl(conforms: "string where") { a: Int }`)
	failed(t, diags, "not a TDL conformance list")
}

// What a handwritten schema holds that TDL has no form for, or that
// regenerates differently, warns with its loss code and is read as near
// as TDL comes.
func TestImportWarnsWhatItLoses(t *testing.T) {
	src, diags := importGraphQL(t, `directive @key(fields: String) on OBJECT

schema {
  query: Query
}

scalar JSON

"""
Not what the generator writes.
"""
scalar Long

interface Node {
  id: ID!
}

input Filter {
  text: String
}

type Query {
  item(id: ID!): Item
  search(filter: Filter): [Item!]!
}

type Item implements Node @key(fields: "id") {
  id: ID!
  count: Long!
  data: JSON
  node: Node
}

extend type Item {
  extra: String
}

type Card {
  last4: String!
}

union Payment = Card

type Wallet {
  card: Card
  payment: Payment
}
`)
	want := map[string]int{
		// @key, the schema's root types, JSON, Node, Filter, Query.item's
		// and search's arguments, Item's interface and directive,
		// Item.data, Item.node, the extension, Payment, and Wallet.payment
		emit.LossUnsupported: 14,
		emit.LossPrimitive:   1, // Item.id
		emit.LossDoc:         1, // Long's description
	}
	got := map[string]int{}
	for _, d := range diags {
		got[d.GetCode()]++
	}
	for code, n := range want {
		if got[code] != n {
			t.Errorf("%d %s warnings, want %d: %v", got[code], code, n, diags)
		}
	}
	for _, s := range []string{
		"type Query {\n  item: Item?\n  search: [Item]\n}",
		"type Item {\n  id: string\n  count: int64\n}",
		"type Card {",
		"type Wallet {\n  card: Card?\n}",
	} {
		if !strings.Contains(src, s) {
			t.Errorf("import lacks %q:\n%s", s, src)
		}
	}
	if strings.Contains(src, "enum Payment") {
		t.Errorf("a union of a type something else names is read as an enum:\n%s", src)
	}
}

// The file's name is the package, unless it is the name a model without a
// package is written to, or no TDL package.
func TestImportReadsThePackageFromTheFileName(t *testing.T) {
	for path, want := range map[string]string{
		"shop.graphql":          "package shop\n",
		"api/orders.graphql":    "package orders\n",
		"schema.graphql":        "",
		"shop-v1.graphql":       "",
		"dir/schema.graphqls":   "",
		"dir/billing.graphqls":  "package billing\n",
		"dir/shop.v1.graphql":   "",
		"dir/Shop_V1.graphql":   "package Shop_V1\n",
		"dir/no_extension_file": "package no_extension_file\n",
	} {
		src, diags := importNamed(t, path, "type A { a: Int }")
		if got := strings.HasPrefix(src, "package"); got != (want != "") || (want != "" && !strings.HasPrefix(src, want)) {
			t.Errorf("%s imports as\n%s\nwant %q", path, src, want)
		}
		named := want != "" || strings.HasSuffix(path, "schema.graphql") || strings.HasSuffix(path, "schema.graphqls")
		if lost := len(diags) > 0 && diags[0].GetCode() == emit.LossName; lost == named {
			t.Errorf("%s: diagnostics = %v", path, diags)
		}
	}
}

// An annotation's value survives every character a string literal
// escapes.
func TestAnnotationsSurviveEscapes(t *testing.T) {
	for _, reason := range []string{`a " b`, `ends in \`, `\\"`, "two\nlines", "tab\there", "café"} {
		src := `directive @tdl(doc: String, reason: String) repeatable on OBJECT
type A @tdl(doc: "") @tdl(reason: ` + graphql.Literal(reason) + `) { a: Int }
`
		got, diags := importGraphQL(t, src)
		if len(diags) > 0 {
			t.Fatalf("%q: %v", reason, diags)
		}
		escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\t", `\t`).Replace(reason)
		if want := `deprecated("` + escaped + `")`; !strings.Contains(got, want) {
			t.Errorf("%q imports as\n%s\nwant %s", reason, got, want)
		}
	}
}
