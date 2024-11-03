package plugin

import (
	"context"

	tdl "github.com/unstoppablemango/tdl/pkg"
)

type LocalRepo interface {
	tdl.Plugin
	Build() error
}

type RemoteRepo interface {
	LocalRepo
	Clone(context.Context) error
}
