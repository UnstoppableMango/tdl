package plugin

import (
	"context"
	"testing"
)

// named is a backend that is only a name.
type named string

func (n named) Describe() Description { return Description{Name: string(n)} }

func (named) Generate(context.Context, *Request) (*Response, error) { return &Response{}, nil }

func TestChoose(t *testing.T) {
	pair := []Backend{named("fsharp"), named("csharp")}
	tests := []struct {
		name       string
		bs         []Backend
		target     string
		executable string
		want       string
		refusal    string
	}{
		{name: "the handshake's target", bs: pair, target: "fsharp", executable: "/bin/tdl-gen-csharp", want: "fsharp"},
		{name: "the executable's name, with no target", bs: pair, executable: "/bin/tdl-gen-csharp", want: "csharp"},
		{name: "the shared executable's own name", bs: pair, executable: "/bin/tdl-gen-dotnet", refusal: "serves csharp, fsharp; asked for dotnet"},
		{name: "an unserved target", bs: pair, target: "vb", executable: "/bin/tdl-gen-csharp", refusal: "serves csharp, fsharp; asked for vb"},
		{name: "one backend, no target", bs: pair[:1], executable: "/bin/main", want: "fsharp"},
		{name: "one backend, another target", bs: pair[:1], target: "csharp", executable: "/bin/tdl-gen-fsharp", refusal: "serves fsharp; asked for csharp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, refusal, ok := choose(tt.bs, tt.target, tt.executable)
			if tt.refusal != "" {
				if ok || refusal != tt.refusal {
					t.Errorf("choose = %v, %q; want refusal %q", b, refusal, tt.refusal)
				}
				return
			}
			if !ok || b.Describe().Name != tt.want {
				t.Errorf("choose = %v, %q; want %s", b, refusal, tt.want)
			}
		})
	}
}
