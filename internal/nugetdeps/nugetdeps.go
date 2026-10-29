// Package nugetdeps writes and checks the NuGet lock buildDotnetModule
// restores the .NET plugins from: dotnet/deps.json, the nugetDeps of
// nix/dotnet.nix.
//
// The lock is what nixpkgs' nuget-to-json writes for a packages folder, in
// the same order and with the same hashes, less what the SDK carries, and
// laid out as jsonfmt formats it. It is generated here rather than with
// nuget-to-json so that Renovate, which has no Nix, regenerates it with
// the same command a person and CI use.
package nugetdeps

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Package is one entry of the lock.
type Package struct {
	Pname   string `json:"pname"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

func (p Package) String() string { return p.Pname + " " + p.Version }

// Restore restores project for rid into the empty packages folder dir.
func Restore(ctx context.Context, project, rid, dir string) error {
	cmd := exec.CommandContext(ctx, "dotnet", "restore", project, "-r", rid, "--packages", dir)
	cmd.Env = append(os.Environ(), "DOTNET_CLI_TELEMETRY_OPTOUT=1", "DOTNET_NOLOGO=1")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dotnet restore %s: %w", project, err)
	}
	return nil
}

// Scan reads the packages folder restore filled, leaving out what the SDK
// carries. Each package's id is its nuspec's, since the folder is
// lowercased, and its hash is the SRI sha256 of its .nupkg, which is what
// fetchurl checks.
func Scan(dir string) ([]Package, error) {
	versions, err := filepath.Glob(filepath.Join(dir, "*", "*"))
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	for _, v := range versions {
		lower, version := filepath.Base(filepath.Dir(v)), filepath.Base(v)
		id, err := nuspecID(filepath.Join(v, lower+".nuspec"))
		if err != nil {
			return nil, err
		}
		if SDKCarried(id) {
			continue
		}
		hash, err := sriSHA256(filepath.Join(v, lower+"."+version+".nupkg"))
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, Package{id, version, hash})
	}
	sortPackages(pkgs)
	return pkgs, nil
}

func nuspecID(name string) (string, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		return "", err
	}
	var spec struct {
		ID string `xml:"metadata>id"`
	}
	if err := xml.Unmarshal(data, &spec); err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	if spec.ID == "" {
		return "", fmt.Errorf("%s: no id", name)
	}
	return spec.ID, nil
}

func sriSHA256(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return "sha256-" + base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// sortPackages orders as nuget-to-json does: its glob over the lowercased
// folder runs under LC_ALL=C.
func sortPackages(pkgs []Package) {
	sort.SliceStable(pkgs, func(i, j int) bool {
		a, b := strings.ToLower(pkgs[i].Pname), strings.ToLower(pkgs[j].Pname)
		if a != b {
			return a < b
		}
		return pkgs[i].Version < pkgs[j].Version
	})
}

// sdkPrefixes and sdkIDs name what the SDK carries in nixpkgs, for any
// runtime: the targeting, runtime, and host packs, the trimming tasks, and
// the AOT compiler. buildDotnetModule supplies them from the SDK, so the
// lock leaves them out, and their versions follow the SDK rather than the
// projects.
var (
	sdkPrefixes = []string{"Microsoft.NETCore.App.", "Microsoft.AspNetCore.App."}
	sdkIDs      = []string{"Microsoft.DotNet.ILCompiler", "Microsoft.NET.ILLink.Tasks", "Microsoft.NETCore.DotNetAppHost"}
)

// SDKCarried reports whether the SDK carries the package id.
func SDKCarried(id string) bool {
	for _, p := range sdkPrefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	// runtime.<rid>.Microsoft.DotNet.ILCompiler and the like.
	if rest, ok := strings.CutPrefix(id, "runtime."); ok {
		if _, name, ok := strings.Cut(rest, "."); ok {
			id = name
		}
	}
	for _, s := range sdkIDs {
		if id == s {
			return true
		}
	}
	return false
}

// Parse reads a lock.
func Parse(data []byte) ([]Package, error) {
	var pkgs []Package
	if err := json.Unmarshal(data, &pkgs); err != nil {
		return nil, err
	}
	return pkgs, nil
}

// Marshal writes a lock as jsonfmt lays it out, so nix fmt leaves it alone.
func Marshal(pkgs []Package) []byte {
	if pkgs == nil {
		pkgs = []Package{}
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "\t")
	if err := enc.Encode(pkgs); err != nil {
		panic(err) // three strings always encode
	}
	return buf.Bytes()
}

// Diff describes, one line per package, how the lock differs from what the
// projects restore. It is nil when they agree.
func Diff(lock, restored []Package) []string {
	type sides struct{ lock, restored []Package }
	byID := map[string]*sides{}
	var ids []string
	side := func(p Package) *sides {
		k := strings.ToLower(p.Pname)
		if byID[k] == nil {
			byID[k] = &sides{}
			ids = append(ids, k)
		}
		return byID[k]
	}
	for _, p := range lock {
		s := side(p)
		s.lock = append(s.lock, p)
	}
	for _, p := range restored {
		s := side(p)
		s.restored = append(s.restored, p)
	}
	sort.Strings(ids)

	var lines []string
	for _, k := range ids {
		s := byID[k]
		if len(s.lock) == 1 && len(s.restored) == 1 && s.lock[0].Version != s.restored[0].Version {
			lines = append(lines, fmt.Sprintf("%s: the lock has %s, the projects restore %s",
				s.restored[0].Pname, s.lock[0].Version, s.restored[0].Version))
			continue
		}
		for _, l := range s.lock {
			r, ok := find(s.restored, l.Version)
			switch {
			case !ok:
				lines = append(lines, fmt.Sprintf("%s: in the lock, but no project restores it", l))
			case r.Hash != l.Hash:
				lines = append(lines, fmt.Sprintf("%s: the lock's hash is %s, the package's %s", l, l.Hash, r.Hash))
			}
		}
		for _, r := range s.restored {
			if _, ok := find(s.lock, r.Version); !ok {
				lines = append(lines, fmt.Sprintf("%s: restored, but missing from the lock", r))
			}
		}
	}
	return lines
}

func find(pkgs []Package, version string) (Package, bool) {
	for _, p := range pkgs {
		if p.Version == version {
			return p, true
		}
	}
	return Package{}, false
}
