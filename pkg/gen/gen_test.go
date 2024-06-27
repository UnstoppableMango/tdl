package gen_test

import (
	"bytes"
	"context"
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/result"
)

var _ = Describe("Gen", func() {
	var input *string = nil
	var output *int = nil

	var sut gen.GeneratorFunc[string, int] = func(ctx context.Context, s string, i int) error {
		input = &s
		output = &i
		return nil
	}

	BeforeEach(func() {
		input = nil
		output = nil
	})

	Describe("MapI", func() {
		It("should work", func() {
			expected := "blah blah blah"

			mapped := gen.MapI(sut, func(reader io.Reader) result.R[string] {
				buf := new(strings.Builder)
				if _, err := io.Copy(buf, reader); err != nil {
					return result.OfErr[string](err)
				}

				return result.Of(buf.String())
			})

			buf := bytes.NewBufferString(expected)

			err := mapped(context.TODO(), buf, 69)

			Expect(err).NotTo(HaveOccurred())
			Expect(input).NotTo(BeNil())
			Expect(*input).To(Equal(expected))
			Expect(output).NotTo(BeNil())
			Expect(*output).To(Equal(69))
		})
	})
})
