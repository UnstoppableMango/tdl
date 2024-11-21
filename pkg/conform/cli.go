package conform

import (
	"context"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"

	"github.com/unmango/go/iter"
	"github.com/unmango/go/option"
	"github.com/unmango/go/slices"
)

type CliTestOptions struct {
	suites iter.Seq[Suite]
	args   []string
}

type CliTestOption func(*CliTestOptions)

// DescribeCli describes TDL conformace tests for binary runners.
// The provided binary must exist and be executable.
// Args can be provided if the codegen functionality is provided by a subcommand.
// [DescribeCli] MUST be called within the Ginkgo test construction phase.
func DescribeCli(binary string, options ...CliTestOption) {
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
		generator := cli.New(binary,
			cli.WithArgs(opts.args...),
		)

		for s := range opts.suites {
			s.ConstructTestsFor(generator)
		}
	})
}

func WithArgs(args ...string) CliTestOption {
	return func(opts *CliTestOptions) {
		opts.args = append(opts.args, args...)
	}
}

func WithSuites(suites ...Suite) CliTestOption {
	return func(opts *CliTestOptions) {
		opts.suites = slices.Values(suites)
	}
}
