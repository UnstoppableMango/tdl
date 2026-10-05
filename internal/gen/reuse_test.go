package gen_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/unstoppablemango/tdl/backend/debug"
	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/plugin"
)

func TestSessionServesSeveralRequests(t *testing.T) {
	onPath(t, pluginDir(t))

	sub, err := gen.Find(debug.Name)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	session, err := gen.Open(context.Background(), sub)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer session.Close()

	if !session.Reused() {
		t.Fatal("a backend that declares reuse was not held open")
	}

	for i := 0; i < 3; i++ {
		resp, err := session.Generate(context.Background(), &plugin.Request{
			Target: debug.Name,
			Model:  sampleModel(),
		})
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if len(resp.GetFiles()) != 1 {
			t.Fatalf("request %d returned %d files", i, len(resp.GetFiles()))
		}
	}
}

// A backend that does not declare reuse gets a fresh process per
// generation.
func TestSessionWithoutReuse(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, gen.CommandPrefix+"once")
	body := "#!/bin/sh\nexec " + goBin(t) + " run " + oncePath(t) + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	onPath(t, dir)

	sub, err := gen.Find("once")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	session, err := gen.Open(context.Background(), sub)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer session.Close()

	if session.Reused() {
		t.Error("a backend that did not declare reuse was held open")
	}

	for i := 0; i < 2; i++ {
		if _, err := session.Generate(context.Background(), &plugin.Request{Target: "once"}); err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
	}
}

// A held connection restarts when the binary underneath changes.
func TestSessionRestartsOnANewBinary(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, gen.CommandPrefix+debug.Name)
	build(t, binary)
	onPath(t, dir)

	sub, err := gen.Find(debug.Name)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	session, err := gen.Open(context.Background(), sub)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer session.Close()

	if _, err := session.Generate(context.Background(), &plugin.Request{Target: debug.Name, Model: sampleModel()}); err != nil {
		t.Fatalf("first request: %v", err)
	}

	// Replace it.
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(binary, future, future); err != nil {
		t.Fatal(err)
	}

	if _, err := session.Generate(context.Background(), &plugin.Request{Target: debug.Name, Model: sampleModel()}); err != nil {
		t.Fatalf("after replacing the binary: %v", err)
	}
	if session.Restarts() != 1 {
		t.Errorf("restarts = %d, want the session to have noticed", session.Restarts())
	}
	if !session.Reused() {
		t.Error("the session did not come back up")
	}

	// A binary that has not changed is not restarted again.
	if _, err := session.Generate(context.Background(), &plugin.Request{Target: debug.Name, Model: sampleModel()}); err != nil {
		t.Fatalf("third request: %v", err)
	}
	if session.Restarts() != 1 {
		t.Errorf("restarts = %d, want no further restart", session.Restarts())
	}
}

func build(t *testing.T, path string) {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(pluginDir(t), gen.CommandPrefix+debug.Name))
	if err != nil {
		t.Fatalf("reading the built plugin: %v", err)
	}
	if err := os.WriteFile(path, src, 0o755); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// oncePath writes a plugin that does not declare reuse.
func oncePath(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := `package main

import (
	"context"
	"os"

	"github.com/unstoppablemango/tdl/plugin"
)

type once struct{}

func (once) Describe() plugin.Description {
	return plugin.Description{Name: "once", Version: "1"}
}

func (once) Generate(context.Context, *plugin.Request) (*plugin.Response, error) {
	return &plugin.Response{Files: []*plugin.File{{Path: "once.txt"}}}, nil
}

func main() {
	if err := plugin.ServeConn(context.Background(), once{}, plugin.NewConn(os.Stdin, os.Stdout)); err != nil {
		os.Exit(1)
	}
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
