package plugin

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/unstoppablemango/tdl/pkg/plugin/cache"
	"github.com/unstoppablemango/tdl/pkg/plugin/github"
)

func NewCache() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Cache known plugins",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()
			c := cache.Observe(cache.XdgBinHome)

			sub := cache.Subscribe(c, func(s string, i int, err error) {
				fmt.Println(i)
			})
			defer sub()

			p := github.NewUml2Ts()
			if p.Cached(c) {
				fmt.Println("Cached: uml2ts")
			} else if err := p.Cache(ctx, c); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		},
	}

	return cmd
}
