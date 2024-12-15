package meta

import (
	"github.com/charmbracelet/log"
	tdl "github.com/unstoppablemango/tdl/pkg"
)

func HasKey(m tdl.Meta, key string) bool {
	_, ok := m.Value(key)
	return ok
}

func HasValue(m tdl.Meta, key, value string) bool {
	v, ok := m.Value(key)
	return ok && v == value
}

func Supports(a tdl.Meta, b tdl.Meta) bool {
	if a == nil || b == nil {
		return false
	}

	log.Debugf("comparing: %v to %v", a, b)
	total, count := 0.0, 0.0
	for k, v := range a.Values() {
		total++
		if HasValue(b, k, v) {
			count++
		}
	}

	return total > 0 && count/total > 0.5
}
