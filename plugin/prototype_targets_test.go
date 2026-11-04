package plugin

// PROTOTYPE for #1398. Run: go test ./plugin -run TestPrototypeChoose -v

import (
	"context"
	"testing"
)

type stub string

func (s stub) Describe() Description { return Description{Name: string(s), Version: "0.0.0"} }
func (s stub) Generate(context.Context, *Request) (*Response, error) {
	return &Response{}, nil
}

func TestPrototypeChoose(t *testing.T) {
	dotnet := []Backend{stub("csharp"), stub("fsharp")}
	cases := []struct {
		name, requested, argv0 string
		bs                     []Backend
		want                   string // backend name, or the refusal
	}{
		{"new host names a target", "fsharp", "/nix/store/x/bin/tdl-gen-csharp", dotnet, "fsharp"},
		{"old host, symlinked name", "", "/usr/bin/tdl-gen-fsharp", dotnet, "fsharp"},
		{"old host, shared binary's own name", "", "/usr/bin/tdl-dotnet", dotnet, "serves csharp, fsharp; asked for tdl-dotnet"},
		{"new host, unserved target", "vb", "tdl-gen-vb", dotnet, "serves csharp, fsharp; asked for vb"},
		{"single backend, old host", "", "tdl-gen-whatever", []Backend{stub("sql")}, "sql"},
		{"single backend, other target named", "postgres", "tdl-gen-postgres", []Backend{stub("sql")}, "serves sql; asked for postgres"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, refusal, ok := choose(c.requested, c.argv0, c.bs)
			got := refusal
			if ok {
				got = b.Describe().Name
			}
			t.Logf("requested=%q argv0=%q -> %s", c.requested, c.argv0, got)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}
