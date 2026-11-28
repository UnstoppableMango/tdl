// Package tooltest finds the external tools tests run, such as tsc,
// smithy, or ghc, and holds every test to one rule: under -short a test
// that runs a tool skips, and otherwise a tool missing from PATH, or one
// older than the test needs, fails the test and names the tool.
//
// No test skips quietly for want of a tool, so a run without one can never
// look like it checked more than it did.
package tooltest

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"testing"
)

// hint ends every failure, saying how to opt out.
const hint = "-short skips the tests that run it"

// Require returns the path of name on PATH. Under -short it skips the
// test; otherwise a missing name fails it.
func Require(t testing.TB, name string) string {
	t.Helper()
	return require(t, testing.Short(), name)
}

// Lookup is [Require] for a test that goes on without the tool under
// -short, checking less: it returns false there rather than skipping.
// A missing name fails the test.
func Lookup(t testing.TB, name string) (string, bool) {
	t.Helper()
	return lookup(t, testing.Short(), name)
}

// Need holds a test to err, the reason a tool it runs cannot run here, as
// a finder such as smithy.CLI returns it; nil means it can. Under -short
// it skips the test; otherwise a non-nil err fails it.
func Need(t testing.TB, err error) {
	t.Helper()
	need(t, testing.Short(), err)
}

// RequireVersion is [Require] for a test that needs a version of name of
// at least min. It runs name with args and reads the version from the
// first match of pattern, whose groups are the version's numbers, most
// significant first. A version below min, or output pattern does not
// match, fails the test as a missing tool does.
func RequireVersion(t testing.TB, name string, args []string, pattern *regexp.Regexp, min ...int) string {
	t.Helper()
	return requireVersion(t, testing.Short(), name, args, pattern, min)
}

func require(t testing.TB, short bool, name string) string {
	t.Helper()
	if short {
		t.Skipf("runs %s, and -short skips it", name)
	}
	path, _ := lookup(t, false, name)
	return path
}

func lookup(t testing.TB, short bool, name string) (string, bool) {
	t.Helper()
	if short {
		return "", false
	}
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("%s is not on PATH; %s", name, hint)
	}
	return path, true
}

func need(t testing.TB, short bool, err error) {
	t.Helper()
	if short {
		t.Skip("runs an external tool, and -short skips it")
	}
	if err != nil {
		t.Fatalf("%v; %s", err, hint)
	}
}

func requireVersion(t testing.TB, short bool, name string, args []string, pattern *regexp.Regexp, min []int) string {
	t.Helper()
	path := require(t, short, name)
	out, err := exec.Command(path, args...).Output()
	if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	have, ok := version(pattern, string(out))
	if !ok {
		t.Fatalf("%s %v printed %q, which holds no version; %s", name, args, out, hint)
	}
	if older(have, min) {
		t.Fatalf("%s is %s, and the test needs %s or later; %s", name, dotted(have), dotted(min), hint)
	}
	return path
}

// version reads the numbers pattern's groups match in out.
func version(pattern *regexp.Regexp, out string) ([]int, bool) {
	m := pattern.FindStringSubmatch(out)
	if m == nil {
		return nil, false
	}
	have := make([]int, len(m)-1)
	for i, s := range m[1:] {
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, false
		}
		have[i] = n
	}
	return have, true
}

// older reports whether have is below min, comparing number by number; a
// number either leaves out counts as 0.
func older(have, min []int) bool {
	for i := range max(len(have), len(min)) {
		h, m := at(have, i), at(min, i)
		if h != m {
			return h < m
		}
	}
	return false
}

func at(v []int, i int) int {
	if i < len(v) {
		return v[i]
	}
	return 0
}

func dotted(v []int) string {
	s := ""
	for i, n := range v {
		if i > 0 {
			s += "."
		}
		s += fmt.Sprint(n)
	}
	return s
}
