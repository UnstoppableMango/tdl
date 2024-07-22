package main_test

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/uml"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

//go:embed testdata/**
var testdata embed.FS

var (
	matcher = regexp.MustCompile(`(?P<name>\w*)\..*`)
	binDir  = os.Getenv("BIN_DIR")
)

type Test struct {
	Name   string
	Source io.Reader
	Target io.Reader
}

var _ = Describe("End to end", func() {
	if binDir == "" {
		Fail("BIN_DIR not found")
	}

	tests, err := readTests()
	if err != nil {
		Fail(err.Error())
	}

	for _, test := range tests {
		It(fmt.Sprintf("should generate %s", test.Name), func() {
			By("Reading source")
			source, err := io.ReadAll(test.Source)
			Expect(err).NotTo(HaveOccurred())

			By("Unmarshaling source")
			spec := &uml.Spec{}
			err = yaml.Unmarshal(source, spec)
			Expect(err).NotTo(HaveOccurred())

			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			data, err := proto.Marshal(spec)
			Expect(err).NotTo(HaveOccurred())

			bin := path.Join(binDir, "um")
			cmd := exec.Command(bin, "gen", "ts")
			cmd.Stdin = bytes.NewReader(data)
			cmd.Stdout = stdout
			cmd.Stderr = stderr

			By("Running generator")
			err = cmd.Run()
			Expect(err).NotTo(HaveOccurred())

			// if err = cmd.Run(); err != nil {
			// 	return errors.Join(err,
			// 		fmt.Errorf("stdout: %s", stdout.String()),
			// 		fmt.Errorf("sdterr: %s", stderr.String()),
			// 	)
			// }

			By("Reading target")
			expectedBytes, err := io.ReadAll(test.Target)
			Expect(err).NotTo(HaveOccurred())

			expected := strings.TrimSpace(string(expectedBytes))
			actual := strings.TrimSpace(stdout.String())
			Expect(actual).To(Equal(expected))
		})
	}
})

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
