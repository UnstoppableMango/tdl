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
		values := []float64{}
		errors := []error{}
		buf := bytes.NewBuffer(data)
		r := progress.NewReader(buf, len(data))

		s := progress.Subscribe(r, func(i float64, err error) {
			values = append(values, i)
			errors = append(errors, err)
		})
		defer s()

		_, err := io.ReadAll(r)
		Expect(err).NotTo(HaveOccurred())
		Expect(values).To(ConsistOf(0.5, 0.875, 1.0))
		Expect(errors).To(ConsistOf(nil, nil, nil))
	})
})
