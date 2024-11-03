package gen_test

import (
	"errors"
	"path/filepath"
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/gen"
	"github.com/unstoppablemango/tdl/pkg/tdl"
)

var _ = Describe("Lookup", func() {
	Describe("Name", func() {
		It("should return tool from PATH", func() {
			path, err := gen.FromPath(tdl.Token{Name: "go"})

			Expect(err).NotTo(HaveOccurred())
			Expect(path).NotTo(Equal("go"))
		})

		It("should fail when tool is not on PATH", func() {
			_, err := gen.FromPath(tdl.Token{Name: "fjdksfljdkfdlkfjsldfklsdlfksj"})

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Lookup", func() {
		It("should return generator for ts", func() {
			expected := filepath.Join(gitRoot, "bin", "uml2ts")

			res, err := gen.Lookup("ts")

			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			cli, ok := res.(*gen.Cli)
			Expect(ok).To(BeTrueBecause("ts is a Cli generator"))
			Expect(cli.String()).To(Equal(expected))
		})

		It("should return generator for uml2ts", func() {
			expected := filepath.Join(gitRoot, "bin", "uml2ts")

			res, err := gen.Lookup("uml2ts")

			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
			cli, ok := res.(*gen.Cli)
			Expect(ok).To(BeTrueBecause("uml2ts is a Cli generator"))
			Expect(cli.String()).To(Equal(expected))
		})

		Describe("Name", func() {
			It("should return generator for ts", func() {
				expected := filepath.Join(gitRoot, "bin", "uml2ts")
				token := tdl.Token{Name: "ts"}

				res, err := gen.Name(token)

				Expect(err).NotTo(HaveOccurred())
				Expect(res).NotTo(BeNil())
				cli, ok := res.(*gen.Cli)
				Expect(ok).To(BeTrueBecause("ts is a Cli generator"))
				Expect(cli.String()).To(Equal(expected))
			})

			It("should return generator for uml2ts", func() {
				expected := filepath.Join(gitRoot, "bin", "uml2ts")
				token := tdl.Token{Name: "uml2ts"}

				res, err := gen.Name(token)

				Expect(err).NotTo(HaveOccurred())
				Expect(res).NotTo(BeNil())
				cli, ok := res.(*gen.Cli)
				Expect(ok).To(BeTrueBecause("uml2ts is a Cli generator"))
				Expect(cli.String()).To(Equal(expected))
			})

			It("should return not found for other names", func() {
				fn := func(name string) bool {
					token := tdl.Token{Name: name}

					_, err := gen.Name(token)

					return errors.Is(err, gen.ErrNotFound)
				}

				Expect(quick.Check(fn, nil)).To(Succeed())
			})
		})
	})
})
