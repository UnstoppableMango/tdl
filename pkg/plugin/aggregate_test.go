package plugin_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin"
	"github.com/unstoppablemango/tdl/pkg/testing"
)

var _ = Describe("Aggregate", func() {
	It("should consist of the given plugins", func() {
		p := &testing.MockPlugin{}

		result := plugin.NewAggregate(p)

		Expect(result).To(ConsistOf(p))
	})
})
