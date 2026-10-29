// Command nugetdeps regenerates dotnet/deps.json, the NuGet lock the .NET
// plugins build from, by restoring the solution into an empty packages
// folder and hashing what it holds. It needs dotnet on PATH and network
// access, and nothing from Nix, so Renovate runs it too.
//
//	go run ./tools/nugetdeps
//
// CI runs it and fails if the lock changes.
//
// See internal/nugetdeps.
package main

import (
	"context"
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
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
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
	return os.WriteFile(lock, nugetdeps.Marshal(restored), 0o644)
}
