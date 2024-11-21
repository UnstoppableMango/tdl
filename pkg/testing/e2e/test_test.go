package e2e_test

import (
	"path/filepath"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unmango/go/slices"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
)

var _ = Describe("Test", func() {

	DescribeTable("InputRegex",
		Entry(nil, "input.yml"),
		Entry(nil, "source.yml"),
		Entry(nil, "input.yaml"),
		func(name string) {
			match := e2e.InputRegex.MatchString(name)

			Expect(match).To(BeTrueBecause("the regex matches"))
		},
	)

	DescribeTable("OutputRegex",
		Entry(nil, "output.ts"),
		Entry(nil, "target.ts"),
		Entry(nil, "output.cs"),
		Entry(nil, "target.fs"),
		Entry(nil, "output.go"),
		Entry(nil, "path/output.go"),
		func(name string) {
			match := e2e.OutputRegex.MatchString(name)

			Expect(match).To(BeTrueBecause("the regex matches"))
		},
	)

	Describe("ListTests", func() {
		It("should work", func() {
			path := filepath.Join("testdata", "list")

			tests, err := e2e.ListTests(testfs, path)

			Expect(err).NotTo(HaveOccurred())
			actual := slices.Collect(tests)
			Expect(actual).To(HaveLen(2))
		})
	})

	Describe("ReadTest", func() {
		It("should work", func() {
			path := filepath.Join("testdata", "test")

			test, err := e2e.ReadTest(testfs, path)

			Expect(err).NotTo(HaveOccurred())
			Expect(test.Name).To(Equal("test"))
			Expect(test.Spec).NotTo(BeNil())
			Expect(test.Spec.Name).To(Equal("ReadTest"))
			Expect(test.Expected).To(ContainFile("output.fs"))
		})

		It("should fail when no output exists", func() {
			path := filepath.Join("testdata", "no_output")

			_, err := e2e.ReadTest(testfs, path)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FindInput", func() {
		DescribeTable("valid input",
			Entry(nil, "testdata/input", "input.yml"),
			Entry(nil, "testdata/source", "source.yml"),
			Entry(nil, "testdata/yaml", "input.yaml"),
			func(path, file string) {
				log.SetLevel(log.DebugLevel)
				input, err := e2e.FindInput(testfs, path)

				Expect(err).NotTo(HaveOccurred())
				Expect(input).To(Equal(file))
			},
		)
	})
})
