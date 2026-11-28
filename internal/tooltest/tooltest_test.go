package tooltest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// fake records how a helper ended a test. Skip and Fatal stop the
// goroutine as testing.T's do.
type fake struct {
	testing.TB
	skipped, failed string
}

func (*fake) Helper() {}

func (f *fake) Skip(args ...any) { f.skipped = fmt.Sprint(args...); runtime.Goexit() }

func (f *fake) Skipf(format string, args ...any) {
	f.skipped = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func (f *fake) Fatalf(format string, args ...any) {
	f.failed = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// run calls body with a fake on its own goroutine, so Goexit stops only it.
func run(body func(t testing.TB)) *fake {
	f := &fake{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		body(f)
	}()
	<-done
	return f
}

// onPath puts a directory holding the named scripts alone on PATH.
func onPath(t *testing.T, scripts map[string]string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake tools are shell scripts, which PATHEXT does not find")
	}
	dir := t.TempDir()
	for name, body := range scripts {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir)
	return dir
}

func TestRequire(t *testing.T) {
	dir := onPath(t, map[string]string{"present": "true"})

	var path string
	f := run(func(tb testing.TB) { path = require(tb, false, "present") })
	if f.skipped != "" || f.failed != "" || path != filepath.Join(dir, "present") {
		t.Errorf("present: path %q, skipped %q, failed %q", path, f.skipped, f.failed)
	}

	f = run(func(tb testing.TB) { require(tb, false, "missing") })
	if !strings.Contains(f.failed, "missing") || !strings.Contains(f.failed, "-short") {
		t.Errorf("missing: failed %q, want the tool and -short named", f.failed)
	}

	for _, name := range []string{"present", "missing"} {
		f = run(func(tb testing.TB) { require(tb, true, name) })
		if f.skipped == "" || f.failed != "" {
			t.Errorf("-short %s: skipped %q, failed %q", name, f.skipped, f.failed)
		}
	}
}

func TestLookup(t *testing.T) {
	onPath(t, map[string]string{"present": "true"})

	var ok bool
	f := run(func(tb testing.TB) { _, ok = lookup(tb, false, "present") })
	if !ok || f.failed != "" {
		t.Errorf("present: ok %v, failed %q", ok, f.failed)
	}

	f = run(func(tb testing.TB) { _, ok = lookup(tb, true, "missing") })
	if ok || f.skipped != "" || f.failed != "" {
		t.Errorf("-short: ok %v, skipped %q, failed %q; want it to carry on without", ok, f.skipped, f.failed)
	}

	f = run(func(tb testing.TB) { lookup(tb, false, "missing") })
	if !strings.Contains(f.failed, "missing") {
		t.Errorf("missing: failed %q", f.failed)
	}
}

func TestNeed(t *testing.T) {
	f := run(func(tb testing.TB) { need(tb, false, nil) })
	if f.skipped != "" || f.failed != "" {
		t.Errorf("nil: skipped %q, failed %q", f.skipped, f.failed)
	}

	f = run(func(tb testing.TB) { need(tb, false, fmt.Errorf("needs smithy on PATH")) })
	if !strings.Contains(f.failed, "smithy") || !strings.Contains(f.failed, "-short") {
		t.Errorf("err: failed %q", f.failed)
	}

	f = run(func(tb testing.TB) { need(tb, true, nil) })
	if f.skipped == "" {
		t.Error("-short did not skip")
	}
}

func TestRequireVersion(t *testing.T) {
	onPath(t, map[string]string{
		"old":     "echo Version 7.0.9",
		"new":     "echo Version 7.1.0",
		"newer":   "echo Version 10.0",
		"garbled": "echo nightly",
	})
	pattern := regexp.MustCompile(`Version (\d+)\.(\d+)`)
	for name, wantFail := range map[string]string{
		"old":     "old is 7.0, and the test needs 7.1 or later",
		"new":     "",
		"newer":   "",
		"garbled": "holds no version",
	} {
		t.Run(name, func(t *testing.T) {
			f := run(func(tb testing.TB) { requireVersion(tb, false, name, []string{"--version"}, pattern, []int{7, 1}) })
			if wantFail == "" && f.failed != "" {
				t.Errorf("failed %q", f.failed)
			}
			if wantFail != "" && (!strings.Contains(f.failed, wantFail) || !strings.Contains(f.failed, "-short")) {
				t.Errorf("failed %q, want %q", f.failed, wantFail)
			}
		})
	}

	f := run(func(tb testing.TB) { requireVersion(tb, true, "old", nil, pattern, []int{7, 1}) })
	if f.skipped == "" {
		t.Error("-short did not skip")
	}
}
