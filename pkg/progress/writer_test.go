package progress_test

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/progress"
)

var _ = Describe("Writer", func() {
	var data []byte

	BeforeEach(func() {
		data = make([]byte, 2048)
	})

	It("should work", func() {
		values := []float64{}
		errors := []error{}
		sink := bytes.NewBuffer(make([]byte, 512))
		w := progress.NewWriter(sink, len(data))

		s := progress.Subscribe(w, func(i float64, err error) {
			values = append(values, i)
			errors = append(errors, err)
		})
		defer s()

		for i := 0; i < 4; i++ {
			_, err := w.Write(make([]byte, 512))
			Expect(err).NotTo(HaveOccurred())
		}

		Expect(values).To(ConsistOf(0.25, 0.5, 0.75, 1.0))
		Expect(errors).To(ConsistOf(nil, nil, nil, nil))
	})
})
