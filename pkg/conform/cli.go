package conform

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
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
	// Basically a naive check that the thing we're executing
	// is at least semi-aware of conformance tests
	options = append(options, WithArgs("--conformance-test"))

	opts := &CliTestOptions{}
	option.Apply(opts, options...)

	ginkgo.It("should stat", func() {
		_, err := os.Stat(binary)

		g.Expect(err).NotTo(g.HaveOccurred())
	})

	ginkgo.It("should execute", func(ctx context.Context) {
		cmd := exec.CommandContext(ctx, binary, opts.args...)
		out, err := cmd.CombinedOutput()

		g.Expect(err).NotTo(g.HaveOccurred(), string(out))
		g.Expect(string(out)).To(g.BeEmpty())
	})

	ginkgo.Describe("Generator", func() {
		var generator tdl.Generator

		ginkgo.BeforeEach(func() {
			generator = gen.NewCli(binary,
				gen.WithCliArgs(opts.args...),
			)
		})

		ginkgo.It("should read conformance spec", func(ctx context.Context) {
			ginkgo.By("Marshalling a TDL spec")
			data, err := proto.Marshal(&tdlv1alpha1.Spec{})
			g.Expect(err).NotTo(g.HaveOccurred())

			cmd := exec.CommandContext(ctx, binary, opts.args...)
			cmd.Stdin = bytes.NewReader(data)
			out, err := cmd.CombinedOutput()

			g.Expect(err).NotTo(g.HaveOccurred(), string(out))
		})

		if opts.ioSuite != nil {
			IOSuite(opts.ioSuite, gen.PipeIO(generator))
		}
	})
}

func WithArgs(args ...string) CliTestOption {
	return func(cto *CliTestOptions) {
		cto.args = append(cto.args, args...)
	}
}
