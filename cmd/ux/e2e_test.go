package main_test

import (
	"context"
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("E2e", func() {
	It("should pass my excessive sanity check", func() {
		Expect(bin).NotTo(BeEmpty())
	})

	It("should execute", func(ctx context.Context) {
		err := exec.CommandContext(ctx, bin).Run()

		Expect(err).NotTo(HaveOccurred())
	})
})
