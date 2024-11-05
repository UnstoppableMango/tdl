package testing_test

import (
	"embed"
	"testing"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ttest "github.com/unstoppablemango/tdl/pkg/testing"
)

//go:embed testdata
var testdata embed.FS

func TestCacheForT(t *testing.T) {
	g := NewWithT(t)
	cache := ttest.NewCacheForT(t)
	data := []byte("dkfjslkdfjksdlf")

	err := cache.Write("test-bin", data)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestTesting(t *testing.T) {
	log.SetLevel(log.FatalLevel)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Testing Suite")
}
