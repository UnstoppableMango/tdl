package crd2pulumi_test

import (
	"embed"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:embed testdata
var testdata embed.FS

func TestCrd2Pulumi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crd2Pulumi Suite")
}

var toolPath string

var _ = BeforeSuite(func() {
	var err error
	toolPath, err = exec.LookPath("crd2pulumi")
	Expect(err).NotTo(HaveOccurred())
	Expect(toolPath).To(BeARegularFile())
})
