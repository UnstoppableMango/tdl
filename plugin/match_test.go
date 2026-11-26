package plugin_test

import (
	"testing"

	"github.com/unstoppablemango/tdl/plugin"
)

func TestMatchImport(t *testing.T) {
	goFiles := []string{"!*_test.go", "*.go"}
	for name, want := range map[string]bool{
		"order.go":      true,
		"order_test.go": false,
		"order.proto":   false,
		"go":            false,
	} {
		if got := plugin.MatchImport(goFiles, name); got != want {
			t.Errorf("MatchImport(%v, %q) = %t, want %t", goFiles, name, got, want)
		}
	}
	if plugin.MatchImport(nil, "order.go") {
		t.Error("no patterns match a file")
	}
	if plugin.MatchImport([]string{"!*.go"}, "order.go") {
		t.Error("an exclude alone matches a file")
	}
	if plugin.MatchImport([]string{"[", "*.go"}, "order.proto") {
		t.Error("a malformed pattern matches a file")
	}
}

// The handshake reply carries the patterns a description declares.
func TestReplyCarriesImportPatterns(t *testing.T) {
	reply := plugin.Description{Name: "reader", Reverse: true, ImportPatterns: []string{"*.proto"}}.Reply()
	if got := reply.GetImportPatterns(); len(got) != 1 || got[0] != "*.proto" {
		t.Errorf("import patterns = %v, want [*.proto]", got)
	}
}
