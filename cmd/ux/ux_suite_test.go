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
	"github.com/spf13/afero"
	. "github.com/unstoppablemango/tdl/pkg/testing"
)

var (
	gitRoot string
	bin     string
)

var typescriptSuite []*Test

func TestUx(t *testing.T) {
	g := NewWithT(t)

	revParse, err := exec.CommandContext(context.Background(),
		"git", "rev-parse", "--show-toplevel",
	).CombinedOutput()
	g.Expect(err).NotTo(HaveOccurred())

	gitRoot = strings.TrimSpace(string(revParse))
	bin = filepath.Join(gitRoot, "bin", "ux")
	g.Expect(os.Stat(bin)).NotTo(BeNil())

	typescriptSuite, err = Discover(
		afero.NewOsFs(),
		filepath.Join(gitRoot, "packages", "ts", "testdata"),
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(typescriptSuite).NotTo(BeEmpty())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Ux Suite")
}
