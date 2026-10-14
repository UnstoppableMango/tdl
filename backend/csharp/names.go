package csharp

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/unstoppablemango/tdl/backend/internal/dotnet"
	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// keywords are C#'s reserved words, which an identifier spells with `@`.
var keywords = map[string]bool{}

// reservedMembers are the members every record declares, which no property
// or variant may be named.
var reservedMembers = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`
		abstract as base bool break byte case catch char checked class const
		continue decimal default delegate do double else enum event explicit
		extern false finally fixed float for foreach goto if implicit in int
		interface internal is lock long namespace new null object operator out
		override params private protected public readonly ref return sbyte
		sealed short sizeof stackalloc static string struct switch this throw
		true try typeof uint ulong unchecked unsafe ushort using virtual void
		volatile while`) {
		keywords[w] = true
	}
	for _, m := range strings.Fields(`
		Equals GetHashCode ToString GetType Deconstruct EqualityContract
		PrintMembers MemberwiseClone Finalize`) {
		reservedMembers[m] = true
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

// identifier reports whether C# accepts s as an identifier, before any `@`.
func identifier(s string) bool {
	for i, r := range s {
		switch {
		case r == '_' || unicode.IsLetter(r):
		case i > 0 && unicode.IsDigit(r):
		default:
			return false
		}
	}
	return s != ""
}

// escape writes a keyword with `@`.
func escape(s string) string {
	if keywords[s] {
		return "@" + s
	}
	return s
}

// namespace is the target block's `namespace` directive, or else the
// package's segments with each first letter upper-cased. It is an error
// when there is neither or C# refuses the name.
func (g *generator) namespace() (string, error) {
	var name string
	var pos *ir.Position
	if dir, ok := g.Block("namespace"); ok {
		name, pos = dir.GetArgs()[0].GetText(), dir.GetPosition()
	} else if pkg := g.Model.GetPackage(); pkg != "" {
		name = dotnet.Namespace(pkg)
	}
	if name == "" {
		return "", emit.Invalid(pos, "a C# namespace needs a name: give the model a package, or the csharp target block a namespace directive")
	}
	for seg := range strings.SplitSeq(name, ".") {
		if !identifier(seg) || keywords[seg] || seg == "global" {
			return "", emit.Invalid(pos, "%s is not a C# namespace", name)
		}
	}
	return name, nil
}

// typeName is a declaration's C# type name.
func (g *generator) typeName(d *ir.Decl) (string, error) {
	n, err := g.memberName(g.DeclName(d, upperFirst))
	if err != nil {
		return "", emit.Unsupported(d.GetMeta().GetPosition(), "%s %v", d.GetMeta().GetName(), err)
	}
	return n, nil
}

// memberName checks a name C# would declare, and escapes a keyword.
func (g *generator) memberName(n string) (string, error) {
	if !identifier(n) {
		return "", fmt.Errorf("would be named %s in C#, which is not an identifier", n)
	}
	return escape(n), nil
}

// variantName is a variant's C# name, refusing one its enum's scope cannot
// hold. seen holds the names taken before it.
func (g *generator) variantName(d *ir.Decl, typ string, v *ir.Variant, seen map[string]bool) (string, error) {
	pos := v.GetMeta().GetPosition()
	owner := d.GetMeta().GetName() + "." + v.GetMeta().GetName()
	n, err := g.memberName(g.VariantName(v, upperFirst))
	if err != nil {
		return "", emit.Unsupported(pos, "%s %v", owner, err)
	}
	bare := strings.TrimPrefix(n, "@")
	switch {
	case bare == strings.TrimPrefix(typ, "@"):
		return "", emit.Unsupported(pos, "%s would be named %s in C#, which a member of %s cannot be named; a name directive resolves it", owner, n, typ)
	case reservedMembers[bare]:
		return "", emit.Unsupported(pos, "%s would be named %s in C#, which every type declares; a name directive resolves it", owner, n)
	case seen[bare]:
		return "", emit.Unsupported(pos, "%s would be named %s in C#, and another variant already is; a name directive resolves it", owner, n)
	}
	seen[bare] = true
	return n, nil
}

// str is a C# string literal.
func str(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20 || r == 0x7f || r == 0x2028 || r == 0x2029:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
