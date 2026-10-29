// Command nugetdeps regenerates dotnet/deps.json, the NuGet lock the .NET
// plugins build from, by restoring the solution into an empty packages
// folder and hashing what it holds. It needs dotnet on PATH and network
// access, and nothing from Nix, so Renovate runs it too.
//
//	go run ./tools/nugetdeps          # rewrite the lock
//	go run ./tools/nugetdeps -check   # fail, naming the drift, if it is stale
//
// See internal/nugetdeps.
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/unstoppablemango/tdl/internal/nugetdeps"
)

const (
	lock    = "dotnet/deps.json"
	project = "dotnet/Tdl.slnx"
)

// rids are the runtimes the flake builds for. Restoring each into one folder
// takes in a package only one of them needs.
var rids = []string{"linux-x64", "linux-arm64", "osx-x64", "osx-arm64"}

func main() {
	check := flag.Bool("check", false, "write nothing, and fail if the lock differs from what restores")
	flag.Parse()

	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	dir, err := os.MkdirTemp("", "nugetdeps")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	for _, rid := range rids {
		if err := nugetdeps.Restore(context.Background(), project, rid, dir); err != nil {
			return err
		}
	}
	restored, err := nugetdeps.Scan(dir)
	if err != nil {
		return err
	}
	data := nugetdeps.Marshal(restored)

	if !check {
		return os.WriteFile(lock, data, 0o644)
	}
	old, err := os.ReadFile(lock)
	if err != nil {
		return err
	}
	if bytes.Equal(old, data) {
		return nil
	}
	pkgs, err := nugetdeps.Parse(old)
	if err != nil {
		return fmt.Errorf("%s: %w", lock, err)
	}
	var msg bytes.Buffer
	fmt.Fprintf(&msg, "%s is out of date with %s:\n", lock, project)
	lines := nugetdeps.Diff(pkgs, restored)
	if lines == nil {
		lines = []string{"the packages agree, but not their order or layout"}
	}
	for _, line := range lines {
		fmt.Fprintf(&msg, "\t%s\n", line)
	}
	fmt.Fprint(&msg, "run `make dotnet-deps` and commit the result")
	return errors.New(msg.String())
}
