package conform

import (
	"os"

	"github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
)

func CliTests(binary string) {
	ginkgo.Describe("CLI Tests", func() {
		ginkgo.It("should stat", func() {
			_, err := os.Stat(binary)
			g.Expect(err).NotTo(g.HaveOccurred())
		})
	})
}
