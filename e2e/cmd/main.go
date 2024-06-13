package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	tdlv1alpha1 "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/**
var testdata embed.FS

var matcher = regexp.MustCompile(`(?P<name>\w*)\..*`)

type Test struct {
	Name   string
	Source string
	Target string
}

func main() {
	tests, err := readTests()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Running %d test(s)\n", len(tests))
	for _, test := range tests {
		fmt.Printf("Runnint test '%s'\n", test.Name)

		var spec tdlv1alpha1.Spec
		err = yaml.Unmarshal([]byte(test.Source), &spec)
		if err != nil {
			panic(err)
		}
	}
}

func readTests() ([]Test, error) {
	tests := []Test{}
	builder := NewTestBuilder()
	err := fs.WalkDir(testdata, "testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.Count(path, "/") == 1 {
			builder.WithName(d.Name())
		} else if strings.Count(path, "/") == 2 {
			file := d.Name()
			matches := matcher.FindStringSubmatch(file)
			i := matcher.SubexpIndex("name")

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			switch matches[i] {
			case "source":
				builder.WithSource(string(content))
			case "target":
				builder.WithTarget(string(content))
			}
		}

		if test, done := builder.Done(); done {
			tests = append(tests, test)
			builder = NewTestBuilder()
		}

		return nil
	})

	return tests, err
}
