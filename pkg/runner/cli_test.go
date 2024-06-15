package runner

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/uml"
)

var (
	binDir = os.Getenv("BIN_DIR")
	bin    = path.Join(binDir, "um")
)

var _ = Describe("NewCli", func() {
	var runner uml.Runner
	var cliRunner *cli
	var err error

	BeforeEach(func() {
		runner, err = NewCli(bin)
		cliRunner = runner.(*cli)
	})

	It("should create a CLI runner", func() {
		Expect(err).To(BeNil())
		Expect(runner).NotTo(BeNil())
	})

	It("should use the provided path", func() {
		Expect(err).To(BeNil())
		Expect(cliRunner.Path).To(Equal(bin))
	})
})
