package crd2pulumi_test

import (
	"archive/tar"
	"compress/gzip"
	"embed"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/afero/tarfs"
	"github.com/unmango/go/fs/github/repository/release/asset"
	. "github.com/unmango/go/testing/matcher"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

//go:embed testdata
var testdata embed.FS

func TestCrd2Pulumi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crd2Pulumi Suite")
}

var toolPath string

var _ = BeforeSuite(func() {
	tmp := GinkgoT().TempDir()
	fs := afero.NewBasePathFs(afero.NewOsFs(), tmp)
	Expect(fetch(fs)).To(Succeed())
	Expect(fs).To(ContainFile("crd2pulumi"))
	toolPath = filepath.Join(tmp, "crd2pulumi")
	Expect(toolPath).To(BeARegularFile())
})

func fetch(fs afero.Fs) error {
	client := github.DefaultClient
	assetfs := asset.NewFs(client, "pulumi", "crd2pulumi", "v1.5.4")
	assetName := fmt.Sprintf("crd2pulumi-v1.5.4-%s-amd64.tar.gz", runtime.GOOS)
	asset, err := assetfs.Open(assetName)
	if err != nil {
		return err
	}

	gz, err := gzip.NewReader(asset)
	if err != nil {
		return err
	}

	tarfs := tarfs.New(tar.NewReader(gz))
	bin, err := tarfs.Open("crd2pulumi")
	if err != nil {
		return err
	}

	if err = afero.WriteReader(fs, "crd2pulumi", bin); err != nil {
		return err
	}

	return fs.Chmod("crd2pulumi", 0o755)
}
