// Package skills writes the skill library's generated references and holds
// the tests every skill under skills/ shares.
//
// A skill's SKILL.md is written by hand; its references are copied from the
// docs by `make skills`, which runs tools/skills.
package skills

import (
	"fmt"
	"maps"
	"path"
	"slices"

	"github.com/unstoppablemango/tdl/docs"
	"github.com/unstoppablemango/tdl/internal/cli"
	"github.com/unstoppablemango/tdl/prelude"
)

// Dir is the skill library, relative to the module root.
const Dir = "skills"

// VersionFile is the reference every skill carries, naming the release its
// other references were copied from.
const VersionFile = "references/version.md"

// sources are each skill's copied references, by path within the skill.
var sources = map[string]map[string]string{
	"tdl-modeling": {
		"references/spec.md":      docs.Spec,
		"references/grammar.ebnf": docs.Grammar,
		"references/std.tdl":      prelude.Source,
	},
}

// Names lists every skill that has generated references, sorted.
func Names() []string {
	return slices.Sorted(maps.Keys(sources))
}

// References returns every generated reference, keyed by its path relative
// to the module root.
func References() map[string][]byte {
	version := versionFile()
	out := map[string][]byte{}
	for name, refs := range sources {
		for p, src := range refs {
			out[path.Join(Dir, name, p)] = []byte(src)
		}
		out[path.Join(Dir, name, VersionFile)] = version
	}
	return out
}

// versionFile records toolVersion and specVersion. release-please rewrites
// the annotated line on a release, so it writes what this does.
func versionFile() []byte {
	tool, spec := cli.Versions()
	return fmt.Appendf(nil, `# Versions

The release this skill's references were copied from.
`+"`tdl version`"+` prints the same two versions.

- tdl: %s <!-- x-release-please-version -->
- spec: %s
`, tool, spec)
}
