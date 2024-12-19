package crd2pulumi_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/tool/crd2pulumi"
)

var _ = Describe("Options", func() {

	Describe("Paths", func() {
		DescribeTable("should generate paths relative to root",
			Entry("nodejs", "nodejs", "blah/nodejs"),
			Entry("python", "python", "blah/python"),
			Entry("dotnet", "dotnet", "blah/dotnet"),
			Entry("golang", "golang", "blah/golang"),
			Entry("java", "java", "blah/java"),
			func(key, value string) {
				root := "blah"
				t := crd2pulumi.Options{
					NodeJS: crd2pulumi.LangOptions{},
					Python: crd2pulumi.LangOptions{},
					Dotnet: crd2pulumi.LangOptions{},
					Go:     crd2pulumi.LangOptions{},
					Java:   crd2pulumi.LangOptions{},
				}

				paths := t.Paths(root)

				Expect(paths).To(HaveKeyWithValue(key, value))
			},
		)

		DescribeTable("should use the path from options",
			Entry("nodejs", "nodejs", "blah/nodejs"),
			Entry("python", "python", "blah/python"),
			Entry("dotnet", "dotnet", "blah/dotnet"),
			Entry("golang", "golang", "blah/golang"),
			Entry("java", "java", "blah/java"),
			func(key, value string) {
				t := crd2pulumi.Options{
					NodeJS: crd2pulumi.LangOptions{Path: value},
					Python: crd2pulumi.LangOptions{Path: value},
					Dotnet: crd2pulumi.LangOptions{Path: value},
					Go:     crd2pulumi.LangOptions{Path: value},
					Java:   crd2pulumi.LangOptions{Path: value},
				}

				paths := t.Paths("doesn't matter")

				Expect(paths).To(HaveKeyWithValue(key, value))
			},
		)

		It("should include a path when an option is enabled", func() {
			t := crd2pulumi.Options{
				NodeJS: crd2pulumi.LangOptions{Enabled: true},
				Python: crd2pulumi.LangOptions{},
				Dotnet: crd2pulumi.LangOptions{},
				Go:     crd2pulumi.LangOptions{},
				Java:   crd2pulumi.LangOptions{},
			}

			paths := t.Paths("/root")

			Expect(paths).To(HaveKeyWithValue("nodejs", "/root/nodejs"))
		})
	})

	Describe("Args", func() {
		It("should work", func() {
			t := crd2pulumi.Options{
				NodeJS:  crd2pulumi.LangOptions{Path: "doesn't matter, the path is take from paths"},
				Python:  crd2pulumi.LangOptions{Name: "peethon"},
				Dotnet:  crd2pulumi.LangOptions{Enabled: true},
				Go:      crd2pulumi.LangOptions{},
				Java:    crd2pulumi.LangOptions{},
				Force:   true,
				Version: "v420",
			}
			paths := map[string]string{
				"nodejs": "/test",
				"dotnet": "/dotnet",
			}

			args := t.Args(paths)

			Expect(args).To(ConsistOf(
				"--nodejsPath", "/test",
				"--pythonName", "peethon",
				"--dotnet",
				"--dotnetPath", "/dotnet",
				"--force",
				"--version", "v420",
			))
		})
	})

	Describe("Parse", func() {
		It("should return empty options", func() {
			o, err := crd2pulumi.Parse([]string{})

			Expect(err).NotTo(HaveOccurred())
			Expect(o.Dotnet.Enabled).To(BeFalse())
			Expect(o.Dotnet.Name).To(BeEmpty())
			Expect(o.Dotnet.Path).To(BeEmpty())
			Expect(o.Go.Enabled).To(BeFalse())
			Expect(o.Go.Name).To(BeEmpty())
			Expect(o.Go.Path).To(BeEmpty())
			Expect(o.NodeJS.Enabled).To(BeFalse())
			Expect(o.NodeJS.Name).To(BeEmpty())
			Expect(o.NodeJS.Path).To(BeEmpty())
			Expect(o.Python.Enabled).To(BeFalse())
			Expect(o.Python.Name).To(BeEmpty())
			Expect(o.Python.Path).To(BeEmpty())
			Expect(o.Force).To(BeFalse())
			Expect(o.Version).To(BeEmpty())
		})

		It("should enable dotnet options", func() {
			o, err := crd2pulumi.Parse([]string{
				"--dotnet",
				"--dotnetName", "bleh",
				"--dotnetPath", "blah",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(o.Dotnet.Enabled).To(BeTrue())
			Expect(o.Dotnet.Name).To(Equal("bleh"))
			Expect(o.Dotnet.Path).To(Equal("blah"))
		})
	})
})
