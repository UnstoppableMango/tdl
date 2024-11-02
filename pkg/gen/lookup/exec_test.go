package lookup_test

import (
	"bytes"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
)

var _ = Describe("Command", func() {
	It("should lookup tool from PATH", func() {
		buf := &bytes.Buffer{}
		err := lookup.Execute("uml2ts", buf)

		Expect(err).NotTo(HaveOccurred())
		Expect(buf.String()).To(Equal(filepath.Join(gitRoot, "bin", "uml2ts") + "\n"))
	})
})
