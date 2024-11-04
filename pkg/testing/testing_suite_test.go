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

var testCacheForT *ttest.CacheForT

func TestTesting(t *testing.T) {
	log.SetLevel(log.FatalLevel)

	testCacheForT = ttest.NewCacheForT(t)

	RegisterFailHandler(Fail)
	RunSpecs(t, "Testing Suite")
}
