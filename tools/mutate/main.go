// Command mutate runs gremlins over each package in the mutation testing
// scope, writes the merged report, and prints what survived.
//
// Run it from the module root, in the devShell, which carries gremlins:
//
//	go run ./tools/mutate              # every package in scope
//	go run ./tools/mutate ./lex        # one
//	go run ./tools/mutate -report mutate.json   # reprint a report
//
// See docs/architecture/mutation-testing.md.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/internal/mutation"
)

// scope is the packages mutated when none is named.
var scope = []string{"lex", "parser", "ast", "internal/sema", "internal/analysis"}

func main() {
	out := flag.String("out", "mutate.json", "the merged report to write")
	report := flag.String("report", "", "print this report instead of running gremlins")
	equivalent := flag.String("equivalent", "mutation/equivalent.txt", "the equivalent mutants to leave out")
	flag.Parse()

	if err := run(flag.Args(), *out, *report, *equivalent); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, out, report, equivalent string) error {
	pkgs := scope
	if len(args) > 0 {
		pkgs = make([]string, len(args))
		for i, a := range args {
			pkgs[i] = path.Clean(filepath.ToSlash(a))
		}
	}

	eq, err := readEquivalent(equivalent)
	if err != nil {
		return err
	}

	var rep *mutation.Report
	if report != "" {
		if rep, err = readReport(report); err != nil {
			return err
		}
	} else {
		if rep, err = unleash(pkgs); err != nil {
			return err
		}
		if err := writeReport(out, rep); err != nil {
			return err
		}
	}
	return mutation.Summarize(rep, pkgs, eq).Print(os.Stdout)
}

func readEquivalent(name string) ([]mutation.Mutant, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	eq, err := mutation.ParseEquivalent(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return eq, nil
}

func readReport(name string) (*mutation.Report, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rep, err := mutation.Read(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return rep, nil
}

func writeReport(name string, rep *mutation.Report) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	if err := mutation.Write(f, rep); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// unleash runs gremlins once per package, since a report names its files
// relative to the package it ran in, and merges the reports.
// gremlins has no flag for -short, so GOFLAGS carries it to the coverage
// run and to every mutant.
func unleash(pkgs []string) (*mutation.Report, error) {
	dir, err := os.MkdirTemp("", "mutate")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	env := append(os.Environ(), "GOFLAGS="+strings.TrimSpace(os.Getenv("GOFLAGS")+" -short"))
	reps := make([]*mutation.Report, len(pkgs))
	for i, pkg := range pkgs {
		fmt.Fprintf(os.Stderr, "==> %s <==\n", pkg)
		file := filepath.Join(dir, fmt.Sprintf("%d.json", i))
		cmd := exec.Command("gremlins", "unleash", "--output", file, "./"+pkg)
		cmd.Env = env
		// gremlins narrates every mutant; the summary is what goes to stdout.
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("gremlins unleash ./%s: %w", pkg, err)
		}
		if reps[i], err = readReport(file); err != nil {
			return nil, err
		}
	}
	return mutation.Merge(pkgs, reps), nil
}
