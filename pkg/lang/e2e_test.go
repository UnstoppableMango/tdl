package lang_test

import (
	"context"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/unmango/go/vcs/git"
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("E2e", func() {
	var ses *gexec.Session
	var client tdlv1alpha1.ParserClient

	BeforeEach(func(ctx context.Context) {
		root, err := git.Root(ctx)
		Expect(err).NotTo(HaveOccurred())

		tmp := GinkgoT().TempDir()
		sock := filepath.Join(tmp, "tdl.sock")

		bin := filepath.Join(root, "bin", "lang-host")
		cmd := exec.CommandContext(ctx, bin, sock)

		By("Starting the host")
		ses, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the host to start")
		Eventually(ses.Out).Should(gbytes.Say("Application started"))

		By("Creating a parser client")
		conn, err := grpc.NewClient("unix:"+sock,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		Expect(err).NotTo(HaveOccurred())
		client = tdlv1alpha1.NewParserClient(conn)
	})

	It("should work", func(ctx context.Context) {
		By("Sending a parse request")
		res, err := client.Parse(ctx, &tdlv1alpha1.ParseRequest{
			Data: "type Test",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(res.Result.GetTypes()).To(HaveKeyWithValue("Test", &tdlv1alpha1.Type{}))
	})

	AfterEach(func() {
		By("Terminating the host")
		Eventually(ses.Interrupt()).Should(gexec.Exit())
	})
})
