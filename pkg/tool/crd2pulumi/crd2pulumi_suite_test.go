package crd2pulumi_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCrd2pulumi(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crd2Pulumi Suite")
}
