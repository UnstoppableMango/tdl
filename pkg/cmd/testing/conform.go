package testing

import (
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/conform"
)

func NewConform() *cobra.Command {
	return &cobra.Command{
		Use:   "conform [TARGET]",
		Short: "Run conformance tests against TARGET",
		Long: `This command will attempt to identify the type of
endpoint referred to by TARGET and perform the conformance
tests against it. Currently only supports a path to a binary
that communicates via stdin/stdout`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fs := afero.NewOsFs()
			endpoint := args[0]

			if _, err := fs.Stat(endpoint); err != nil {
				util.Fail(fmt.Errorf("only CLI tests are supported: %w", err))
			}

			ginkgo.Describe("Conformance Tests", func() {
				// conform.DescribeCli(endpoint, conform.WithArgs(args[1:]...))
			})

			gomega.RegisterFailHandler(ginkgo.Fail)
			if !ginkgo.RunSpecs(&conform.T{}, "Conformance Tests") {
				log.Debug("RunSpecs returned false")
			}
		},
	}
}
