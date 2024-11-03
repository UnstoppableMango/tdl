package target

import (
	"context"
	"iter"

	"github.com/unstoppablemango/tdl/internal/util"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/plugin/repo"
)

type TypeScript struct{}

// Plugins implements tdl.Target.
func (t *TypeScript) Plugins() (iter.Seq[tdl.Plugin], error) {
	return func(yield func(tdl.Plugin) bool) {
		ctx := context.Background()
		if path, err := util.GitRoot(ctx); err != nil {
			if !yield(repo.NewLocal(path, t)) {
				return
			}
		}
	}, nil
}

// String implements tdl.Target.
func (t *TypeScript) String() string {
	return "TypeScript"
}
