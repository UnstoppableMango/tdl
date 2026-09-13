package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --watch regenerates on every save until the context ends.
func TestGenWatchRegeneratesOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "m.tdl")
	out := filepath.Join(dir, "out")
	model := filepath.Join(out, "model.txt")

	write := func(decl string) {
		t.Helper()
		src := "package p\n\nprimitive string\n\n" + decl + "\n\ntarget debug for p {\n  out(\"./out\")\n}\n"
		if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("type Before: Entity {\n  id: string\n}")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmd := newGenCmd()
	cmd.SilenceUsage, cmd.SilenceErrors = true, true
	captureCmd(cmd)
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"-o", out, "--watch", path})

	done := make(chan error, 1)
	go func() { done <- cmd.Execute() }()

	waitForOutput(t, model, "Before")
	write("type After: Entity {\n  id: string\n}")
	waitForOutput(t, model, "After")

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("watch returned %v", err)
	}
}

// waitForOutput polls until the generated file mentions want, since a
// watch regenerates on its own schedule.
func waitForOutput(t *testing.T, path, want string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && strings.Contains(string(b), want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("%s never mentioned %q", path, want)
}
