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
	It("should execute", func(ctx context.Context) {
		t := crd2pulumi.New()
		fs := afero.NewMemMapFs()
		err := afero.WriteFile(fs, "blah.yaml", []byte("blah"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		out, err := t.Execute(ctx, fs)

		Expect(err).NotTo(HaveOccurred())
		Expect(afero.IsEmpty(out, "")).To(BeTrueBecause("generates nothing"))
	})
})
