package crd2pulumi_test

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

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
		Expect(afero.IsEmpty(out, "")).To(BeTrueBecause("generates nothing"))
	})
})
