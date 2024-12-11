package crd2pulumi_test

import (
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/tool/crd2pulumi"
)

var _ = Describe("Builder", func() {
	Describe("ForceOpt", func() {
		It("should not mutate the builder", func() {
			fn := func() bool {
				b := crd2pulumi.ArgBuilder{}
				_ = b.ForceOpt()
				return len(b) == 0
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})

		It("should add force option", func() {
			b := crd2pulumi.ArgBuilder{}

			n := b.ForceOpt()

			Expect(n).To(ConsistOf("--force"))
		})

		It("should append to existing args", func() {
			b := crd2pulumi.ArgBuilder{"blah"}

			n := b.ForceOpt()

			Expect(n).To(ConsistOf("blah", "--force"))
		})
	})

	Describe("LangOpt", func() {
		It("should not mutate the builder", func() {
			fn := func(lang string) bool {
				b := crd2pulumi.ArgBuilder{}
				_ = b.LangOpt(lang)
				return len(b) == 0
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})

		It("should add lang option", func() {
			b := crd2pulumi.ArgBuilder{}

			n := b.LangOpt("nodejs")

			Expect(n).To(ConsistOf("--nodejs"))
		})

		It("should append to existing args", func() {
			b := crd2pulumi.ArgBuilder{"blah"}

			n := b.LangOpt("nodejs")

			Expect(n).To(ConsistOf("blah", "--nodejs"))
		})
	})

	Describe("NameOpt", func() {
		It("should not mutate the builder", func() {
			fn := func(lang, name string) bool {
				b := crd2pulumi.ArgBuilder{}
				_ = b.NameOpt(lang, name)
				return len(b) == 0
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})

		It("should add name option", func() {
			b := crd2pulumi.ArgBuilder{}

			n := b.NameOpt("nodejs", "my-name")

			Expect(n).To(ConsistOf("--nodejsName", "my-name"))
		})

		It("should append to existing args", func() {
			b := crd2pulumi.ArgBuilder{"blah"}

			n := b.NameOpt("nodejs", "my-name")

			Expect(n).To(ConsistOf("blah", "--nodejsName", "my-name"))
		})
	})

	Describe("PathOpt", func() {
		It("should not mutate the builder", func() {
			fn := func(lang, path string) bool {
				b := crd2pulumi.ArgBuilder{}
				_ = b.PathOpt(lang, path)
				return len(b) == 0
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})

		It("should add path option", func() {
			b := crd2pulumi.ArgBuilder{}

			n := b.PathOpt("nodejs", "/some/path")

			Expect(n).To(ConsistOf("--nodejsPath", "/some/path"))
		})

		It("should append to existing args", func() {
			b := crd2pulumi.ArgBuilder{"blah"}

			n := b.PathOpt("nodejs", "/some/path")

			Expect(n).To(ConsistOf("blah", "--nodejsPath", "/some/path"))
		})
	})

	Describe("VersionOpt", func() {
		It("should not mutate the builder", func() {
			fn := func(version string) bool {
				b := crd2pulumi.ArgBuilder{}
				_ = b.VersionOpt(version)
				return len(b) == 0
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})

		It("should add force option", func() {
			b := crd2pulumi.ArgBuilder{}

			n := b.VersionOpt("v0.0.69")

			Expect(n).To(ConsistOf("--version", "v0.0.69"))
		})

		It("should append to existing args", func() {
			b := crd2pulumi.ArgBuilder{"blah"}

			n := b.VersionOpt("v0.0.69")

			Expect(n).To(ConsistOf("blah", "--version", "v0.0.69"))
		})
	})
})
