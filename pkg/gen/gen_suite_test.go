package gen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gen Suite")
}
