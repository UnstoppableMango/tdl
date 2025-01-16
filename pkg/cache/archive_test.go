package cache_test

import (
	"io/fs"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/unmango/aferox/testing/gfs"

	"github.com/unstoppablemango/tdl/pkg/cache"
)

var _ = Describe("Archive", func() {
	var archive fs.File

	BeforeEach(func() {
		var err error
		archive, err = testdata.Open("testdata/tdl-linux-amd64.tar.gz")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should extract tar", func() {
		b := cache.NewMemFs()

		err := cache.StoreTarGz(b, archive)

		Expect(err).NotTo(HaveOccurred())
		Expect(b).To(ContainFile("uml2ts"))
		Expect(b).To(ContainFile("ux"))
	})
})
