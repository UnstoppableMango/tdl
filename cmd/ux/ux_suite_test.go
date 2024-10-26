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
	g := NewWithT(t)

	revParse, err := exec.CommandContext(context.Background(),
		"git", "rev-parse", "--show-toplevel",
	).CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred())

	root := strings.TrimSpace(string(revParse))
	bin = filepath.Join(root, "bin", "ux")
	g.Expect(os.Stat(bin)).NotTo(BeNil())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Ux Suite")
}
