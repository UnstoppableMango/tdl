package e2e_test

import (
	"context"
	"embed"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
)

var (
	//go:embed testdata
	testdata embed.FS
	testfs   afero.Fs
)

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2e Suite")
}

var _ = BeforeSuite(func(ctx context.Context) {
	testfs = afero.FromIOFS{
		FS: testdata,
	}
})
