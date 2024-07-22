package main_test

import (
	"bytes"
	"embed"
	"io/fs"
	"os"
	"os/exec"
	"path"
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
	binDir = os.Getenv("BIN_DIR")
)

var _ = DescribeTableSubtree("End to end",
	Entry("interface", "interface", "source.yml", "target.ts"),
	Entry("nested_interface", "nested_interface", "source.yml", "target.ts"),
	func(dir, sourceFile, targetFile string) {
		var source, target []byte

		BeforeEach(func() {
			if binDir == "" {
				Skip("BIN_DIR not found")
			}

			data, err := fs.ReadFile(testdata, path.Join("testdata", dir, sourceFile))
			Expect(err).NotTo(HaveOccurred())
			source = data

			data, err = fs.ReadFile(testdata, path.Join("testdata", dir, targetFile))
			Expect(err).NotTo(HaveOccurred())
			target = data
		})

		It("should generate", func() {
			By("Unmarshaling source")
			spec := &uml.Spec{}
			err := yaml.Unmarshal(source, spec)
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

			expected := strings.TrimSpace(string(target))
			actual := strings.TrimSpace(stdout.String())
			Expect(actual).To(Equal(expected))
		})
	},
)
