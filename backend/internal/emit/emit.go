// Package emit holds what every code generator in backend/ shares: which
// declarations are the model's own, reading directives for one target,
// reporting what cannot be generated, and resolving a type reference to the
// prelude's shapes. It knows nothing about any target language.
package emit

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
)

// Session is one request's state. A backend makes a fresh one per Generate
// call.
type Session struct {
	Model  *ir.Model
	Target string

	// Lang is the target language as a message names it: "Go",
	// "Protobuf", and so on.
	Lang string

	// Externs makes [Session.Resolve] return an [Extern] ref for an extern
	// rather than refusing it.
	Externs bool

	Diags []*plugin.Diagnostic
}

// NewSession starts a session for one request.
func NewSession(req *plugin.Request, lang string) *Session {
	return &Session{Model: req.GetModel(), Target: req.GetTarget(), Lang: lang}
}

// Response returns the files with every diagnostic the session collected.
func (s *Session) Response(files []*plugin.File) *plugin.Response {
	return &plugin.Response{Files: files, Diagnostics: s.Diags}
}

// Own returns the declarations that did not come from the prelude, which is
// merged into the declaration table untagged.
//
// The embedded prelude's filename is exactly [prelude.Name], so a user's
// `std.tdl` in any directory is still generated. A replacement prelude
// passed to `sema.WithPrelude` is not recognized and is generated too.
func (s *Session) Own() []*ir.Decl {
	var own []*ir.Decl
	for _, d := range s.Model.GetDecls() {
		if IsOwn(d) {
			own = append(own, d)
		}
	}
	return own
}

// IsOwn reports whether a declaration is the model's rather than the
// prelude's, by the rule [Session.Own] describes.
func IsOwn(d *ir.Decl) bool {
	return d.GetMeta().GetPosition().GetFilename() != prelude.Name
}

// Find returns the session target's directive with this name carrying at
// least one argument.
func (s *Session) Find(all []*ir.Directive, name string) (*ir.Directive, bool) {
	for _, d := range plugin.Directives(s.Target, all) {
		if d.GetName() != name {
			continue
		}
		if len(d.GetArgs()) > 0 {
			return d, true
		}
	}
	return nil, false
}

// Text returns the first argument of a directive on a node, as written.
func (s *Session) Text(all []*ir.Directive, name string) (string, bool) {
	if d, ok := s.Find(all, name); ok {
		return d.GetArgs()[0].GetText(), true
	}
	return "", false
}

// Block reads a directive written on the target block itself. It returns
// the directive so a diagnostic about the value has its position.
func (s *Session) Block(name string) (*ir.Directive, bool) {
	for _, block := range s.Model.GetTargets() {
		if block.GetMeta().GetName() != s.Target {
			continue
		}
		if d, ok := s.Find(block.GetDirectives(), name); ok {
			return d, true
		}
	}
	return nil, false
}

// DeclName is a declaration's name in the target language: a `name`
// directive's argument when there is one, and otherwise the last segment of
// the TDL name passed through style.
func (s *Session) DeclName(d *ir.Decl, style func(string) string) string {
	if n, ok := s.Text(d.GetDirectives(), "name"); ok {
		return n
	}
	return style(LastSegment(d.GetMeta().GetName()))
}

// FieldName is a field's name in the target language, by the same rule as
// [Session.DeclName].
func (s *Session) FieldName(f *ir.Field, style func(string) string) string {
	if n, ok := s.Text(f.GetDirectives(), "name"); ok {
		return n
	}
	return style(f.GetMeta().GetName())
}

// VariantName is an enum variant's name in the target language, by the same
// rule as [Session.DeclName].
func (s *Session) VariantName(v *ir.Variant, style func(string) string) string {
	if n, ok := s.Text(v.GetDirectives(), "name"); ok {
		return n
	}
	return style(v.GetMeta().GetName())
}

// UnsupportedError reports a shape the backend cannot express. It reaches
// the user as a warning.
type UnsupportedError struct {
	What     string
	Position *ir.Position
}

func (e *UnsupportedError) Error() string { return e.What }

// Unsupported returns an [UnsupportedError] at a position.
func Unsupported(pos *ir.Position, format string, args ...any) error {
	return &UnsupportedError{What: fmt.Sprintf(format, args...), Position: pos}
}

// Warn reports something the backend cannot handle as a warning, with a
// position when err is an [UnsupportedError].
func (s *Session) Warn(err error) {
	d := &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_WARNING,
		Message:  err.Error(),
	}
	if u, ok := errors.AsType[*UnsupportedError](err); ok {
		d.Position = u.Position
	}
	s.Diags = append(s.Diags, d)
}

// Error reports a problem that makes the output unusable. The host writes
// nothing when a response carries one.
func (s *Session) Error(pos *ir.Position, format string, args ...any) {
	s.Diags = append(s.Diags, &plugin.Diagnostic{
		Severity: plugin.Severity_SEVERITY_ERROR,
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
	})
}

// WarnWhere warns that a newtype's `where` constraints are not enforced.
// The newtype is still emitted, since fields naming it need it declared.
func (s *Session) WarnWhere(d *ir.Decl) {
	if n := len(d.GetNewtype().GetValueConstraints()); n > 0 {
		s.Warn(Unsupported(d.GetMeta().GetPosition(),
			"%s carries %d where constraint(s), and validation is not generated yet",
			d.GetMeta().GetName(), n))
	}
}

// WarnConstraints warns that a declaration's constraints are not enforced:
// a newtype's `where` block and each field's, including enum variants'.
func (s *Session) WarnConstraints(d *ir.Decl) {
	s.WarnWhere(d)

	fields := slices.Clone(d.Fields())
	for _, v := range d.GetEnumeration().GetVariants() {
		fields = append(fields, v.GetFields()...)
	}
	for _, f := range fields {
		if n := len(f.GetConstraints()); n > 0 {
			s.Warn(Unsupported(f.GetMeta().GetPosition(),
				"%s.%s carries %d constraint(s), and validation is not generated yet",
				LastSegment(d.GetMeta().GetName()), f.GetMeta().GetName(), n))
		}
	}
}

// Declares reports whether a schema backend writes anything for a
// declaration, or why it cannot generate one: an alias is expanded where it
// is used and a primitive names an opaque type, so neither declares
// anything, while classes, units, and type parameters are not generated
// yet.
func Declares(d *ir.Decl) (bool, error) {
	pos := d.GetMeta().GetPosition()
	name := d.GetMeta().GetName()

	switch {
	case d.GetClass() != nil:
		return false, Unsupported(pos, "%s is a class, and classes are not generated yet", name)
	case d.GetUnit() != nil:
		return false, Unsupported(pos, "%s is a unit, and units are not generated yet", name)
	case d.GetStructure() == nil && d.GetEnumeration() == nil && d.GetNewtype() == nil:
		return false, nil
	}
	if len(d.Params()) > 0 {
		return false, Unsupported(pos, "%s is parameterized, and generics are not generated yet", name)
	}
	return true, nil
}

// Fielded reports whether any variant of an enum carries fields, making it
// a sum type rather than a plain enum.
func Fielded(e *ir.Enum) bool {
	for _, v := range e.GetVariants() {
		if len(v.GetFields()) > 0 {
			return true
		}
	}
	return false
}

// Doc returns a node's documentation, one line per entry, with trailing
// whitespace removed and indentation kept.
func Doc(m *ir.Meta) []string {
	lines := make([]string, len(m.GetDoc()))
	for i, line := range m.GetDoc() {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return lines
}

// DocComment writes a node's documentation and deprecation as a /** */
// block, the deprecation as an @deprecated tag, as JSDoc and ApexDoc read
// it.
func DocComment(b *strings.Builder, indent string, m *ir.Meta) {
	lines := Doc(m)
	if reason, ok := Deprecated(m); ok {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, strings.TrimSpace("@deprecated "+reason))
	}
	BlockComment(b, indent, lines)
}

// BlockComment writes lines as a /** */ block, or nothing when there are
// none. An empty line is a bare " *".
func BlockComment(b *strings.Builder, indent string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "%s/**\n", indent)
	for _, line := range lines {
		if line == "" {
			fmt.Fprintf(b, "%s *\n", indent)
			continue
		}
		fmt.Fprintf(b, "%s * %s\n", indent, strings.ReplaceAll(line, "*/", "* /"))
	}
	fmt.Fprintf(b, "%s */\n", indent)
}

// Deprecated returns the reason a node is deprecated, which may be empty,
// and whether it is.
func Deprecated(m *ir.Meta) (string, bool) {
	d := m.GetDeprecated()
	if d == nil {
		return "", false
	}
	return d.GetReason(), true
}
