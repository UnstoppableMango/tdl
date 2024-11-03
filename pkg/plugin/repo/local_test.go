package repo_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/unstoppablemango/tdl/pkg/plugin/repo"
)

var _ = Describe("Local", func() {
	Describe("Available", func() {
		It("should be available when running this test", func() {
			repo := repo.NewLocal(gitRoot, nil)

			result := repo.Available()

			Expect(result).To(BeTrueBecause("the repo is available"))
		})
	})

	Describe("String", func() {
		It("should match the current repo", func() {
			repo := repo.NewLocal(gitRoot, nil)

			result := repo.String()

			Expect(result).To(Equal(gitRoot))
		})
	})
})
