package main_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/conform"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

var _ = Describe("End to end", Pending, func() {
	Describe("CLI Conformance", func() {
		conform.CliTests(bin, []string{"gen", "ts"})
	})

	Describe("TypeScript Conformance", FlakeAttempts(5), func() {
		conform.IOSuite(typescriptSuite, ExecuteIO)
	})

	It("should pass my excessive sanity check", func() {
		Expect(bin).NotTo(BeEmpty())
	})

	It("should execute", func(ctx context.Context) {
		err := exec.CommandContext(ctx, bin).Run()

		Expect(err).NotTo(HaveOccurred())
	})
})

func ExecuteIO(input io.Reader, output io.Writer) error {
	if bin == "" {
		return errors.New("test has not been initialized: bin was empty")
	}
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
