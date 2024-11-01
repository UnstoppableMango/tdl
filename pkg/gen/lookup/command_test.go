package lookup_test

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
)

var _ = Describe("Command", func() {
	var gitRoot string

	BeforeEach(func(ctx context.Context) {
		revParse, err := exec.CommandContext(ctx,
			"git", "rev-parse", "--show-toplevel",
		).CombinedOutput()
		Expect(err).NotTo(HaveOccurred())

		gitRoot = strings.TrimSpace(string(revParse))
	})

	It("should lookup tool from PATH", func() {
		buf := &bytes.Buffer{}
		err := lookup.Execute("uml2ts", buf)

		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal(filepath.Join(gitRoot, "bin", "uml2ts") + "\n"))
	})
})
