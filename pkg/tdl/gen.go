package tdl

import (
	tdlv1alpha1 "github.com/unstoppablemango/tdl/pkg/unmango/dev/tdl/v1alpha1"
)

type Gen[T any] func(*tdlv1alpha1.Spec) T
