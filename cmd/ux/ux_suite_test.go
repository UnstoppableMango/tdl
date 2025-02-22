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
	"github.com/unstoppablemango/tdl/pkg/conform"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

var (
	gitRoot string
	bin     string
)

var typescriptSuite e2e.Suite

func TestUx(t *testing.T) {
	g := NewWithT(t)

	var err error
	ctx := context.Background()
	gitRoot, err = util.GitRoot(ctx)
	g.Expect(err).NotTo(HaveOccurred())

	bin = filepath.Join(gitRoot, "bin", "ux")
	g.Expect(os.Stat(bin)).NotTo(BeNil())

	fs := afero.NewOsFs()
	typescriptSuite, err = conform.ReadLocalGitTests(ctx, fs, "typescript", map[string][]e2e.Assertion{
		"interface":        {conform.AssertStdout},
		"nested_interface": {conform.AssertStdout},
	})
	g.Expect(err).NotTo(HaveOccurred())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Ux Suite")
}

func UxCommand(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, bin, args...)
}

func tsSuitePath() string {
	return filepath.Join(gitRoot, "conformance", "typescript")
}
