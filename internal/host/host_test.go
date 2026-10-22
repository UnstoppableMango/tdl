package host_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/plugin"
)

// A warning with an allowed code is dropped; an error and a warning with
// no code never are.
func TestSilence(t *testing.T) {
	diags := []*plugin.Diagnostic{
		{Severity: plugin.Severity_SEVERITY_WARNING, Message: "order", Code: "lossy.order"},
		{Severity: plugin.Severity_SEVERITY_WARNING, Message: "set", Code: "lossy.collection"},
		{Severity: plugin.Severity_SEVERITY_ERROR, Message: "broken", Code: "lossy.order"},
		{Severity: plugin.Severity_SEVERITY_WARNING, Message: "plain"},
	}
	got := host.Silence(diags, []string{"lossy.order", ""})

	var msgs []string
	for _, d := range got {
		msgs = append(msgs, d.GetMessage())
	}
	if want := "set broken plain"; strings.Join(msgs, " ") != want {
		t.Errorf("kept %v, want %s", msgs, want)
	}
	if len(diags) != 4 {
		t.Error("Silence changed its argument")
	}
}

// Allowed reads the nearest tdl.toml's codes for one target.
func TestAllowed(t *testing.T) {
	dir := t.TempDir()
	toml := "[lossy.typescript]\nallow = [\"lossy.alias\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "tdl.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "models")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := host.Allowed(sub, "typescript")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []string{"lossy.alias"}) {
		t.Errorf("Allowed = %v, want [lossy.alias]", got)
	}
	if got, _ := host.Allowed(sub, "go"); len(got) != 0 {
		t.Errorf("Allowed(go) = %v, want none", got)
	}
}
