package lookup_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen/lookup"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var _ = Describe("Path", func() {
	It("should return tool from PATH", func() {
		path, err := lookup.FromPath(tdl.Token{Name: "go"})

		Expect(err).NotTo(HaveOccurred())
		Expect(path).NotTo(Equal("go"))
	})

	It("should fail when tool is not on PATH", func() {
		_, err := lookup.FromPath(tdl.Token{Name: "fjdksfljdkfdlkfjsldfklsdlfksj"})

		Expect(err).To(HaveOccurred())
	})
})
