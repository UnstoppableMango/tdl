package nugetdeps

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// restore writes a package as restore leaves it in a packages folder:
// lowercased directories, the nuspec carrying the id's own case.
func restore(t *testing.T, dir, id, version, nupkg string) string {
	t.Helper()
	lower := strings.ToLower(id)
	d := filepath.Join(dir, lower, version)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	nuspec := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>` + id + `</id>
    <version>` + version + `</version>
  </metadata>
</package>`
	write(t, filepath.Join(d, lower+".nuspec"), nuspec)
	write(t, filepath.Join(d, lower+"."+version+".nupkg"), nupkg)
	write(t, filepath.Join(d, lower+"."+version+".nupkg.sha512"), "unused")
	sum := sha256.Sum256([]byte(nupkg))
	return "sha256-" + base64.StdEncoding.EncodeToString(sum[:])
}

func write(t *testing.T, name, text string) {
	t.Helper()
	if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanReadsIdHashAndOrder(t *testing.T) {
	dir := t.TempDir()
	xunit := restore(t, dir, "xunit.v3", "3.2.2", "x")
	xassert := restore(t, dir, "xunit.v3.assert", "3.2.2", "a")
	csharp := restore(t, dir, "Microsoft.CodeAnalysis.CSharp", "5.0.0", "c")
	common := restore(t, dir, "Microsoft.CodeAnalysis.Common", "5.0.0", "o")
	platform := restore(t, dir, "Microsoft.Testing.Platform", "1.9.1", "p")
	msbuild := restore(t, dir, "Microsoft.Testing.Platform.MSBuild", "1.9.1", "m")

	got, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Ordered as nixpkgs' nuget-to-json orders: by lowercased id, bytewise.
	want := []Package{
		{"Microsoft.CodeAnalysis.Common", "5.0.0", common},
		{"Microsoft.CodeAnalysis.CSharp", "5.0.0", csharp},
		{"Microsoft.Testing.Platform", "1.9.1", platform},
		{"Microsoft.Testing.Platform.MSBuild", "1.9.1", msbuild},
		{"xunit.v3", "3.2.2", xunit},
		{"xunit.v3.assert", "3.2.2", xassert},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan:\ngot  %v\nwant %v", got, want)
	}
}

func TestScanKeepsEveryVersion(t *testing.T) {
	dir := t.TempDir()
	a := restore(t, dir, "Newtonsoft.Json", "13.0.3", "a")
	b := restore(t, dir, "Newtonsoft.Json", "13.0.1", "b")

	got, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Package{
		{"Newtonsoft.Json", "13.0.1", b},
		{"Newtonsoft.Json", "13.0.3", a},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan:\ngot  %v\nwant %v", got, want)
	}
}

func TestScanLeavesOutWhatTheSDKCarries(t *testing.T) {
	dir := t.TempDir()
	for _, id := range []string{
		"Microsoft.NET.ILLink.Tasks",
		"Microsoft.DotNet.ILCompiler",
		"runtime.linux-x64.Microsoft.DotNet.ILCompiler",
		"runtime.osx-arm64.Microsoft.DotNet.ILCompiler",
		"Microsoft.NETCore.App.Runtime.linux-x64",
		"Microsoft.NETCore.App.Host.osx-arm64",
		"Microsoft.NETCore.App.Ref",
		"Microsoft.AspNetCore.App.Runtime.linux-x64",
		"Microsoft.NETCore.DotNetAppHost",
		"runtime.linux-x64.Microsoft.NETCore.DotNetAppHost",
	} {
		restore(t, dir, id, "10.0.12", id)
	}
	kept := restore(t, dir, "Google.Protobuf", "3.36.2", "g")

	got, err := Scan(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []Package{{"Google.Protobuf", "3.36.2", kept}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Scan:\ngot  %v\nwant %v", got, want)
	}
}

func TestSDKCarriedIsNarrow(t *testing.T) {
	for _, id := range []string{
		"Microsoft.NET.Test.Sdk",
		"Microsoft.NETCore.Platforms",
		"Microsoft.CodeAnalysis.CSharp",
		"runtime.native.System",
	} {
		if SDKCarried(id) {
			t.Errorf("SDKCarried(%q) = true", id)
		}
	}
}

func TestMarshalIsJsonfmtCanonical(t *testing.T) {
	got := Marshal([]Package{
		{"Google.Protobuf", "3.36.2", "sha256-a"},
		{"xunit.v3", "3.2.2", "sha256-b"},
	})
	want := `[
	{
		"pname": "Google.Protobuf",
		"version": "3.36.2",
		"hash": "sha256-a"
	},
	{
		"pname": "xunit.v3",
		"version": "3.2.2",
		"hash": "sha256-b"
	}
]
`
	if string(got) != want {
		t.Errorf("Marshal:\n%s\nwant\n%s", got, want)
	}
}

// The committed lock is what Marshal writes, so regenerating it changes
// nothing but the packages, and it names nothing the SDK carries.
func TestCommittedLockIsCanonical(t *testing.T) {
	data, err := os.ReadFile("../../dotnet/deps.json")
	if err != nil {
		t.Fatal(err)
	}
	pkgs, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := Marshal(pkgs); !bytes.Equal(got, data) {
		t.Errorf("dotnet/deps.json is not what Marshal writes:\n%s", got)
	}
	sorted := append([]Package(nil), pkgs...)
	sortPackages(sorted)
	if !reflect.DeepEqual(sorted, pkgs) {
		t.Error("dotnet/deps.json is not in nuget-to-json's order")
	}
	for _, p := range pkgs {
		if SDKCarried(p.Pname) {
			t.Errorf("dotnet/deps.json names %s, which the SDK carries", p.Pname)
		}
	}
}
