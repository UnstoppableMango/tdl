package host_test

import (
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
