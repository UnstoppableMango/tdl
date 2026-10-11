package haskell

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

var (
	// conid is a name Haskell accepts for a type, a constructor, or a module
	// segment.
	conid = regexp.MustCompile(`^[A-Z][A-Za-z0-9_']*$`)
	// varid is a name Haskell accepts for a field.
	varid = regexp.MustCompile(`^([a-z][A-Za-z0-9_']*|_[A-Za-z0-9_']+)$`)
)

// keywords are the words a field or parameter cannot be spelled as, which
// get `_` appended.
var keywords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		case class data default deriving do else forall foreign if import in
		infix infixl infixr instance let module newtype of then type where`) {
		keywords[w] = true
	}
}

func upperFirst(s string) string { return mapFirst(s, unicode.ToUpper) }

func lowerFirst(s string) string { return mapFirst(s, unicode.ToLower) }

func mapFirst(s string, f func(rune) rune) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = f(r[0])
	return string(r)
}

// escape appends `_` to a keyword.
func escape(s string) string {
	if keywords[s] {
		return s + "_"
	}
	return s
}

// module is the module a model generates: the target block's `module`
// directive, or else the package's segments with each first letter
// upper-cased. It is an error when there is neither or Haskell refuses the
// name.
func (g *generator) module() (string, error) {
	var name string
	var pos *ir.Position
	if dir, ok := g.Block("module"); ok {
		name, pos = dir.GetArgs()[0].GetText(), dir.GetPosition()
	} else if pkg := g.Model.GetPackage(); pkg != "" {
		name = packageModule(pkg)
	}
	switch {
	case name == "":
		return "", emit.Invalid(pos, "a Haskell module needs a name: give the model a package, or the haskell target block a module directive")
	case !moduleName(name):
		return "", emit.Invalid(pos, "%s is not a Haskell module name", name)
	}
	return name, nil
}

// typeName is a declaration's Haskell type name.
func (g *generator) typeName(d *ir.Decl) (string, error) {
	n := g.DeclName(d, upperFirst)
	if !conid.MatchString(n) {
		return "", emit.Unsupported(d.GetMeta().GetPosition(),
			"%s would be named %s in Haskell, which cannot name a type", d.GetMeta().GetName(), n)
	}
	return n, nil
}

// fieldName is a field's Haskell label: its name with the first letter
// lower-cased, or a `name` directive's argument, with a keyword escaped.
func (g *generator) fieldName(owner string, f *ir.Field) (string, error) {
	n := escape(g.FieldName(f, lowerFirst))
	if !varid.MatchString(n) {
		return "", emit.Unsupported(f.GetMeta().GetPosition(),
			"%s.%s would be labelled %s in Haskell, which cannot name a field", owner, f.GetMeta().GetName(), n)
	}
	return n, nil
}

// variantName is a variant's Haskell constructor: its name with the first
// letter upper-cased, behind the enum's name under `prefix`, or a `name`
// directive's argument.
func (g *generator) variantName(d *ir.Decl, enum string, v *ir.Variant) (string, error) {
	prefix := ""
	if g.prefix || g.Tagged(d.GetDirectives(), "prefix") {
		prefix = enum
	}
	n := g.VariantName(v, func(s string) string { return prefix + upperFirst(s) })
	if !conid.MatchString(n) {
		return "", emit.Unsupported(v.GetMeta().GetPosition(),
			"%s.%s would be named %s in Haskell, which cannot name a constructor", d.GetMeta().GetName(), v.GetMeta().GetName(), n)
	}
	return n, nil
}

// str is a Haskell string literal.
func str(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f:
			b.WriteString(`\` + strconv.Itoa(int(r)))
			if i+1 < len(rs) && unicode.IsDigit(rs[i+1]) {
				// A digit after a numeric escape would extend it.
				b.WriteString(`\&`)
			}
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
