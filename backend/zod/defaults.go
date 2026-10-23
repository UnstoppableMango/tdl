package zod

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A field's default is `.default(value)`. Zod does not parse a default, so
// each is checked here against the schema it defaults, by the rules Zod's
// own checks apply, and one that fails warns and is dropped.

// The patterns Zod checks a primitive's format with, in RE2. Zod's
// duration pattern looks ahead, which [duration] checks instead.
var (
	decimalPattern = regexp.MustCompile(`^-?[0-9]+(\.[0-9]+)?$`)
	datePattern    = `(?:(?:\d\d[2468][048]|\d\d[13579][26]|\d\d0[48]|[02468][048]00|[13579][26]00)-02-29|\d{4}-(?:(?:0[13578]|1[02])-(?:0[1-9]|[12]\d|3[01])|(?:0[469]|11)-(?:0[1-9]|[12]\d|30)|(?:02)-(?:0[1-9]|1\d|2[0-8])))`
	dateRegexp     = regexp.MustCompile(`^` + datePattern + `$`)
	// instant is Zod's datetime with an offset, and the seconds the
	// module's instant helper requires.
	instantRegexp  = regexp.MustCompile(`^` + datePattern + `T(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d(?:\.\d+)?(?:Z|[+-](?:[01]\d|2[0-3]):[0-5]\d)$`)
	durationRegexp = regexp.MustCompile(`^P(?:\d+W|(?:\d+Y)?(?:\d+M)?(?:\d+D)?(?:T(?:\d+H)?(?:\d+M)?(?:\d+(?:[.,]\d+)?S)?)?)$`)
	uuidRegexp     = regexp.MustCompile(`^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$`)
	base64Regexp   = regexp.MustCompile(`^$|^(?:[0-9a-zA-Z+/]{4})*(?:(?:[0-9a-zA-Z+/]{2}==)|(?:[0-9a-zA-Z+/]{3}=))?$`)
)

// duration reports whether s is an ISO 8601 duration as Zod reads one:
// at least one component, and a digit after the T.
func duration(s string) bool {
	if !durationRegexp.MatchString(s) || s == "P" {
		return false
	}
	t := strings.IndexByte(s, 'T')
	return t < 0 || (t+1 < len(s) && s[t+1] >= '0' && s[t+1] <= '9')
}

// maxSafe is the largest integer Zod's z.int() accepts.
const maxSafe = 1<<53 - 1

// intRanges bounds each integer primitive.
var intRanges = map[string][2]int64{
	"int":    {-maxSafe, maxSafe},
	"int64":  {-maxSafe, maxSafe},
	"int32":  {math.MinInt32, math.MaxInt32},
	"uint32": {0, math.MaxUint32},
	"uint64": {0, maxSafe},
}

// withDefault adds a field's default to its schema, or warns and leaves
// the schema as it is when the default cannot be written.
func (g *generator) withDefault(owner string, f *ir.Field, r *emit.Ref, s schema) schema {
	def := f.GetDefaultValue()
	at := def.GetPosition()
	if at == nil {
		at = f.GetMeta().GetPosition()
	}
	// warn drops the default. The mixin a field was copied from warns
	// for it.
	warn := func(format string, args ...any) schema {
		if f.GetIncludedFrom() == nil {
			g.Lossy(emit.LossDefault, at, "%s.%s's default %s "+format,
				append([]any{owner, f.GetMeta().GetName(), emit.LiteralText(def)}, args...)...)
		}
		return s
	}

	for r.Form == emit.Option || r.Form == emit.Nullable {
		if r.Form == emit.Option {
			return warn("is not written: the field may be left out, and a default would make the parsed value always hold it")
		}
		r = r.Elem
	}
	lit, why := g.literal(r, def)
	if why != "" {
		return warn("is not written: %s", why)
	}
	return schema{expr: s.expr + ".default(" + lit + ")", typ: "z.ZodDefault<" + s.typ + ">", nullable: s.nullable}
}

// literal is the TypeScript value of a literal for a schema of r, or why
// the schema would refuse it.
func (g *generator) literal(r *emit.Ref, l *ir.Literal) (string, string) {
	kind, text := l.GetKind(), l.GetText()
	switch r.Form {
	case emit.Option, emit.Nullable:
		// An element, which no literal leaves absent.
		return g.literal(r.Elem, l)
	case emit.Extern:
		return "", "is on a type another package declares, which this backend cannot check"
	case emit.List, emit.Set:
		if kind != ir.LiteralKind_LITERAL_KIND_LIST {
			return "", "is not a list"
		}
		items := make([]string, len(l.GetItems()))
		seen := map[string]bool{}
		for i, item := range l.GetItems() {
			v, why := g.literal(r.Elem, item)
			if why != "" {
				return "", why
			}
			if r.Form == emit.Set && seen[v] {
				return "", "holds " + v + " twice, and a set holds each element once"
			}
			seen[v] = true
			items[i] = v
		}
		return "[" + strings.Join(items, ", ") + "]", ""
	case emit.Map:
		return "", "is not a map"
	case emit.Named:
		d := r.Decl
		switch {
		case func() bool { _, ok := g.Find(d.GetDirectives(), "foreign"); return ok }():
			return "", "is on a foreign type, which this backend cannot check"
		case d.GetNewtype() != nil:
			base, err := g.Resolve(d.GetNewtype().GetBase())
			if err != nil {
				return "", err.Error()
			}
			return g.literal(base, l)
		case d.GetEnumeration() != nil && !emit.Fielded(d.GetEnumeration()):
			if kind != ir.LiteralKind_LITERAL_KIND_NAME {
				return "", "is not a variant of " + d.GetMeta().GetName()
			}
			return strconv.Quote(text), ""
		}
		return "", "is on " + d.GetMeta().GetName() + ", which no literal can hold"
	}

	str := kind == ir.LiteralKind_LITERAL_KIND_STRING
	number := kind == ir.LiteralKind_LITERAL_KIND_INT || kind == ir.LiteralKind_LITERAL_KIND_FLOAT
	switch name := r.Name; name {
	case "string":
		if str {
			return strconv.Quote(text), ""
		}
		return "", "is not a string"
	case "bool":
		if kind == ir.LiteralKind_LITERAL_KIND_BOOL {
			return text, ""
		}
		return "", "is not a bool"
	case "int", "int64", "int32", "uint32", "uint64":
		n, err := strconv.ParseInt(text, 10, 64)
		if kind != ir.LiteralKind_LITERAL_KIND_INT || err != nil {
			return "", "is not an integer " + name + " holds"
		}
		if bounds := intRanges[name]; n < bounds[0] || n > bounds[1] {
			return "", "is out of " + name + "'s range"
		}
		return strconv.FormatInt(n, 10), ""
	case "float32", "float64":
		f, err := strconv.ParseFloat(text, 64)
		if !number || err != nil || math.IsInf(f, 0) {
			return "", "is not a finite number"
		}
		if name == "float32" && math.Abs(f) > math.MaxFloat32 {
			return "", "is out of float32's range"
		}
		return strconv.FormatFloat(f, 'g', -1, 64), ""
	case "decimal":
		if (str || number) && decimalPattern.MatchString(text) {
			return strconv.Quote(text), ""
		}
		return "", "is not a decimal string"
	}

	formats := map[string]func(string) bool{
		"uuid":     uuidRegexp.MatchString,
		"instant":  instantRegexp.MatchString,
		"date":     dateRegexp.MatchString,
		"duration": duration,
		"bytes":    base64Regexp.MatchString,
	}
	check, ok := formats[r.Name]
	if !ok {
		return "", "is on " + r.Name + ", which has no Zod schema"
	}
	if !str || !check(text) {
		return "", "is not a " + r.Name + " string"
	}
	return strconv.Quote(text), ""
}

// foreignLosses warns about what a foreign declaration's schema is not
// known to check.
func (g *generator) foreignLosses(d *ir.Decl) {
	name := emit.LastSegment(d.GetMeta().GetName())
	if n := len(d.GetNewtype().GetValueConstraints()); n > 0 {
		g.Lossy(emit.LossConstraint, d.GetMeta().GetPosition(),
			"%s is foreign, and its %d where constraint(s) are the foreign schema's to check", name, n)
	}
}
