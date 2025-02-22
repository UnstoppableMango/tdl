package lang_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/unmango/go/vcs/git"
)

var _ = Describe("E2e", func() {
	var ses *gexec.Session
	var sock string

	BeforeEach(func(ctx context.Context) {
		root, err := git.Root(ctx)
		Expect(err).NotTo(HaveOccurred())

		tmp := GinkgoT().TempDir()
		sock = filepath.Join(tmp, "tdl.sock")

		bin := filepath.Join(root, "bin", "lang-host")
		cmd := exec.CommandContext(ctx, bin, sock)

		By("Starting the host")
		ses, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the host to start")
		Eventually(ses.Out).Should(gbytes.Say("Application started"))
	})

	It("should work", func() {
		time.Sleep(time.Second * 5)
		Expect(true).To(BeTrue())
	})

	AfterEach(func() {
		By("Terminating the host")
		Eventually(ses.Interrupt()).Should(gexec.Exit(0))
	})
})
