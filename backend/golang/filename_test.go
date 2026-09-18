package golang_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/backend/internal/irtest"
)

// goSuffixes is what go/build matches the last underscore-separated element
// of a file name against: the GOOS and GOARCH lists in internal/syslist,
// plus the test suffix that leaves a file out of the package.
var goSuffixes = []string{
	"test",
	"aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos",
	"ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris",
	"wasip1", "windows", "zos",
	"386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be",
	"loong64", "mips", "mipsle", "mips64", "mips64le", "mips64p32",
	"mips64p32le", "ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390",
	"s390x", "sparc", "sparc64", "wasm",
}

// A declaration named FooTest is a declaration, so foo_test.go would leave
// it out of the package, and OrderLinux is not an OS-specific type.
func TestFileNameAvoidsGoSuffixes(t *testing.T) {
	m := irtest.New("shop")
	for _, s := range goSuffixes {
		m.Own(structure("Report_"+s, nil, irtest.Field("id", m.Named("string"))))
	}
	// The same suffixes reached through camel case, including the pair form
	// go/build reads as a GOOS and a GOARCH together.
	for _, name := range []string{"FooTest", "OrderLinux", "ReportArm64", "JobLinuxAmd64", "TaskLinuxTest"} {
		m.Own(structure(name, nil, irtest.Field("id", m.Named("string"))))
	}

	for path := range files(t, generate(t, m)) {
		base := strings.TrimSuffix(path, ".go")
		// go/build ignores everything up to the first underscore, so
		// linux.go carries no constraint and foo_linux.go does.
		i := strings.Index(base, "_")
		if i < 0 {
			continue
		}
		parts := strings.Split(base[i+1:], "_")
		if last := parts[len(parts)-1]; slices.Contains(goSuffixes, last) {
			t.Errorf("%s ends in _%s, which go build reads as a test file or a build constraint", path, last)
		}
	}
}

// Only a suffix Go reads is escaped, so every other name keeps the plain
// snake case spelling.
func TestFileNameKeepsUnreservedNames(t *testing.T) {
	m := irtest.New("shop")
	for _, name := range []string{"Order", "Linux", "Test", "TestCase", "Arm64Report"} {
		m.Own(structure(name, nil, irtest.Field("id", m.Named("string"))))
	}

	got := files(t, generate(t, m))
	for _, want := range []string{"order.go", "linux.go", "test.go", "test_case.go", "arm64_report.go"} {
		if _, ok := got[want]; !ok {
			t.Errorf("no %s in %v", want, keys(got))
		}
	}
}
