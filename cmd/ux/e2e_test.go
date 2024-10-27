package main_test

import (
	"context"
	"errors"
	"io"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/conform"
)

var _ = Describe("End to end", func() {
	Describe("CLI Conformance", func() {
		conform.CliTests(bin, []string{"gen", "ts"})
	})

	conform.IOSuite("TypeScript Conformance", typescriptSuite, ExecuteIO)

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

	cmd := exec.Command(bin, "gen", "ts")
	cmd.Stdin = input
	cmd.Stdout = output
	cmd.Stderr = output

	return cmd.Run()
}
