package progress_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/progress"
)

var _ = Describe("Reader", func() {
	var data []byte

	BeforeEach(func() {
		data = make([]byte, 1024)
	})

	It("should work", func() {
		var sentinel bool
		buf := bytes.NewBuffer(data)
		r := progress.NewReader(buf)

		s := progress.Subscribe(r, func(i int, err error) {
			sentinel = true
		})
		defer s()

		_, err := io.ReadAll(r)
		Expect(err).NotTo(HaveOccurred())
		Expect(sentinel).To(BeTrueBecause("report was called"))
	})
})
