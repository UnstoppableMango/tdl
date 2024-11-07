package main_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/conform"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

var _ = Describe("End to end", func() {
	Describe("CLI Conformance", func() {
		conform.CliTests(bin,
			conform.WithArgs("gen", "ts"),
		)
	})

	Describe("TypeScript Conformance", FlakeAttempts(5), func() {
		conform.IOSuite(typescriptSuite, ExecuteIO)
	})

	It("should pass my excessive sanity check", func() {
		Expect(bin).NotTo(BeEmpty())
	})

	Describe("gen", func() {
		It("should read spec from yaml file", func(ctx context.Context) {
			input := filepath.Join(tsSuiteRoot, "interface", "source.yml")
			output, err := os.ReadFile(filepath.Join(tsSuiteRoot, "interface", "target.ts"))
			Expect(err).NotTo(HaveOccurred())
			cmd := UxCommand(ctx, "gen", "ts", input)

			out, err := cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred(), string(out))
			Expect(string(out)).To(Equal(string(output)))
		})
	})

	Describe("which", func() {
		DescribeTable("uml2ts",
			Entry(nil, "ts"),
			Entry(nil, "typescript"),
			Entry(nil, "uml2ts"),
			Entry(nil, "TypeScript"),
			Entry(nil, "tS"),
			func(ctx context.Context, input string) {
				expected := filepath.Join(gitRoot, "bin", "uml2ts")
				cmd := UxCommand(ctx, "which", input)

				out, err := cmd.CombinedOutput()

				Expect(err).NotTo(HaveOccurred())
				Expect(string(out)).To(Equal(expected + "\n"))
			},
		)
	})
})

func ExecuteIO(input io.Reader, output io.Writer) error {
	data, err := io.ReadAll(input)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	var spec tdlv1alpha1.Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return fmt.Errorf("reading spec: %w", err)
	}

	protoInput, err := proto.Marshal(&spec)
	if err != nil {
		return fmt.Errorf("marshalling spec: %w", err)
	}

	cmd := exec.Command(bin, "gen", "ts")
	cmd.Stdin = bytes.NewReader(protoInput)
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}
