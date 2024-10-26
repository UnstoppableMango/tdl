package conform

import (
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/spf13/afero"
)

type fakeT struct{}

func (t *fakeT) Fail() {
	fmt.Println("fail")
}

func Execute(fsys afero.Fs, endpoint string) error {
	if _, err := fsys.Stat(endpoint); err != nil {
		return err
	}

	CliTests(endpoint)

	gomega.RegisterFailHandler(ginkgo.Fail)
	if !ginkgo.RunSpecs(&fakeT{}, "Conformance Tests") {
		fmt.Println("RunSpecs returned false")
	}

	return nil
}
