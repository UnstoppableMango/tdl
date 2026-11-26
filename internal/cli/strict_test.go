package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/gen/echo"
	"github.com/unstoppablemango/tdl/plugin"
)

// warnBackend generates echo's file and one warning, carrying code when
// it is set.
type warnBackend struct {
	echo.Backend
	code string
}

func (b warnBackend) Generate(ctx context.Context, req *plugin.Request) (*plugin.Response, error) {
	resp, err := b.Backend.Generate(ctx, req)
	if err == nil {
		resp.Diagnostics = append(resp.Diagnostics, &plugin.Diagnostic{
			Severity: plugin.Severity_SEVERITY_WARNING,
			Code:     b.code,
			Message:  "something was lost",
		})
	}
	return resp, err
}

const strictSummary = "1 warning(s) remain under --strict"

// strictModel writes a model with an echo target block, and the manifest
// beside it when there is one, returning the model and its output
// directory.
func strictModel(t *testing.T, manifest string) (path, out string) {
	t.Helper()
	dir := t.TempDir()
	path = filepath.Join(dir, "m.tdl")
	if err := os.WriteFile(path, []byte("package p\n\nprimitive string\n\ntarget echo for p {\n  out(\"./out\")\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if manifest != "" {
		if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(manifest), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path, filepath.Join(dir, "out")
}

func strictGen(t *testing.T, code string, args ...string) (string, error) {
	t.Helper()
	cmd := genCmd(func(string) (plugin.Backend, error) { return warnBackend{code: code}, nil })
	_, errOut, err := run(t, cmd, args...)
	return errOut, err
}

// Strict fails a run on a warning that silencing left, coded or not, and
// still writes the files.
func TestGenStrict(t *testing.T) {
	cases := []struct {
		name, code, manifest string
		args                 []string
		fails                bool
	}{
		{"off", "lossy.order", "", nil, false},
		{"coded", "lossy.order", "", []string{"--strict"}, true},
		{"uncoded", "", "", []string{"--strict"}, true},
		{"uncoded is never allowed", "", "[lossy]\nallow = [\"lossy.order\"]\n", []string{"--strict"}, true},
		{"allowed by flag", "lossy.order", "", []string{"--strict", "--allow-lossy", "lossy.order"}, false},
		{"allowed by tdl.toml", "lossy.order", "[lossy]\nallow = [\"lossy.order\"]\n", []string{"--strict"}, false},
		{"tdl.toml", "lossy.order", "[lossy]\nstrict = true\n", nil, true},
		{"flag overrides tdl.toml", "lossy.order", "[lossy]\nstrict = true\n", []string{"--strict=false"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, out := strictModel(t, tc.manifest)
			errOut, err := strictGen(t, tc.code, append(tc.args, path)...)
			if tc.fails != (err != nil) {
				t.Fatalf("err = %v, want failure %v\n%s", err, tc.fails, errOut)
			}
			if tc.fails && err.Error() != strictSummary {
				t.Errorf("err = %q, want %q", err, strictSummary)
			}
			if _, statErr := os.Stat(filepath.Join(out, echo.File)); statErr != nil {
				t.Errorf("nothing written: %v", statErr)
			}
		})
	}
}

// Under --verify, strict fails on a warning, drift, or both, reporting
// each.
func TestGenStrictVerify(t *testing.T) {
	cases := []struct {
		name, code string
		args       []string
		stale      bool
		want       []string
	}{
		{"warning", "lossy.order", []string{"--strict"}, false, []string{strictSummary}},
		{"drift", "lossy.order", nil, true, []string{"1 file failed"}},
		{"both", "lossy.order", []string{"--strict"}, true, []string{"1 file failed", strictSummary}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, out := strictModel(t, "")
			if errOut, err := strictGen(t, tc.code, path); err != nil {
				t.Fatalf("gen: %v\n%s", err, errOut)
			}
			if tc.stale {
				if err := os.WriteFile(filepath.Join(out, echo.File), []byte("stale\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			errOut, err := strictGen(t, tc.code, append([]string{"--verify", path}, tc.args...)...)
			if err == nil {
				t.Fatalf("--verify passed\n%s", errOut)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("err = %q, want it to contain %q", err, want)
				}
			}
			if !tc.stale && strings.Contains(err.Error(), "failed") {
				t.Errorf("err = %q, want no drift", err)
			}
		})
	}
}

// Import is strict the same way, from the flag or the manifest.
func TestImportStrict(t *testing.T) {
	cases := []struct {
		name, manifest string
		args           []string
		fails          bool
	}{
		{"off", "", nil, false},
		{"flag", "", []string{"--strict"}, true},
		{"allowed", "", []string{"--strict", "--allow-lossy", "lossy.order"}, false},
		{"tdl.toml", "[lossy]\nstrict = true\n", nil, true},
		{"flag overrides tdl.toml", "[lossy]\nstrict = true\n", []string{"--strict=false"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := echoFile(t)
			if tc.manifest != "" {
				if err := os.WriteFile(filepath.Join(filepath.Dir(path), config.FileName), []byte(tc.manifest), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			dest := filepath.Join(t.TempDir(), "imported.tdl")
			_, errOut, err := run(t, importCmd(echoResolver), append([]string{"--from", "echo", "-o", dest, path}, tc.args...)...)
			if tc.fails != (err != nil) {
				t.Fatalf("err = %v, want failure %v\n%s", err, tc.fails, errOut)
			}
			if tc.fails && err.Error() != strictSummary {
				t.Errorf("err = %q, want %q", err, strictSummary)
			}
			if _, statErr := os.Stat(dest); statErr != nil {
				t.Errorf("nothing written: %v", statErr)
			}
		})
	}
}
