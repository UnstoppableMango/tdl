package conform

import (
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spf13/afero"
	aferox "github.com/unmango/go/fs"
	"github.com/unstoppablemango/tdl/pkg/testing/e2e"
	. "github.com/unstoppablemango/tdl/pkg/testing/matcher"
)

func AssertStdout(test *e2e.Test, output afero.Fs) {
	By("opening test output")
	expected, err := aferox.OpenSingle(test.Expected, "")
	Expect(err).NotTo(HaveOccurred())

	By("reading test output")
	data, err := io.ReadAll(expected)
	Expect(err).NotTo(HaveOccurred())

	Expect(output).To(ContainFileWithBytes("stdout", data))
}

func AssertFile(path string) e2e.Assertion {
	return func(test *e2e.Test, output afero.Fs) {
		By("opening test output")
		expected, err := test.Expected.Open(path)
		Expect(err).NotTo(HaveOccurred())

		By("reading test output")
		data, err := io.ReadAll(expected)
		Expect(err).NotTo(HaveOccurred())

		Expect(output).To(ContainFileWithBytes(path, data))
	}
}
