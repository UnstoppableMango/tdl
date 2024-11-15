package conform

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unmango/go/option"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/testing"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
)

type CliTestOptions struct {
	ioSuite []*testing.Test
	args    []string
}

type CliTestOption func(*CliTestOptions)

// CliTests describes TDL conformace tests for binary runners.
// The provided binary must exist and be executable.
// Args can be provided if the codegen functionality is provided by a subcommand.
// CliTests MUST be called within the Ginkgo test construction phase.
func CliTests(binary string, options ...CliTestOption) {
	opts := &CliTestOptions{}
	option.Apply(opts, options...)

	It("should stat", func() {
		_, err := os.Stat(binary)

		Expect(err).NotTo(HaveOccurred())
	})

	It("should execute", func(ctx context.Context) {
		cmd := exec.CommandContext(ctx, binary, opts.args...)
		out, err := cmd.CombinedOutput()

		Expect(err).NotTo(HaveOccurred(), string(out))
		Expect(string(out)).To(BeEmpty())
	})

	Describe("Generator", func() {
		var generator tdl.Generator

		BeforeEach(func() {
			generator = gen.NewCli(binary,
				gen.WithCliArgs(opts.args...),
			)
		})

		It("should read conformance spec", func(ctx context.Context) {
			data, err := proto.Marshal(&tdlv1alpha1.Spec{})
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.CommandContext(ctx, binary, opts.args...)
			cmd.Stdin = bytes.NewReader(data)
			out, err := cmd.CombinedOutput()

			Expect(err).NotTo(HaveOccurred(), string(out))
		})

		if len(opts.ioSuite) > 0 {
			IOSuite(opts.ioSuite, gen.PipeIO(generator))
		}
	})
}

func WithArgs(args ...string) CliTestOption {
	return func(opts *CliTestOptions) {
		opts.args = append(opts.args, args...)
	}
}

func WithIOTests(tests ...*testing.Test) CliTestOption {
	return func(opts *CliTestOptions) {
		opts.ioSuite = tests
	}
}
