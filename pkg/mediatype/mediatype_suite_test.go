package mediatype_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMediaType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MediaType Suite")
}
