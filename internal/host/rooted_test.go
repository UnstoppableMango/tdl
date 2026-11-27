package host_test

import (
	"runtime"
	"testing"

	"github.com/unstoppablemango/tdl/internal/host"
)

func TestRooted(t *testing.T) {
	windows := runtime.GOOS == "windows"
	for _, tt := range []struct {
		path string
		want bool
	}{
		{"/gen/ts", true},
		{"gen/ts", false},
		{"../gen", false},
		{"", false},
		// A backslash is a name on Unix and a root on Windows, as is a drive.
		{`\gen`, windows},
		{`C:\gen`, windows},
		{"C:gen", windows},
	} {
		if got := host.Rooted(tt.path); got != tt.want {
			t.Errorf("Rooted(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}
