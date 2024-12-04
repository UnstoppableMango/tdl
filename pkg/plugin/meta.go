package plugin

import (
	"github.com/unmango/go/iter"
	tdl "github.com/unstoppablemango/tdl/pkg"
	"github.com/unstoppablemango/tdl/pkg/meta"
)

func HasKey(plugin tdl.Plugin, key string) bool {
	return meta.HasKey(plugin.Meta(), key)
}

func HasMetaValue(plugin tdl.Plugin, key, value string) bool {
	return meta.HasValue(plugin.Meta(), key, value)
}

func MetaValue(plugin tdl.Plugin, key string) (string, bool) {
	return plugin.Meta().Value(key)
}

func MetaValues(plugin tdl.Plugin) iter.Seq2[string, string] {
	return plugin.Meta().Values()
}

func SupportsLang(plugin tdl.Plugin, lang string) bool {
	return HasMetaValue(plugin, meta.WellKnown.Lang, lang)
}

func MatchesName(plugin tdl.Plugin, name string) bool {
	if plugin.String() == name {
		return true
	}

	return HasMetaValue(plugin, meta.WellKnown.Name, name)
}
