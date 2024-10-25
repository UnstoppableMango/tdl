package main_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var bin string

func TestUx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ux Suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	revParse, err := exec.CommandContext(ctx,
		"git", "rev-parse", "--show-toplevel",
	).CombinedOutput()
	Expect(err).NotTo(HaveOccurred())

	root := strings.TrimSpace(string(revParse))
	bin = filepath.Join(root, "bin", "ux")
	Expect(os.Stat(bin)).NotTo(BeNil())
})
