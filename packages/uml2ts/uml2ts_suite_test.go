package uml2ts_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/unstoppablemango/tdl/pkg/conform"
	"github.com/unstoppablemango/tdl/pkg/gen/cli"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
)

var suite e2e.Suite

func TestUml2ts(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()
	fs := afero.NewOsFs()

	var err error
	suite, err = conform.ReadLocalGitTests(ctx, fs, "typescript", map[string][]e2e.Assertion{
		"interface":        {conform.AssertStdout},
		"nested_interface": {conform.AssertStdout},
	})
	g.Expect(err).NotTo(HaveOccurred())

	RegisterFailHandler(Fail)
	RunSpecs(t, "Uml2ts Suite")
}

var _ = Describe("uml2ts Conformance", FlakeAttempts(5), func() {
	conform.DescribeGenerator(suite, cli.New("uml2ts", cli.ExpectStdout))
})
