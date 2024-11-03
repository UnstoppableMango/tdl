package token_test

import (
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/token"
)

var _ = Describe("Token", func() {
	Describe("Parse", func() {
		It("should use input as token name", func() {
			fn := func(input string) bool {
				token, err := token.Parse(input)

				return err == nil && token.Name == input
			}

			Expect(quick.Check(fn, nil)).To(Succeed())
		})
	})
})
