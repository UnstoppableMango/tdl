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
		values := []int{}
		errors := []error{}
		buf := bytes.NewBuffer(data)
		r := progress.NewReader(buf)

		s := progress.Subscribe(r, func(i int, err error) {
			values = append(values, i)
			errors = append(errors, err)
		})
		defer s()

		_, err := io.ReadAll(r)
		Expect(err).NotTo(HaveOccurred())
		// I'm not sure why these values are used,
		// but they seem to be pretty consistent
		Expect(values).To(ConsistOf(512, 384, 128))
		Expect(errors).To(ConsistOf(nil, nil, nil))
	})
})
