package cache_test

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/unmango/go/testing/matcher"

	"github.com/unstoppablemango/tdl/pkg/cache"
)

var _ = Describe("Archive", func() {
	var archive io.Reader

	BeforeEach(func() {
		var err error
		archive, err = testdata.Open("testdata/tdl-linux-amd64.tar.gz")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should extract tar", func() {
		b := cache.NewMemFs()

		err := cache.ExtractTar(b, "tdl-linux-amd64.tar.gz", archive)

		Expect(err).NotTo(HaveOccurred())
		Expect(b).To(ContainFile("uml2ts"))
		Expect(b).To(ContainFile("ux"))
	})
})
