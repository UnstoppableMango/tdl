package crd2pulumi_test

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	aferox "github.com/unmango/go/fs"

	"github.com/unstoppablemango/tdl/pkg/tool/crd2pulumi"
)

var _ = Describe("Tool", func() {
	var crdyaml []byte

	BeforeEach(func() {
		var err error
		crdyaml, err = testdata.ReadFile("testdata/objectbucket.io_objectbucketclaims.yaml")
		Expect(err).NotTo(HaveOccurred())
	})

	It("should execute", Pending, func(ctx context.Context) {
		t := crd2pulumi.Tool{
			NodeJS: &crd2pulumi.LangOptions{
				Enabled: true,
			},
		}
		fs := afero.NewMemMapFs()
		err := afero.WriteFile(fs, "blah.yaml", crdyaml, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		out, err := t.Execute(ctx, fs)

		Expect(err).NotTo(HaveOccurred())
		f, err := aferox.StatFirst(out, "")
		Expect(err).NotTo(HaveOccurred())
		Expect(f.Name()).To(BeEmpty())
		Expect(afero.IsEmpty(out, "")).To(BeTrueBecause("generates nothing"))
	})

	Describe("Paths", func() {
		DescribeTable("should generate paths relative to root",
			Entry("nodejs", "nodejs", "blah/nodejs"),
			Entry("python", "python", "blah/python"),
			Entry("dotnet", "dotnet", "blah/dotnet"),
			Entry("golang", "golang", "blah/golang"),
			Entry("java", "java", "blah/java"),
			func(key, value string) {
				root := "blah"
				t := crd2pulumi.Tool{
					NodeJS: &crd2pulumi.LangOptions{},
					Python: &crd2pulumi.LangOptions{},
					Dotnet: &crd2pulumi.LangOptions{},
					Go:     &crd2pulumi.LangOptions{},
					Java:   &crd2pulumi.LangOptions{},
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
				t := crd2pulumi.Tool{
					NodeJS: &crd2pulumi.LangOptions{Path: value},
					Python: &crd2pulumi.LangOptions{Path: value},
					Dotnet: &crd2pulumi.LangOptions{Path: value},
					Go:     &crd2pulumi.LangOptions{Path: value},
					Java:   &crd2pulumi.LangOptions{Path: value},
				}

				paths := t.Paths("doesn't matter")

				Expect(paths).To(HaveKeyWithValue(key, value))
			},
		)

		It("should exclude nil options", func() {
			t := crd2pulumi.Tool{
				NodeJS: &crd2pulumi.LangOptions{},
				Python: nil,
				Dotnet: nil,
				Go:     nil,
				Java:   nil,
			}

			paths := t.Paths("doesn't matter")

			Expect(paths).To(HaveKey("nodejs"))
			Expect(paths).NotTo(HaveKey("python"))
			Expect(paths).NotTo(HaveKey("dotnet"))
			Expect(paths).NotTo(HaveKey("golang"))
			Expect(paths).NotTo(HaveKey("java"))
		})
	})

	Describe("Args", func() {
		It("should work", func() {
			t := crd2pulumi.Tool{
				NodeJS:  &crd2pulumi.LangOptions{Path: "doesn't matter, the path is take from paths"},
				Python:  &crd2pulumi.LangOptions{Name: "peethon"},
				Dotnet:  &crd2pulumi.LangOptions{Enabled: true},
				Go:      &crd2pulumi.LangOptions{},
				Java:    &crd2pulumi.LangOptions{},
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
})
