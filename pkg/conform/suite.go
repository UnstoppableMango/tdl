package conform

import (
	"context"
	"os"
	"os/exec"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
)

func CliTests(description, binary string, args []string) {
	ginkgo.Describe(description, func() {
		ginkgo.It("should stat", func() {
			_, err := os.Stat(binary)

			g.Expect(err).NotTo(g.HaveOccurred())
		})

		ginkgo.It("should execute", func(ctx context.Context) {
			cmd := exec.CommandContext(ctx, binary, args...)
			out, err := cmd.CombinedOutput()

			g.Expect(err).NotTo(g.HaveOccurred(), string(out))
			g.Expect(string(out)).To(g.BeEmpty())
		})
	})
}
