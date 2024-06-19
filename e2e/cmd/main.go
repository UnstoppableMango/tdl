package main

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"

	tdl "github.com/unstoppablemango/tdl/gen/proto/go/unmango/dev/tdl/v1alpha1"
	"github.com/unstoppablemango/tdl/pkg/runner"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/**
var testdata embed.FS

var matcher = regexp.MustCompile(`(?P<name>\w*)\..*`)
var binDir = os.Getenv("BIN_DIR")

type Test struct {
	Name   string
	Source io.Reader
	Target io.Reader
}

func main() {
	tests, err := readTests()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Running %d test(s)\n", len(tests))
	for _, test := range tests {
		if err := runTest(test); err != nil {
			fmt.Println(err)
		}
	}
}

func runTest(test Test) error {
	fmt.Printf("Running test '%s'\n", test.Name)

	fmt.Println("Reading source")
	source, err := io.ReadAll(test.Source)
	if err != nil {
		return err
	}

	fmt.Println("Unmarshaling source")
	var spec tdl.Spec
	if err = yaml.Unmarshal(source, &spec); err != nil {
		return err
	}

	if binDir == "" {
		return errors.New("BIN_DIR not found")
	}

	bin := path.Join(binDir, "um")
	cmd, err := runner.NewCli(bin, runner.WithArgs("ts"))
	if err != nil {
		return err
	}

	fmt.Println("Running generator")
	buf := &bytes.Buffer{}
	if err = cmd.Gen(context.TODO(), &spec, buf); err != nil {
		return err
	}

	fmt.Println("Reading target")
	expectedBytes, err := io.ReadAll(test.Target)
	if err != nil {
		return err
	}

	fmt.Println("Reading actual from buffer")
	actualBytes, err := io.ReadAll(buf)
	if err != nil {
		return err
	}

	expected := string(expectedBytes)
	actual := string(actualBytes)
	if actual != expected {
		fmt.Printf("Expected: %s\n", expected)
		fmt.Printf("Actual:   %s\n", actual)
		return errors.New("output did not match")
	}

	return nil
}

func readTests() ([]Test, error) {
	tests := []Test{}
	builder := NewTestBuilder()
	err := fs.WalkDir(testdata, "testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		switch strings.Count(path, "/") {
		case 1:
			builder.WithName(d.Name())
		case 2:
			file := d.Name()
			matches := matcher.FindStringSubmatch(file)
			i := matcher.SubexpIndex("name")

			reader, err := os.Open(path)
			if err != nil {
				return err
			}

			switch matches[i] {
			case "source":
				builder.WithSource(reader)
			case "target":
				builder.WithTarget(reader)
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
