package token_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/token"
)

var _ = Describe("Token", func() {
	DescribeTable("String",
		func(tok token.Token, expected string) {
			Expect(tok.String()).To(Equal(expected))
		},
		Entry(nil, token.COLON, ":"),
		Entry(nil, token.TYPE, "type"),
		Entry(nil, token.Token(12345), "token(12345)"),
	)

	DescribeTable("Lookup",
		func(input string, tok token.Token) {
			Expect(token.Lookup(input)).To(Equal(tok))
		},
		Entry(nil, "foo", token.IDENT),
		Entry(nil, "bar", token.IDENT),
		Entry(nil, "type", token.TYPE),
	)
})
