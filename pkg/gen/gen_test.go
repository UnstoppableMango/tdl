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
	var output *string = nil

	var sut gen.GeneratorFunc[string, string] = func(ctx context.Context, s string, i string) error {
		input = &s
		output = &i
		return nil
	}

	mapper := func(reader io.Reader) result.R[string] {
		buf := new(strings.Builder)
		if _, err := io.Copy(buf, reader); err != nil {
			return result.OfErr[string](err)
		}

		return result.Of(buf.String())
	}

	BeforeEach(func() {
		input = nil
		output = nil
	})

	Describe("MapI", func() {
		It("should work", func(ctx context.Context) {
			expectedInput := "blah blah blah"
			expectedOutput := "eh doesn't matter"
			reader := bytes.NewBufferString(expectedInput)

			mapped := gen.MapI(sut, mapper)
			err := mapped(ctx, reader, expectedOutput)

			Expect(err).NotTo(HaveOccurred())
			Expect(input).NotTo(BeNil())
			Expect(*input).To(Equal(expectedInput))
			Expect(output).NotTo(BeNil())
			Expect(*output).To(Equal(expectedOutput))
		})
	})

	Describe("MapO", func() {
		It("should work", func(ctx context.Context) {
			expectedInput := "blah blah blah"
			expectedOutput := "eh doesn't matter"
			reader := bytes.NewBufferString(expectedOutput)

			mapped := gen.MapO(sut, mapper)
			err := mapped(ctx, expectedInput, reader)

			Expect(err).NotTo(HaveOccurred())
			Expect(input).NotTo(BeNil())
			Expect(*input).To(Equal(expectedInput))
			Expect(output).NotTo(BeNil())
			Expect(*output).To(Equal(expectedOutput))
		})
	})
})
