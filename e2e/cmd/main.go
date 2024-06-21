package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/**
var testdata embed.FS

var (
	matcher = regexp.MustCompile(`(?P<name>\w*)\..*`)
	binDir  = os.Getenv("BIN_DIR")
	logger  = slog.New(slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{Level: slog.LevelDebug}),
	)
)

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

	logger.Info("Running test(s)", "num", len(tests))
	for _, test := range tests {
		if err := runTest(test); err != nil {
			logger.Error("test failed", "err", err)
		}
	}
}

func runTest(test Test) error {
	logger := logger.With("name", test.Name)

	logger.Info("Reading source")
	source, err := io.ReadAll(test.Source)
	if err != nil {
		return err
	}

	logger.Info("Unmarshaling source")
	spec := &uml.Spec{}
	if err = yaml.Unmarshal(source, spec); err != nil {
		return err
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	data, err := proto.Marshal(spec)
	if err != nil {
		return err
	}

	if binDir == "" {
		return errors.New("BIN_DIR not found")
	}

	bin := path.Join(binDir, "um")
	cmd := exec.Command(bin, "gen", "ts")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	logger.Info("Running generator")
	if err = cmd.Run(); err != nil {
		return errors.Join(err,
			fmt.Errorf("stdout: %s", stdout.String()),
			fmt.Errorf("sdterr: %s", stderr.String()),
		)
	}

	logger.Info("Reading target")
	expectedBytes, err := io.ReadAll(test.Target)
	if err != nil {
		return err
	}

	expected := strings.TrimSpace(string(expectedBytes))
	actual := strings.TrimSpace(stdout.String())
	if actual != expected {
		fmt.Printf("Expected:\n%s", expected)
		fmt.Printf("Actual:\n%s", actual)
		return errors.New("output did not match")
	}

	logger.Info("Success! ✅")
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
