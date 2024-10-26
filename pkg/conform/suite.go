package conform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/protobuf/proto"
)

func CliTests(description, binary string, args []string) {
	execArgs := append(args, "--conformance-test")

	ginkgo.Describe(fmt.Sprintf("%s: CLI: %s %s", description, binary, args), func() {
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
