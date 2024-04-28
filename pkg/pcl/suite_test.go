package pcl

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPcl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Pcl Suite")
}
