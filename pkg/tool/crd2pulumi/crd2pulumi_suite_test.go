package crd2pulumi_test

import (
	"embed"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

//go:embed testdata
var testdata embed.FS

func TestCrd2pulumi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crd2Pulumi Suite")
}
