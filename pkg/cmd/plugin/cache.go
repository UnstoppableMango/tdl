package plugin

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/internal/util"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

func NewCache() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Cache known plugins",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			cache := cache.XdgBinHome
			p := github.NewUml2Ts(
				github.WithProgress(func(f float64, err error) {
					fmt.Printf("%f\r", f*100)
				}),
			)

			if p.Cached(cache) {
				fmt.Println("Cached: uml2ts")
				return
			}

			fmt.Println("Caching uml2ts")
			if err := p.Cache(ctx, cache); err != nil {
				util.Fail(err)
			}

			fmt.Printf("\nDone\n")
		},
	}

	return cmd
}
