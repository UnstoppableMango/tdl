// Package emit is what every code generator in backend/ shares: which
// declarations are the model's own, reading directives for one target,
// reporting what cannot be generated, and resolving a type reference to the
// shapes the prelude spells.
//
// It knows nothing about any target language. A backend owns its type
// mapping, its naming, and its output; this package owns the parts that
// would otherwise be written once per backend and drift.
//
// It is internal to backend/ and free to change.
package emit

import (
	"errors"
	"fmt"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
	"github.com/unstoppablemango/tdl/prelude"
)

// Session is one request's state. A backend makes a fresh one per Generate
// call, so a reused connection shares nothing between requests.
type Session struct {
	Model  *ir.Model
	Target string

	// Lang is the target language as a message names it: "Go",
	// "Protobuf", and so on.
	Lang string

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

// Own returns the declarations the model's own file declared.
//
// The prelude is merged into the declaration table untagged, so a model
// whose source declares two things arrives with twenty-one declarations. A
// backend that emits per declaration has to decide what is the user's, and
// which file a declaration came from is what says so.
//
// The embedded prelude is named [prelude.Name] and nothing else is: it is
// parsed under that name rather than read from a path, so the comparison is
// against the whole name and not its ending. A user's `my-std.tdl`, or a
// `std.tdl` of their own in any directory, is theirs and is generated.
//
// A replacement prelude passed to `sema.WithPrelude` is named by whoever
// passed it and is not recognized here. Marking the prelude on the wire is
// the fix, and `plugins.md` argues the opposite, that a backend should see
// prelude declarations as declarations like any other; until that is
// settled, a project replacing the prelude generates it too.
func (s *Session) Own() []*ir.Decl {
	var own []*ir.Decl
	for _, d := range s.Model.GetDecls() {
		if d.GetMeta().GetPosition().GetFilename() == prelude.Name {
			continue
		}
		own = append(own, d)
	}
	return own
}

// Find returns a directive carrying at least one argument.
//
// A model carries directives for every target block in it, tagged with the
// block they came from, so this filters rather than assuming what it is
// handed is its own.
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

// Block reads a directive written on the target block itself rather than
// against a node in the model.
//
// It returns the directive rather than its argument, because a diagnostic
// about the value wants the position it was written at.
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

// UnsupportedError reports a shape the backend cannot express. It reaches
// the user as a warning rather than stopping the run, so a model that is
// mostly generatable generates.
type UnsupportedError struct {
	What     string
	Position *ir.Position
}

func (e *UnsupportedError) Error() string { return e.What }

// Unsupported returns an [UnsupportedError] at a position.
func Unsupported(pos *ir.Position, format string, args ...any) error {
	return &UnsupportedError{What: fmt.Sprintf(format, args...), Position: pos}
}

// Warn reports something the backend cannot handle, with a position when
// the failure carried one.
//
// A backend says what it cannot do here rather than returning an error,
// because this reaches the user with a position attached and does not stop
// the run.
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

// WarnWhere says out loud that a newtype's `where` constraints are not
// enforced. The newtype is still emitted: skipping it would leave every
// field naming it referring to something the output does not declare.
func (s *Session) WarnWhere(d *ir.Decl) {
	if n := len(d.GetNewtype().GetValueConstraints()); n > 0 {
		s.Warn(Unsupported(d.GetMeta().GetPosition(),
			"%s carries %d where constraint(s), and validation is not generated yet",
			d.GetMeta().GetName(), n))
	}
}

// Fielded reports whether any variant of an enum carries fields, which is
// what separates a plain enum from a sum type in every target.
func Fielded(e *ir.Enum) bool {
	for _, v := range e.GetVariants() {
		if len(v.GetFields()) > 0 {
			return true
		}
	}
	return false
}

// Doc returns a node's documentation, one trimmed line per entry.
func Doc(m *ir.Meta) []string {
	lines := make([]string, len(m.GetDoc()))
	for i, line := range m.GetDoc() {
		lines[i] = trim(line)
	}
	return lines
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
