package crd2pulumi_test

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	. "github.com/unmango/go/testing/matcher"

	"github.com/unstoppablemango/tdl/pkg/tool/crd2pulumi"
)

var _ = Describe("Tool", func() {
	AfterEach(func() {
		log.SetLevel(log.InfoLevel)
	})

	DescribeTable("should match yaml files",
		Entry(nil, "blah.yaml"),
		Entry(nil, "path/blah.yaml"),
		Entry(nil, "blah.yml"),
		Entry(nil, "path/blah.yml"),
		func(input string) {
			match := crd2pulumi.CrdRegex.MatchString(input)

			Expect(match).To(BeTrue())
		},
	)

	Describe("Execute", func() {
		var crdyaml []byte

		BeforeEach(func() {
			log.SetLevel(log.ErrorLevel)
			var err error
			crdyaml, err = testdata.ReadFile("testdata/objectbucket.io_objectbucketclaims.yaml")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should execute", Label("E2E"), func(ctx context.Context) {
			t := crd2pulumi.Tool{
				Path: toolPath,
				Options: crd2pulumi.Options{
					NodeJS: &crd2pulumi.LangOptions{
						Enabled: true,
					},
				},
			}
			fs := afero.NewMemMapFs()
			err := afero.WriteFile(fs, "blah.yaml", crdyaml, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			out, err := t.Execute(ctx, fs)

			Expect(err).NotTo(HaveOccurred())
			Expect(afero.IsEmpty(out, "")).To(BeFalseBecause("the tool generated files"))
			Expect(out).To(ContainFile("nodejs/package.json"))
			Expect(out).To(ContainFile("nodejs/index.ts"))
		})
	})
})
