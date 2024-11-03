package repo_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/internal/util"
)

var gitRoot string

func TestRepo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Repo Suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	var err error
	gitRoot, err = util.GitRoot(ctx)
	Expect(err).NotTo(HaveOccurred())
})
