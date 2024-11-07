package main_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/internal/util"
	. "github.com/unstoppablemango/tdl/pkg/testing"
)

var (
	gitRoot     string
	bin         string
	tsSuiteRoot string
)

var typescriptSuite []*Test

func TestUx(t *testing.T) {
	g := NewWithT(t)

	var err error
	gitRoot, err = util.GitRoot(context.Background())
	g.Expect(err).NotTo(HaveOccurred())

	bin = filepath.Join(gitRoot, "bin", "ux")
	g.Expect(os.Stat(bin)).NotTo(BeNil())

	tsSuiteRoot = filepath.Join(gitRoot, "packages", "ts", "testdata")
	typescriptSuite, err = Discover(afero.NewOsFs(), tsSuiteRoot)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(typescriptSuite).NotTo(BeEmpty())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Ux Suite")
}

func UxCommand(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, bin, args...)
}
