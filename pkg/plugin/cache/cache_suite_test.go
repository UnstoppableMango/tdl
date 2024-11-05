package cache_test

import (
	"testing"

	"github.com/charmbracelet/log"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCache(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cache Suite")
}
