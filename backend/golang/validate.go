package golang

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/ir"
)

// A type with something to check gets two methods. Validate joins every
// violation into one error, and validate threads the path a container
// prefixes and appends to what was found so far: a container prefixing a
// child's joined error would prefix only its first line.
//
// A message is `<path>: <constraint as written>: <detail>`, the path rooted
// at the TDL name. It echoes numbers, counts, indices, and enum values, and
// never a string's or bytes' contents, since validation errors reach logs
// and those contents are often addresses or secrets.

// validKey names what carries the methods: a declaration, or one variant of
// a sealed enum.
type validKey struct {
	decl    int32
	variant int
}

// checkUnit is one Go type that can carry the methods: a struct, a newtype,
// or a variant of a sealed enum.
type checkUnit struct {
	variant int // -1 for the declaration itself
	goName  string
	root    string // the TDL name a path starts from
	fields  []*ir.Field
	cs      []*ir.Constraint
	base    *ir.ID // a newtype's base
}

// planValidation decides which types validate before anything is rendered,
// since a container calls into what it holds. A type validates when its
// checks render to anything, which can depend on a type later in the table,
// so this repeats until a pass adds nothing.
func (g *generator) planValidation() {
	g.valid = map[validKey]bool{}
	saved := g.Diags
	defer func() { g.Diags = saved }()

	for again := true; again; {
		again = false
		for i, d := range g.Model.GetDecls() {
			if !emit.IsOwn(d) {
				continue
			}
			g.cur, g.imports = int32(i), map[string]bool{}
			g.curClasses = g.paramClasses(d, false)
			for _, u := range g.checkUnits(d) {
				key := validKey{int32(i), u.variant}
				if !g.valid[key] && g.checks(d, u, false) != nil {
					g.valid[key] = true
					again = true
				}
			}
		}
	}
}

func (g *generator) checkUnits(d *ir.Decl) []checkUnit {
	goName, root := g.declName(d), tdlName(d.GetMeta().GetName())
	if goName == "" {
		return nil
	}
	switch {
	case d.GetStructure() != nil:
		return []checkUnit{{variant: -1, goName: goName, root: root, fields: d.Fields()}}
	case d.GetNewtype() != nil:
		n := d.GetNewtype()
		return []checkUnit{{variant: -1, goName: goName, root: root, cs: n.GetValueConstraints(), base: n.GetBase()}}
	case d.GetEnumeration() != nil && emit.Fielded(d.GetEnumeration()):
		var out []checkUnit
		for i, v := range d.GetEnumeration().GetVariants() {
			out = append(out, checkUnit{
				variant: i,
				goName:  goName + exported(v.GetMeta().GetName()),
				root:    root + "." + v.GetMeta().GetName(),
				fields:  v.GetFields(),
			})
		}
		return out
	}
	return nil
}

// writeValidation writes the methods of the declaration being rendered, or
// of one of its variants, when it has something to check.
func (g *generator) writeValidation(b *strings.Builder, d *ir.Decl, variant int) {
	for _, u := range g.checkUnits(d) {
		if u.variant != variant {
			continue
		}
		w := g.checks(d, u, true)
		if w == nil {
			return
		}
		g.use("errors")
		for p := range w.imports {
			g.use(p)
		}

		self := u.goName + typeArgs(d)
		for _, p := range w.patterns {
			fmt.Fprintf(b, "\n%s\n", p)
		}
		fmt.Fprintf(b, "\n// Validate reports every constraint %s breaks, joined, or nil.\n", u.root)
		fmt.Fprintf(b, "func (%s %s) Validate() error {\n\treturn errors.Join(%s.validate(%q, nil)...)\n}\n",
			w.recv, self, w.recv, u.root)
		fmt.Fprintf(b, "\nfunc (%s %s) validate(%s string, %s []error) []error {\n%s\treturn %s\n}\n",
			w.recv, self, w.path, w.errs, w.b.String(), w.errs)
	}
}

// checks renders a unit's validate body. With report set, what it cannot
// check is a warning. It returns nil when there is nothing to check or when
// a field leaves the methods no room.
func (g *generator) checks(d *ir.Decl, u checkUnit, report bool) *checkWriter {
	w := newCheckWriter(d, u.goName)
	root := checkPath{format: "%s", args: []string{w.path}}

	if u.base != nil {
		if g.pointerOrInterface(u.base, map[int32]bool{}) {
			if report && len(u.cs) > 0 {
				g.Warn(emit.Unsupported(d.GetMeta().GetPosition(),
					"%s is a newtype over a pointer or an interface, which Go gives no methods, so its constraints are not checked",
					d.GetMeta().GetName()))
			}
			return nil
		}
		// An inherited constraint warns where it was written.
		g.constrain(w, w.recv, true, u.base, nil, u.cs, root, func(c *ir.Constraint) bool {
			return report && (c.GetFrom() == nil || !emit.IsOwn(g.Model.Decl(c.GetFrom())))
		})
		// The compiler hands a newtype its parents' constraints, so only
		// what is under the chain has checks of its own.
		g.visit(w, w.recv, true, g.pastNewtypes(u.base), nil, root)
	}
	for _, f := range u.fields {
		expr := w.recv + "." + g.fieldName(f)
		fp := root.child("." + escapeVerbs(f.GetMeta().GetName()))
		// A field copied from a mixin this model owns warns on the mixin.
		mixin := g.Model.Decl(f.GetIncludedFrom())
		g.constrain(w, expr, false, f.GetType(), nil, f.GetConstraints(), fp, func(*ir.Constraint) bool {
			return report && (mixin == nil || !emit.IsOwn(mixin))
		})
		g.visit(w, expr, false, f.GetType(), nil, fp)
	}

	if w.b.Len() == 0 {
		return nil
	}
	for _, f := range u.fields {
		if n := g.fieldName(f); n == "Validate" || n == "validate" {
			if report {
				g.Warn(emit.Unsupported(d.GetMeta().GetPosition(),
					"%s has a field named %s, which the Validate method would collide with, so its constraints are not checked",
					d.GetMeta().GetName(), n))
			}
			return nil
		}
	}
	return w
}

// checkWriter is a validate body being written, and the names it has taken.
type checkWriter struct {
	b        *strings.Builder
	goName   string
	recv     string
	path     string
	errs     string
	taken    map[string]bool
	imports  map[string]bool
	patterns []string
}

// newCheckWriter names the receiver and parameters around a type's type
// parameters, since the method shares a scope with them.
func newCheckWriter(d *ir.Decl, goName string) *checkWriter {
	w := &checkWriter{b: &strings.Builder{}, goName: goName, taken: map[string]bool{}, imports: map[string]bool{}}
	for _, n := range paramNames(d) {
		w.taken[n] = true
	}
	w.recv = receiver(d, goName)
	w.taken[w.recv] = true
	w.path, w.errs = w.local("path"), w.local("errs")
	return w
}

// local reserves a name spelled like base, with a number after it when that
// is taken.
func (w *checkWriter) local(base string) string {
	for i := 0; ; i++ {
		name := base
		if i > 0 {
			name += strconv.Itoa(i)
		}
		if !w.taken[name] {
			w.taken[name] = true
			return name
		}
	}
}

// release frees names whose scope has closed.
func (w *checkWriter) release(names ...string) {
	for _, n := range names {
		delete(w.taken, n)
	}
}

func (w *checkWriter) line(format string, args ...any) {
	fmt.Fprintf(w.b, format+"\n", args...)
}

// errorf appends a violation at a path.
func (w *checkWriter) errorf(p checkPath, text, detail string, args ...string) {
	w.imports["fmt"] = true
	all := append(slices.Clone(p.args), args...)
	w.line("%s = append(%s, fmt.Errorf(%s, %s))",
		w.errs, w.errs, strconv.Quote(p.format+": "+escapeVerbs(text)+": "+detail), strings.Join(all, ", "))
}

// pattern declares a compiled pattern and returns the variable holding it.
func (w *checkWriter) pattern(pat string) string {
	name := fmt.Sprintf("pattern%s_%d", w.goName, len(w.patterns))
	lit := "`" + pat + "`"
	if strings.Contains(pat, "`") {
		lit = strconv.Quote(pat)
	}
	w.patterns = append(w.patterns, fmt.Sprintf("var %s = regexp.MustCompile(%s)", name, lit))
	w.imports["regexp"] = true
	return name
}

// checkPath is a value's path as a format and the expressions filling it.
type checkPath struct {
	format string
	args   []string
}

func (p checkPath) child(suffix string, args ...string) checkPath {
	return checkPath{p.format + suffix, append(slices.Clone(p.args), args...)}
}

// expr is a Go expression for the path, to hand to a child's validate.
func (p checkPath) expr(w *checkWriter) string {
	rest := strings.TrimPrefix(p.format, "%s")
	switch {
	case len(p.args) == 1 && rest == "":
		return p.args[0]
	case len(p.args) == 1:
		return p.args[0] + "+" + strconv.Quote(strings.ReplaceAll(rest, "%%", "%"))
	}
	w.imports["fmt"] = true
	return "fmt.Sprintf(" + strconv.Quote(p.format) + ", " + strings.Join(p.args, ", ") + ")"
}

func escapeVerbs(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}

// shape is what a constraint checks: the Go value under a type, through
// aliases and newtypes.
type shape int

const (
	shapeOther shape = iota
	shapeInt
	shapeString
	shapeBool
	shapeBytes
	shapeList
	shapeSet
	shapeMap
	shapeEnum // a fieldless enum
	shapeOption
	shapeDecimal
	shapeTime
	shapeParam
)

var primitiveShapes = map[string]shape{
	"int":      shapeInt,
	"string":   shapeString,
	"uuid":     shapeString,
	"bool":     shapeBool,
	"bytes":    shapeBytes,
	"decimal":  shapeDecimal,
	"instant":  shapeTime,
	"date":     shapeTime,
	"duration": shapeTime,
	"List":     shapeList,
	"Set":      shapeSet,
	"Map":      shapeMap,
}

// checked is the value a constraint checks.
type checked struct {
	shape shape
	t     *ir.Type // the type with that shape, whose arguments a collection reads
	fr    *frame
	enum  *ir.Decl
	// named says the value's Go type is a newtype over the shape, so a
	// string operation converts it and an enum comparison names the enum.
	named bool
}

func (c checked) describe() string {
	switch c.shape {
	case shapeInt:
		return "an integer"
	case shapeString:
		return "a string"
	case shapeBool:
		return "a bool"
	case shapeBytes:
		return "bytes"
	case shapeList:
		return "a list"
	case shapeSet:
		return "a set"
	case shapeMap:
		return "a map"
	case shapeEnum:
		return "an enum"
	case shapeDecimal:
		return "a decimal, which is a placeholder string until foreign types"
	case shapeTime:
		return "a time"
	case shapeParam:
		return "a type parameter, whose values Go knows nothing about"
	}
	return "a type this backend has no check for"
}

// underlying resolves the value a constraint on a type checks.
func (g *generator) underlying(id *ir.ID, fr *frame) checked {
	var c checked
	seen := map[int32]bool{}
	for {
		t := g.Model.Type(id)
		if t == nil {
			return c
		}
		c.t, c.fr = t, fr
		if ref := t.GetParam(); ref != nil {
			if fr == nil || len(t.GetArgs()) > 0 || int(ref.GetIndex()) >= len(fr.args) {
				c.shape = shapeParam
				return c
			}
			id, fr = fr.args[ref.GetIndex()], fr.outer
			continue
		}
		d := g.Model.Decl(t.GetCtor())
		if d == nil || seen[t.GetCtor().GetIndex()] {
			return c
		}
		seen[t.GetCtor().GetIndex()] = true

		switch {
		case d.GetAlias() != nil:
			id, fr = d.GetAlias().GetTarget(), bind(t.GetArgs(), fr)
			continue
		case d.GetNewtype() != nil:
			c.named = true
			id, fr = d.GetNewtype().GetBase(), bind(t.GetArgs(), fr)
			continue
		case isOption(d, t):
			c.shape = shapeOption
		case d.GetEnumeration() != nil && !emit.Fielded(d.GetEnumeration()):
			c.shape, c.enum = shapeEnum, d
		case d.GetPrimitive() != nil:
			c.shape = primitiveShapes[d.GetMeta().GetName()]
		}
		return c
	}
}

func isOption(d *ir.Decl, t *ir.Type) bool {
	name := d.GetMeta().GetName()
	return d.GetEnumeration() != nil && (name == "Option" || name == "Nullable") && len(t.GetArgs()) == 1
}

// pastNewtypes follows a type through the newtypes it is, to the first type
// that is not one.
func (g *generator) pastNewtypes(id *ir.ID) *ir.ID {
	seen := map[int32]bool{}
	for {
		t := g.Model.Type(id)
		d := g.Model.Decl(t.GetCtor())
		if d == nil || d.GetNewtype() == nil || seen[t.GetCtor().GetIndex()] {
			return id
		}
		seen[t.GetCtor().GetIndex()] = true
		id = d.GetNewtype().GetBase()
	}
}

// constrain writes the checks a value's constraints call for. named says
// the value's Go type is a declared type over the one checked.
func (g *generator) constrain(w *checkWriter, expr string, named bool, id *ir.ID, fr *frame, cs []*ir.Constraint, p checkPath, report func(*ir.Constraint) bool) {
	if len(cs) == 0 {
		return
	}
	v := g.underlying(id, fr)
	v.named = v.named || named

	// An absent value has nothing to check, so an optional one is checked
	// through its pointer when it is there.
	if v.shape == shapeOption {
		saved := w.b
		w.b = &strings.Builder{}
		g.constrain(w, "(*"+expr+")", false, v.t.GetArgs()[0], v.fr, cs, p, report)
		body := w.b.String()
		w.b = saved
		if body != "" {
			w.line("if %s != nil {", expr)
			w.b.WriteString(body)
			w.line("}")
		}
		return
	}

	for _, c := range cs {
		if err := g.constraintCheck(w, c, expr, v, p); err != nil && report(c) {
			g.Warn(err)
		}
	}
}

// constraintCheck writes one constraint's check, or says why it cannot.
func (g *generator) constraintCheck(w *checkWriter, c *ir.Constraint, expr string, v checked, p checkPath) error {
	text, args := constraintText(c), c.GetArgs()
	refuse := func(why string, a ...any) error {
		return emit.Unsupported(c.GetPosition(), "%s is not checked: "+why, append([]any{text}, a...)...)
	}
	str := expr
	if v.named {
		str = "string(" + expr + ")"
	}

	switch c.GetName() {
	case "min", "max":
		if v.shape != shapeInt {
			return refuse("it compares integers, and this is %s", v.describe())
		}
		if len(args) != 1 {
			return refuse("it takes one bound")
		}
		bound, float, err := number(args[0])
		if err != nil {
			return refuse("%v", err)
		}
		lhs, op := expr, "<"
		if float {
			lhs = "float64(" + expr + ")"
		}
		if c.GetName() == "max" {
			op = ">"
		}
		w.line("if %s %s %s {", lhs, op, bound)
		w.errorf(p, text, "got %d", expr)
		w.line("}")

	case "length":
		var count string
		switch v.shape {
		case shapeString:
			// A length is characters, not bytes.
			w.imports["unicode/utf8"] = true
			count = "utf8.RuneCountInString(" + str + ")"
		case shapeBytes, shapeList, shapeSet, shapeMap:
			count = "len(" + expr + ")"
		default:
			return refuse("it measures a string, bytes, or a collection, and this is %s", v.describe())
		}
		if len(args) != 1 {
			return refuse("it takes one length or range")
		}
		cv := w.local("count")
		defer w.release(cv)
		cond, err := lengthCond(args[0], cv)
		if err != nil {
			return refuse("%v", err)
		}
		w.line("if %s := %s; %s {", cv, count, cond)
		w.errorf(p, text, "got %d", cv)
		w.line("}")

	case "matches":
		if v.shape != shapeString {
			return refuse("it matches a string, and this is %s", v.describe())
		}
		if len(args) != 1 || args[0].GetKind() != ir.LiteralKind_LITERAL_KIND_REGEX {
			return refuse("it takes one pattern")
		}
		// A pattern Go's regexp refuses would panic when the package loads.
		pat := args[0].GetText()
		if _, err := regexp.Compile(pat); err != nil {
			return refuse("Go's regexp refuses the pattern: %v", err)
		}
		name := w.pattern(pat)
		w.line("if !%s.MatchString(%s) {", name, str)
		w.errorf(p, text, "no match")
		w.line("}")

	case "oneOf":
		if len(args) == 0 {
			return refuse("it takes the values it allows")
		}
		var conds []string
		for _, a := range args {
			cond, err := g.oneOfCond(expr, v, a)
			if err != nil {
				return refuse("%v", err)
			}
			conds = append(conds, cond)
		}
		w.line("if %s {", strings.Join(conds, " && "))
		switch v.shape {
		case shapeInt:
			w.errorf(p, text, "got %d", expr)
		case shapeBool:
			w.errorf(p, text, "got %t", expr)
		case shapeEnum:
			w.errorf(p, text, "got %q", expr)
		default:
			w.errorf(p, text, "not one of them")
		}
		w.line("}")

	case "unique":
		switch v.shape {
		case shapeSet:
			// A set holds distinct values already.
			return nil
		case shapeList:
		default:
			return refuse("it applies to a list, and this is %s", v.describe())
		}
		if len(v.t.GetArgs()) == 0 {
			return refuse("the list has no element type")
		}
		elem := v.t.GetArgs()[0]
		if !g.comparableIn(elem, v.fr, map[int32]bool{}, nil) {
			return refuse("the list's elements are not comparable in Go")
		}
		et, err := g.typeIn(elem, v.fr)
		if err != nil {
			return refuse("%v", err)
		}
		seen, idx, el, prev, ok := w.local("seen"), w.local("idx"), w.local("elem"), w.local("prev"), w.local("ok")
		defer w.release(seen, idx, el, prev, ok)
		w.line("{")
		w.line("%s := make(map[%s]int, len(%s))", seen, et, expr)
		w.line("for %s, %s := range %s {", idx, el, expr)
		w.line("if %s, %s := %s[%s]; %s {", prev, ok, seen, el, ok)
		w.errorf(p, text, "[%d] repeats [%d]", idx, prev)
		w.line("continue")
		w.line("}")
		w.line("%s[%s] = %s", seen, el, idx)
		w.line("}")
		w.line("}")

	default:
		return emit.Unsupported(c.GetPosition(), "%s is not a constraint the Go backend knows, so it is not checked", c.GetName())
	}
	return nil
}

// oneOfCond is the condition that a value is not one argument of oneOf.
func (g *generator) oneOfCond(expr string, v checked, a *ir.Literal) (string, error) {
	kind := a.GetKind()
	switch {
	case kind == ir.LiteralKind_LITERAL_KIND_LIST:
		return "", errors.New("a list could mean its items or itself")
	case v.shape == shapeString && kind == ir.LiteralKind_LITERAL_KIND_STRING:
		return expr + " != " + strconv.Quote(a.GetText()), nil
	case v.shape == shapeInt && (kind == ir.LiteralKind_LITERAL_KIND_INT || kind == ir.LiteralKind_LITERAL_KIND_FLOAT):
		n, float, err := number(a)
		if err != nil {
			return "", err
		}
		if float {
			return "float64(" + expr + ") != " + n, nil
		}
		return expr + " != " + n, nil
	case v.shape == shapeBool && kind == ir.LiteralKind_LITERAL_KIND_BOOL:
		return expr + " != " + a.GetText(), nil
	case v.shape == shapeEnum && kind == ir.LiteralKind_LITERAL_KIND_NAME:
		enumGo := g.declName(v.enum)
		for _, variant := range v.enum.GetEnumeration().GetVariants() {
			if variant.GetMeta().GetName() != a.GetText() {
				continue
			}
			lhs := expr
			if v.named {
				lhs = enumGo + "(" + expr + ")"
			}
			return lhs + " != " + enumGo + exported(a.GetText()), nil
		}
		return "", fmt.Errorf("%s has no variant %s", v.enum.GetMeta().GetName(), a.GetText())
	case v.shape == shapeEnum && kind == ir.LiteralKind_LITERAL_KIND_STRING:
		return "", errors.New("name the enum's variants rather than quoting them")
	}
	return "", fmt.Errorf("%s cannot be %s", v.describe(), ir.KindName(kind))
}

// number reads a bound, as Go source and whether it is a float.
func number(l *ir.Literal) (string, bool, error) {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_INT:
		n, err := strconv.ParseInt(l.GetText(), 0, 64)
		if err != nil {
			return "", false, fmt.Errorf("%s does not fit in an int64", l.GetText())
		}
		return strconv.FormatInt(n, 10), false, nil
	case ir.LiteralKind_LITERAL_KIND_FLOAT:
		f, err := strconv.ParseFloat(l.GetText(), 64)
		if err != nil {
			return "", false, fmt.Errorf("%s is not a float Go can read", l.GetText())
		}
		return strconv.FormatFloat(f, 'g', -1, 64), true, nil
	}
	return "", false, fmt.Errorf("its bound is %s", ir.KindName(l.GetKind()))
}

// lengthCond is the condition that a count breaks a length.
func lengthCond(l *ir.Literal, cv string) (string, error) {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_INT:
		n, err := strconv.ParseInt(l.GetText(), 0, 64)
		if err != nil {
			return "", fmt.Errorf("%s does not fit in an int64", l.GetText())
		}
		return fmt.Sprintf("%s != %d", cv, n), nil
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		r := l.GetRange()
		var low, high *int64
		if r != nil {
			low, high = r.Low, r.High
		}
		switch {
		case low == nil && high == nil:
			return "", errors.New("a range with neither end constrains nothing")
		case low != nil && high != nil && *low > *high:
			return "", fmt.Errorf("%d..%d can never hold", *low, *high)
		case low == nil:
			return fmt.Sprintf("%s > %d", cv, *high), nil
		case high == nil:
			return fmt.Sprintf("%s < %d", cv, *low), nil
		}
		return fmt.Sprintf("%s < %d || %s > %d", cv, *low, cv, *high), nil
	}
	return "", fmt.Errorf("its length is %s", ir.KindName(l.GetKind()))
}

// visit writes the calls validating what a value holds: the value when its
// type validates, what a pointer points at, and each element of a
// collection. convert says the value's Go type is a newtype over the type
// visited, so that type's method needs a conversion.
//
// A type parameter's values are not visited: asserting a method on a T that
// holds a nil pointer panics.
func (g *generator) visit(w *checkWriter, expr string, convert bool, id *ir.ID, fr *frame, p checkPath) {
	if !g.needsVisit(id, fr, map[int32]bool{}) {
		return
	}
	t := g.Model.Type(id)
	if ref := t.GetParam(); ref != nil {
		g.visit(w, expr, convert, fr.args[ref.GetIndex()], fr.outer, p)
		return
	}
	d := g.Model.Decl(t.GetCtor())
	name, args := d.GetMeta().GetName(), t.GetArgs()

	switch {
	case d.GetAlias() != nil:
		g.visit(w, expr, convert, d.GetAlias().GetTarget(), bind(args, fr), p)
	case isOption(d, t):
		w.line("if %s != nil {", expr)
		g.visit(w, "(*"+expr+")", false, args[0], fr, p)
		w.line("}")
	case d.GetPrimitive() != nil && name == "List":
		idx, elem := w.local("idx"), w.local("elem")
		w.line("for %s, %s := range %s {", idx, elem, expr)
		g.visit(w, elem, false, args[0], fr, p.child("[%d]", idx))
		w.line("}")
		w.release(idx, elem)
	case d.GetPrimitive() != nil && name == "Set":
		key := w.local("key")
		label, largs := g.keyLabel(args[0], fr, key)
		w.line("for %s := range %s {", key, expr)
		g.visit(w, key, false, args[0], fr, p.child("{"+label+"}", largs...))
		w.line("}")
		w.release(key)
	case d.GetPrimitive() != nil && name == "Map":
		key, val := w.local("key"), w.local("val")
		label, largs := g.keyLabel(args[0], fr, key)
		visitKey, visitVal := g.needsVisit(args[0], fr, map[int32]bool{}), g.needsVisit(args[1], fr, map[int32]bool{})
		keyVar, valVar := key, val
		if !visitKey && len(largs) == 0 {
			keyVar = "_"
		}
		if !visitVal {
			valVar = "_"
		}
		w.line("for %s, %s := range %s {", keyVar, valVar, expr)
		if visitKey {
			g.visit(w, key, false, args[0], fr, p.child("{"+label+"}", largs...))
		}
		if visitVal {
			g.visit(w, val, false, args[1], fr, p.child("["+label+"]", largs...))
		}
		w.line("}")
		w.release(key, val)
	case d.GetEnumeration() != nil && emit.Fielded(d.GetEnumeration()):
		// An interface cannot carry the methods, so the value it holds is
		// asked; a nil interface holds nothing to ask.
		inner, ok := w.local("inner"), w.local("ok")
		w.line("if %s, %s := %s.(interface{ validate(string, []error) []error }); %s {", inner, ok, expr, ok)
		w.line("%s = %s.validate(%s, %s)", w.errs, inner, p.expr(w), w.errs)
		w.line("}")
		w.release(inner, ok)
	default:
		if convert {
			if goT, err := g.typeIn(id, fr); err == nil {
				expr = goT + "(" + expr + ")"
			}
		}
		w.line("%s = %s.validate(%s, %s)", w.errs, expr, p.expr(w), w.errs)
	}
}

// keyLabel is how a path names a set element or map key: by its value when
// that is a number or an enum's, and as ? otherwise, since a key can hold
// what a log should not.
func (g *generator) keyLabel(id *ir.ID, fr *frame, key string) (string, []string) {
	switch g.underlying(id, fr).shape {
	case shapeInt, shapeEnum:
		return "%v", []string{key}
	}
	return "?", nil
}

// needsVisit reports whether a value of a type holds anything to validate.
func (g *generator) needsVisit(id *ir.ID, fr *frame, seen map[int32]bool) bool {
	t := g.Model.Type(id)
	if t == nil {
		return false
	}
	if ref := t.GetParam(); ref != nil {
		if fr == nil || len(t.GetArgs()) > 0 || int(ref.GetIndex()) >= len(fr.args) {
			return false
		}
		return g.needsVisit(fr.args[ref.GetIndex()], fr.outer, seen)
	}
	d := g.Model.Decl(t.GetCtor())
	if d == nil {
		return false
	}
	idx := t.GetCtor().GetIndex()
	name, args := d.GetMeta().GetName(), t.GetArgs()

	switch {
	case d.GetAlias() != nil:
		if seen[idx] {
			return false
		}
		seen[idx] = true
		defer delete(seen, idx)
		return g.needsVisit(d.GetAlias().GetTarget(), bind(args, fr), seen)
	case isOption(d, t):
		return g.needsVisit(args[0], fr, seen)
	case d.GetPrimitive() != nil && (name == "List" || name == "Set"):
		return len(args) > 0 && g.needsVisit(args[0], fr, seen)
	case d.GetPrimitive() != nil && name == "Map":
		return len(args) > 1 && (g.needsVisit(args[0], fr, seen) || g.needsVisit(args[1], fr, seen))
	case !emit.IsOwn(d):
		return false
	case d.GetEnumeration() != nil && emit.Fielded(d.GetEnumeration()):
		for i := range d.GetEnumeration().GetVariants() {
			if g.valid[validKey{idx, i}] {
				return true
			}
		}
		return false
	}
	return g.valid[validKey{idx, -1}]
}

// constraintText is a constraint as it was written.
func constraintText(c *ir.Constraint) string {
	if len(c.GetArgs()) == 0 {
		return c.GetName()
	}
	args := make([]string, len(c.GetArgs()))
	for i, a := range c.GetArgs() {
		args[i] = literalText(a)
	}
	return c.GetName() + "(" + strings.Join(args, ", ") + ")"
}

func literalText(l *ir.Literal) string {
	switch l.GetKind() {
	case ir.LiteralKind_LITERAL_KIND_STRING:
		return strconv.Quote(l.GetText())
	case ir.LiteralKind_LITERAL_KIND_REGEX:
		return "/" + l.GetText() + "/"
	case ir.LiteralKind_LITERAL_KIND_LIST:
		items := make([]string, len(l.GetItems()))
		for i, item := range l.GetItems() {
			items[i] = literalText(item)
		}
		return "[" + strings.Join(items, ", ") + "]"
	case ir.LiteralKind_LITERAL_KIND_RANGE:
		var s string
		if r := l.GetRange(); r != nil {
			if r.Low != nil {
				s = strconv.FormatInt(*r.Low, 10)
			}
			s += ".."
			if r.High != nil {
				s += strconv.FormatInt(*r.High, 10)
			}
		}
		return s
	}
	return l.GetText()
}

// tdlName is a declaration's name without its package, which is how a path
// names it.
func tdlName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
