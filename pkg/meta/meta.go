package meta

import tdl "github.com/unstoppablemango/tdl/pkg"

func HasKey(m tdl.Meta, key string) bool {
	_, ok := m.Value(key)
	return ok
}

func HasValue(m tdl.Meta, key, value string) bool {
	v, ok := m.Value(key)
	return ok && v == value
}
