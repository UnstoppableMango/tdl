package gen_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/gen/echo"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

var buildOnce struct {
	sync.Once
	dir string
	err error
}

// pluginDir builds every shipped backend and the echo test backend as
// plugins and returns the directory holding them.
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
				filepath.Join(dir, binary+exe),
				"github.com/unstoppablemango/tdl/cmd/"+binary)
			if out, err := cmd.CombinedOutput(); err != nil {
				buildOnce.err = err
				t.Logf("building %s: %s", binary, out)
				return
			}
		}
		// The echo test backend, which imports, is not shipped.
		cmd := exec.Command("go", "build", "-o",
			filepath.Join(dir, gen.CommandPrefix+echo.Name+exe),
			"github.com/unstoppablemango/tdl/internal/gen/echo/"+gen.CommandPrefix+echo.Name)
		if out, err := cmd.CombinedOutput(); err != nil {
			buildOnce.err = err
			t.Logf("building the echo backend: %s", out)
			return
		}
		buildOnce.dir = dir
	})

	if buildOnce.err != nil {
		t.Fatalf("building the plugin: %v", buildOnce.err)
	}
	return buildOnce.dir
}

// exe is the suffix of a binary go build writes, which Windows finds a
// command by.
var exe = map[bool]string{true: ".exe"}[runtime.GOOS == "windows"]

// writeScript writes a plugin named tdl-gen-<name> into dir: the shell
// script sh, or on Windows the batch file bat, since PATH finds only a file
// whose extension PATHEXT names there.
func writeScript(t *testing.T, dir, name, sh, bat string) {
	t.Helper()
	path, body := filepath.Join(dir, gen.CommandPrefix+name), "#!/bin/sh\n"+sh
	if runtime.GOOS == "windows" {
		path, body = path+".cmd", "@echo off\r\n"+bat+"exit /b %ERRORLEVEL%\r\n"
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
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

// A plugin that dies before answering has its stderr attached to the error.
func TestPluginThatDies(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "dies",
		"echo 'the backend fell over' >&2\nexit 3\n",
		"echo the backend fell over 1>&2\r\ncmd /c exit 3\r\n")
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

// A plugin serving two targets sees which one it is started for, in the
// handshake of a generate and of an import, and a target it refuses is
// refused in its own words.
func TestHandshakeNamesTheTarget(t *testing.T) {
	dir := twinDir(t)
	onPath(t, dir)

	for _, target := range []string{"left", "right"} {
		sub, err := gen.Find(target)
		if err != nil {
			t.Fatalf("find: %v", err)
		}
		resp, err := sub.Generate(context.Background(), &plugin.Request{Target: target, Model: sampleModel()})
		if err != nil {
			t.Fatalf("%s: generate: %v", target, err)
		}
		if got := resp.GetFiles(); len(got) != 1 || got[0].GetPath() != target+".txt" {
			t.Errorf("%s: generated %v, want %s.txt", target, got, target)
		}

		imported, err := sub.Import(context.Background(), &plugin.ImportRequest{Target: target})
		if err != nil {
			t.Fatalf("%s: import: %v", target, err)
		}
		if got := imported.GetSource(); got != "imported as "+target {
			t.Errorf("%s: import saw %q", target, got)
		}
	}

	sub, err := gen.Find("vb")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	_, err = sub.Generate(context.Background(), &plugin.Request{Target: "vb", Model: sampleModel()})
	if want := "tdl-gen-vb refused: serves left, right; asked for vb"; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
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

// twinDir builds a plugin that serves the targets left and right, without
// the SDK, and links it as tdl-gen-left, tdl-gen-right, and tdl-gen-vb.
// The links are hard links, since Windows grants a symbolic link only to
// an administrator or in developer mode.
func twinDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	src := filepath.Join(dir, "main.go")
	if err := os.WriteFile(src, []byte(twinSource), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(dir, "twin"+exe)
	if out, err := exec.Command(goBin(t), "build", "-o", bin, src).CombinedOutput(); err != nil {
		t.Fatalf("building the twin plugin: %v\n%s", err, out)
	}
	for _, target := range []string{"left", "right", "vb"} {
		if err := os.Link(bin, filepath.Join(dir, gen.CommandPrefix+target+exe)); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// twinSource answers for the target its handshake names, and refuses any
// other.
const twinSource = `package main

import (
	"os"

	"github.com/unstoppablemango/tdl/plugin"
)

func main() {
	conn := plugin.NewConn(os.Stdin, os.Stdout)
	var hello plugin.Handshake
	if err := conn.Recv(&hello); err != nil {
		os.Exit(1)
	}
	target := hello.GetTarget()
	if target != "left" && target != "right" {
		_ = conn.Send(&plugin.HandshakeReply{Refusal: "serves left, right; asked for " + target})
		return
	}
	_ = conn.Send(&plugin.HandshakeReply{
		Accepted: true,
		Name:     target,
		Features: &plugin.Features{Reverse: true},
	})
	if hello.GetMode() == plugin.Mode_MODE_IMPORT {
		_ = conn.Recv(&plugin.ImportRequest{})
		_ = conn.Send(&plugin.ImportResponse{Source: "imported as " + target})
		return
	}
	_ = conn.Recv(&plugin.Request{})
	_ = conn.Send(&plugin.Response{Files: []*plugin.File{{Path: target + ".txt"}}})
}
`
