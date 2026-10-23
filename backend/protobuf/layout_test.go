package protobuf

import "testing"

// The expectations are what buf format v1.47.2 writes for each input.
func TestReflowMatchesBufFormat(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`(mo) = {}`, `(mo) = {}`},
		{`(mo) = { a: 1 }`, `(mo) = {a: 1}`},
		{`(mo) = { a: -1.5e3 }`, `(mo) = {a: -1.5e3}`},
		{`(mo) = { a: FOO_BAR }`, `(mo) = {a: FOO_BAR}`},
		{`(mo) = { b: 'it\'s' }`, `(mo) = {b: 'it\'s'}`},
		{`(mo) = { a: 1, b: "x"; }`, "(mo) = {\n  a: 1\n  b: \"x\"\n}"},
		{
			`(google.api.resource) = { type: "a/B" pattern: "bs/{b}" singular: "b" plural: "bs" }`,
			"(google.api.resource) = {\n  type: \"a/B\"\n  pattern: \"bs/{b}\"\n  singular: \"b\"\n  plural: \"bs\"\n}",
		},
		{`(mo) = { n { x: 2 } }`, "(mo) = {\n  n: {x: 2}\n}"},
		{`(mo) = { n {} }`, "(mo) = {\n  n: {}\n}"},
		{`(mo) = { n { x: 2 y: 3 } }`, "(mo) = {\n  n: {\n    x: 2\n    y: 3\n  }\n}"},
		{`(mo) = { b: [] }`, "(mo) = {\n  b: []\n}"},
		{`(mo) = { b: ["x"] }`, "(mo) = {\n  b: [\"x\"]\n}"},
		{`(mo) = { b: ["x", "y"] }`, "(mo) = {\n  b: [\n    \"x\",\n    \"y\"\n  ]\n}"},
		{`(mo) = { ns: [{x: 1}] }`, "(mo) = {\n  ns: [\n    {x: 1}]\n}"},
		{`(mo) = { ns: [{x: 1 y: 2}] }`, "(mo) = {\n  ns: [\n    {\n      x: 1\n      y: 2\n    }\n  ]\n}"},
		{`(mo) = { ns: [{x: 1}, {x: 2 y: 3}] }`, "(mo) = {\n  ns: [\n    {x: 1},\n    {\n      x: 2\n      y: 3\n    }\n  ]\n}"},
		// Values reflow cannot read are kept as given.
		{`(mo) = { b: "x" "y" }`, `(mo) = { b: "x" "y" }`},
		{`(mo) = { a: 1 # c }`, `(mo) = { a: 1 # c }`},
		{`(mo) = < a: 1 >`, `(mo) = < a: 1 >`},
		{`deprecated = true`, `deprecated = true`},
	} {
		if got := reflow(tc.in, ""); got != tc.want {
			t.Errorf("reflow(%q) =\n%s\nwant\n%s", tc.in, got, tc.want)
		}
	}
}

func TestBracketsMatchBufFormat(t *testing.T) {
	for _, tc := range []struct {
		opts []string
		want string
	}{
		{nil, ""},
		{[]string{"(fi) = 1"}, " [(fi) = 1]"},
		{[]string{"(fo) = { a: 1 }"}, " [(fo) = {a: 1}]"},
		{[]string{"(fo) = {}"}, " [(fo) = {}]"},
		{[]string{`(fo) = { a: 1 b: "x" }`}, " [(fo) = {\n    a: 1\n    b: \"x\"\n  }]"},
		{[]string{"(fi) = 1", "deprecated = true"}, " [\n    (fi) = 1,\n    deprecated = true\n  ]"},
		{
			[]string{"(fo).n = { x: 1 y: 2 }", "(fo).a = 1"},
			" [\n    (fo).n = {\n      x: 1\n      y: 2\n    },\n    (fo).a = 1\n  ]",
		},
	} {
		if got := brackets("  ", tc.opts); got != tc.want {
			t.Errorf("brackets(%q) =\n%s\nwant\n%s", tc.opts, got, tc.want)
		}
	}
}

func TestCollapseEmpty(t *testing.T) {
	in := "message A {\n  message B {\n  }\n  oneof o {\n  }\n}\nservice S {\n}\nmessage C {\n  int32 x = 1;\n}\n"
	want := "message A {\n  message B {}\n  oneof o {}\n}\nservice S {}\nmessage C {\n  int32 x = 1;\n}\n"
	if got := collapseEmpty(in); got != want {
		t.Errorf("collapseEmpty =\n%s\nwant\n%s", got, want)
	}
}
