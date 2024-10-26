package main_test

import (
	"context"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/conform"
)

var _ = Describe("End to end", func() {
	conform.CliTests("ux e2e", bin, []string{"gen", "ts"})

	It("should pass my excessive sanity check", func() {
		Expect(bin).NotTo(BeEmpty())
	})

	It("should execute", func(ctx context.Context) {
		err := exec.CommandContext(ctx, bin).Run()

		Expect(err).NotTo(HaveOccurred())
	})
})
