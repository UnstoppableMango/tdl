package golang

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// The JSON wire convention writes a duration as an ISO 8601 string, which
// encoding/json does not do for a time.Duration. The helper implements
// encoding.TextMarshaler, which encoding/json uses for a value and a map key. Under json, the package
// declares a helper type in its place, written once and only when the model
// uses the primitive. It is named for the primitive, or Wire and that name
// when the model declares a type of that name.

// wireType is a helper type standing for a primitive under json.
type wireType struct {
	// prim is the primitive it stands for, and name its Go name before a
	// clash renames it.
	prim, name string
	// doc follows the type's name in its doc comment, which import reads
	// to tell the helper from a newtype.
	doc     string
	imports []string
	// body writes the type and its methods, given its name.
	body func(name string) string
}

var wireTypes = []wireType{
	{
		prim:    "duration",
		name:    "Duration",
		doc:     "is a time.Duration that the JSON wire convention writes as an ISO 8601 duration with no year or month part, such as PT1M30S.",
		imports: []string{"strconv", "strings", "time"},
		body:    durationBody,
	},
}

// wireFor is the helper type standing for a primitive, if one does.
func wireFor(prim string) (wireType, bool) {
	for _, w := range wireTypes {
		if w.prim == prim {
			return w, true
		}
	}
	return wireType{}, false
}

// wireTypeName is the Go name of a helper type in this package, or an
// error when the package declares both names it could take.
func (g *generator) wireTypeName(w wireType, pos *ir.Position) (string, error) {
	for _, name := range []string{w.name, "Wire" + w.name} {
		if !g.packageDeclares(name) {
			return name, nil
		}
	}
	return "", emit.Unsupported(pos, "%s is written by a type named %s or Wire%s under json, and the model declares both", w.prim, w.name, w.name)
}

// packageDeclares reports whether the package declares an identifier for
// the model: a declaration, a variant, a key struct, or an enum's decoder.
func (g *generator) packageDeclares(goName string) bool {
	if g.declares(goName) {
		return true
	}
	for _, d := range g.Own() {
		name := g.declName(d)
		if goName == name+"Key" || goName == "Unmarshal"+name {
			return true
		}
		for _, v := range d.GetEnumeration().GetVariants() {
			if g.variantName(name, v) == goName {
				return true
			}
		}
	}
	return false
}

// useWire records that the declaration being rendered names a helper type.
func (g *generator) useWire(w wireType) {
	g.wireUsed[w.prim] = true
}

// wirePieces renders the helper types the pieces name, in a stable order.
func (g *generator) wirePieces(pieces []*piece) []*wirePiece {
	var out []*wirePiece
	for _, w := range wireTypes {
		used := false
		for _, p := range pieces {
			used = used || p.wire[w.prim]
		}
		if !used {
			continue
		}
		// Every use already resolved the name.
		name, _ := g.wireTypeName(w, nil)
		imports := map[string]string{}
		for _, path := range w.imports {
			imports[path] = ""
		}
		body := fmt.Sprintf("// %s %s\n", name, w.doc) + w.body(name)
		out = append(out, &wirePiece{name: name, piece: &piece{body: body, imports: imports}})
	}
	return out
}

// wirePiece is a rendered helper type.
type wirePiece struct {
	name  string
	piece *piece
}

// wirePath is the file a helper type is written to without a file
// directive, past the paths taken.
func wirePath(name string, taken func(string) bool) string {
	path := fileName(name)
	for taken(path) {
		path = "tdl_" + path
	}
	return path
}

// durationBody writes a Duration, whose String writes the convention's form.
func durationBody(name string) string {
	return strings.NewReplacer("$T", name, "$r", strings.ToLower(name[:1])).Replace(`type $T time.Duration

// String writes $r as an ISO 8601 duration, counting a day as 24 hours.
func ($r $T) String() string {
	if $r == 0 {
		return "PT0S"
	}
	var b strings.Builder
	u := uint64($r)
	if $r < 0 {
		b.WriteByte('-')
		u = -u
	}
	b.WriteByte('P')
	if n := u / uint64(24*time.Hour); n > 0 {
		b.WriteString(strconv.FormatUint(n, 10) + "D")
		u %= uint64(24 * time.Hour)
	}
	if u == 0 {
		return b.String()
	}
	b.WriteByte('T')
	if n := u / uint64(time.Hour); n > 0 {
		b.WriteString(strconv.FormatUint(n, 10) + "H")
		u %= uint64(time.Hour)
	}
	if n := u / uint64(time.Minute); n > 0 {
		b.WriteString(strconv.FormatUint(n, 10) + "M")
		u %= uint64(time.Minute)
	}
	if u > 0 {
		b.WriteString(strconv.FormatUint(u/uint64(time.Second), 10))
		if frac := u % uint64(time.Second); frac > 0 {
			b.WriteString("." + strings.TrimRight(strconv.FormatUint(frac+uint64(time.Second), 10)[1:], "0"))
		}
		b.WriteByte('S')
	}
	return b.String()
}

// MarshalText writes $r as an ISO 8601 duration, which encoding/json
// writes as a string, as a map key too.
func ($r $T) MarshalText() ([]byte, error) {
	return []byte($r.String()), nil
}
`)
}
