package gen_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

var buildOnce struct {
	sync.Once
	dir string
	err error
}

// pluginDir builds every backend tdl ships as a plugin and returns the
// directory holding them, so a test can put it on PATH.
func pluginDir(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "tdl-plugins")
		if err != nil {
			buildOnce.err = err
			return
		}
		for _, row := range shipped {
			binary := gen.CommandPrefix + row.backend.Describe().Name
			cmd := exec.Command("go", "build", "-o",
				filepath.Join(dir, binary),
				"github.com/unstoppablemango/tdl/cmd/"+binary)
			if out, err := cmd.CombinedOutput(); err != nil {
				buildOnce.err = err
				t.Logf("building %s: %s", binary, out)
				return
			}
		}
		buildOnce.dir = dir
	})

	if buildOnce.err != nil {
		t.Fatalf("building the plugin: %v", buildOnce.err)
	}
	return buildOnce.dir
}

func onPath(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func sampleModel() *ir.Model {
	return &ir.Model{
		Package: "shop",
		Decls: []*ir.Decl{{
			Meta: &ir.Meta{Name: "Order", Position: &ir.Position{Filename: "shop.tdl"}},
			Node: &ir.Decl_Structure{Structure: &ir.Struct{
				Fields: []*ir.Field{{
					Meta: &ir.Meta{Name: "id"},
					Type: &ir.ID{Name: "string"},
				}},
			}},
		}},
	}
}

func TestFindReportsAMissingPlugin(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := gen.Find("nonesuch"); err == nil {
		t.Fatal("a plugin that is not there was found")
	}
}

// A plugin that dies before answering leaves a read error describing a
// pipe; its stderr is usually the actual answer, so it is attached.
func TestPluginThatDies(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, gen.CommandPrefix+"dies")
	body := "#!/bin/sh\necho 'the backend fell over' >&2\nexit 3\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	onPath(t, dir)

	sub, err := gen.Find("dies")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	_, err = sub.Generate(context.Background(), &plugin.Request{Target: "dies", Model: sampleModel()})
	if err == nil {
		t.Fatal("a plugin that exited 3 was treated as success")
	}
	if !strings.Contains(err.Error(), "the backend fell over") {
		t.Errorf("the plugin's stderr was not reported: %v", err)
	}
	if !strings.Contains(err.Error(), "dies") {
		t.Errorf("the error does not name the plugin: %v", err)
	}
}

// A plugin that refuses the handshake fails with both versions named,
// rather than silently ignoring what it cannot read.
func TestPluginThatRefuses(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, gen.CommandPrefix+"refuses")
	body := "#!/bin/sh\nexec " + goBin(t) + " run " + refuserPath(t) + "\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	onPath(t, dir)

	sub, err := gen.Find("refuses")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	_, err = sub.Generate(context.Background(), &plugin.Request{Target: "refuses", Model: sampleModel()})
	if err == nil {
		t.Fatal("a refusal was treated as acceptance")
	}
	if !strings.Contains(err.Error(), "refused") {
		t.Errorf("err = %v", err)
	}
}

func goBin(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("go is not on PATH: %v", err)
	}
	return path
}

// refuserPath writes a plugin that always refuses, since a real backend
// has no reason to.
func refuserPath(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	src := `package main

import (
	"os"

	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	conn := plugin.NewConn(os.Stdin, os.Stdout)
	_ = conn.Recv(&plugin.Handshake{})
	_ = conn.Send(&plugin.HandshakeReply{
		Accepted: false,
		Refusal:  "this plugin only speaks framing 99",
	})
}
`
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
