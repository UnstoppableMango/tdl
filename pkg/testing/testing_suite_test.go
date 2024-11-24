package testing_test

import (
	"bytes"
	"embed"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ttest "github.com/unstoppablemango/tdl/pkg/testing"
)

//go:embed testdata
var testdata embed.FS

func TestCacheForT(t *testing.T) {
	g := NewWithT(t)
	cache := ttest.NewCacheForT(t)
	data := bytes.NewBufferString("dkfjslkdfjksdlf")

	err := cache.WriteAll("test-bin", data)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestTesting(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testing Suite")
}
