package pipeline

import "github.com/unstoppablemango/tdl/pkg/tdl/constraint"

type Lookup[T, V any, P constraint.Pipeline[T, V]] func(string) (P, error)
