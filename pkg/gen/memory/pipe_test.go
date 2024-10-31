package memory_test

import (
	"bytes"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/gen/memory"
)

var _ = Describe("Pipe", func() {
	Describe("PorcelainPipe", func() {
		var pipe *memory.PorcelainPipe

		BeforeEach(func() {
			pipe = &memory.PorcelainPipe{}
		})

		It("should return the provided reader", func() {
			expected := &bytes.Buffer{}
			unit := "test-unit"
			Expect(pipe.WriteUnit(unit, expected)).To(Succeed())

			reader, err := pipe.Reader(unit)

			Expect(err).NotTo(HaveOccurred())
			Expect(reader).To(BeIdenticalTo(expected))
		})

		It("should error when reader does not exist", func() {
			_, err := pipe.Reader("does-not-exist")

			Expect(err).To(HaveOccurred())
		})

		It("should error when unit is not matched", func() {
			expected := &bytes.Buffer{}
			unit := "test-unit"
			Expect(pipe.WriteUnit(unit, expected)).To(Succeed())

			_, err := pipe.Reader("does-not-exist")

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("BufferedPipe", func() {
		var pipe *memory.BufferedPipe

		BeforeEach(func() {
			pipe = &memory.BufferedPipe{}
		})

		It("should not return the provided reader", func() {
			expected := &bytes.Buffer{}
			unit := "test-unit"
			Expect(pipe.WriteUnit(unit, expected)).To(Succeed())

			reader, err := pipe.Reader(unit)

			Expect(err).NotTo(HaveOccurred())
			Expect(reader).NotTo(BeIdenticalTo(expected))
		})

		It("should return a buffer connected to the provided reader", func() {
			input := &bytes.Buffer{}
			expected := "some content"
			_, err := input.WriteString(expected)
			Expect(err).NotTo(HaveOccurred())
			unit := "test-unit"
			Expect(pipe.WriteUnit(unit, input)).To(Succeed())

			reader, err := pipe.Reader(unit)

			Expect(err).NotTo(HaveOccurred())
			actual, err := io.ReadAll(reader)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(actual)).To(Equal(expected))
		})

		It("should error when reader does not exist", func() {
			_, err := pipe.Reader("does-not-exist")

			Expect(err).To(HaveOccurred())
		})

		It("should error when unit is not matched", func() {
			expected := &bytes.Buffer{}
			unit := "test-unit"
			Expect(pipe.WriteUnit(unit, expected)).To(Succeed())

			_, err := pipe.Reader("does-not-exist")

			Expect(err).To(HaveOccurred())
		})
	})
})
