package pipeline

import "github.com/unstoppablemango/tdl/pkg/tdl"

type Lookup[T, V any, P tdl.Pipeline[T, V]] func(string) (P, error)
