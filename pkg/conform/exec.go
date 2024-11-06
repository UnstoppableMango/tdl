package conform

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/spf13/afero"
)

type fakeT struct{}

func (t *fakeT) Fail() {
	fmt.Fprintln(os.Stderr, "fail")
	os.Exit(1)
}

func Execute(fsys afero.Fs, endpoint string, args []string) error {
	if _, err := fsys.Stat(endpoint); err != nil {
		return fmt.Errorf("only CLI tests are supported: %w", err)
	}

	ginkgo.Describe("Conformance Tests", func() {
		CliTests(endpoint, WithArgs(args...))
	})

	gomega.RegisterFailHandler(ginkgo.Fail)
	if !ginkgo.RunSpecs(&fakeT{}, "Conformance Tests") {
		log.Debug("RunSpecs returned false")
	}

	return nil
}
