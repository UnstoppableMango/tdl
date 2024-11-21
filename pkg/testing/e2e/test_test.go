package e2e_test

import (
	"path/filepath"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
)

var _ = Describe("Test", func() {
	It("should work", func() {
		path := filepath.Join("testdata", "test")

		test, err := e2e.ReadTest(testfs, path)

		Expect(err).NotTo(HaveOccurred())
		Expect(test.Name).To(Equal("test"))
		Expect(test.Spec).NotTo(BeNil()) // TODO
		Expect(test.Expected).To(ContainFile("output.fs"))
	})

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
