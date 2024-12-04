package meta

import (
	"github.com/unmango/go/iter"
	"github.com/unmango/go/maps"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

type Map map[string]string

// Value implements tdl.Meta.
func (m Map) Value(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}

// Values implements tdl.Meta.
func (m Map) Values() iter.Seq2[string, string] {
	return maps.All(m)
}

func Empty() tdl.Meta {
	return Map{}
}
