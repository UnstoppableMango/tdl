package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

//go:embed testdata/**
var testdata embed.FS

var matcher = regexp.MustCompile(".")

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

	for _, test := range tests {
		fmt.Println(test.Name)
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
			switch matches[i] {
			case "source":
				content, err := os.ReadFile(path)
				builder.WithSource(string(content))
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
