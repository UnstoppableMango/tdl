package conform_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConform(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conform Suite")
}
