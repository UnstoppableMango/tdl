package internal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/unstoppablemango/tdl/pkg/cmd/internal"
)

var _ = Describe("Args", func() {
	DescribeTable("SplitAt",
		Entry(nil, []string{}, 0, []string{}, []string{}),
		Entry(nil, []string{"blah"}, 0, []string{}, []string{"blah"}),
		Entry(nil, []string{"blah"}, 1, []string{"blah"}, []string{}),
		Entry(nil, []string{"blah", "bleh"}, 0, []string{}, []string{"blah", "bleh"}),
		Entry(nil, []string{"blah", "bleh"}, 1, []string{"blah"}, []string{"bleh"}),
		Entry(nil, []string{"blah", "bleh"}, 2, []string{"blah", "bleh"}, []string{}),
		func(args []string, i int, l, r []string) {
			al, ar := internal.SplitAt(args, i)

			Expect(al).To(ConsistOf(l))
			Expect(ar).To(ConsistOf(r))
		},
	)
})
