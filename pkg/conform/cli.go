package conform

import (
	"bytes"
	"context"
	"os"
	"os/exec"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
)

// CliTests describes TDL conformace tests for binary runners.
// The provided binary must exist and be executable.
// Args can be provided if the codegen functionality is provided by a subcommand.
// CliTests MUST be called within the Ginkgo test construction phase.
func CliTests(description, binary string, args []string) {
	// Basically a naive check that the thing we're executing
	// is at least semi-aware of conformance tests
	execArgs := append(args, "--conformance-test")

	ginkgo.Describe(description, func() {
		ginkgo.It("should stat", func() {
			_, err := os.Stat(binary)

			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.It("should execute", func(ctx context.Context) {
			cmd := exec.CommandContext(ctx, binary, execArgs...)
			out, err := cmd.CombinedOutput()

			g.Expect(err).NotTo(g.HaveOccurred(), string(out))
			g.Expect(string(out)).To(g.BeEmpty())
		})

		ginkgo.It("should read conformance spec", func(ctx context.Context) {
			ginkgo.By("Marshalling a TDL spec")
			data, err := proto.Marshal(&tdlv1alpha1.Spec{})
			g.Expect(err).NotTo(g.HaveOccurred())

			cmd := exec.CommandContext(ctx, binary, execArgs...)
			cmd.Stdin = bytes.NewReader(data)
			out, err := cmd.CombinedOutput()

			g.Expect(err).NotTo(g.HaveOccurred(), string(out))
		})
	})
}
