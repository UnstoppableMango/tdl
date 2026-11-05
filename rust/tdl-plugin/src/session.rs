use std::collections::HashMap;

use crate::Error;
use crate::backend::directives_for;
use crate::case::{self, Style};
use crate::ir::{
    Constraint, Decl, Directive, Field, ID, Meta, Model, Param, Position, Variant, decl::Node,
};
use crate::problem::{Problem, loss};
use crate::protocol::{
    Analysis, ConstraintView, DeclView, Diagnostic, File, MemberView, Request, Response, Severity,
    TypeView,
};

/// One request's state: the model, the analysis tdl sent with it, and the
/// diagnostics the backend collects. A backend makes a fresh one per
/// request.
///
/// The views are looked up by the node's identity in the request, so a node
/// cloned out of it has none.
pub struct Session<'a> {
    request: &'a Request,
    analysis: &'a Analysis,
    lang: String,
    decls: HashMap<*const Decl, usize>,
    members: HashMap<*const (), &'a MemberView>,
    constraints: HashMap<*const Constraint, &'a ConstraintView>,
    /// Makes [`lose`](Session::lose) report nothing, since annotations
    /// carry what it would.
    pub roundtrip: bool,
    /// Makes [`lose`](Session::lose) report read-back loss, which it
    /// otherwise leaves out: a backend sets it from a bare `readback`
    /// directive in its target block.
    pub readback: bool,
    /// The diagnostics collected so far.
    pub diagnostics: Vec<Diagnostic>,
}

impl<'a> Session<'a> {
    /// Starts a session for one request. `lang` is the target language as a
    /// message names it. It fails when the request carries no analysis for
    /// its model, which tdl sends with every request.
    pub fn new(request: &'a Request, lang: impl Into<String>) -> Result<Self, Error> {
        let analysis = request.analysis.as_option().ok_or_else(|| {
            Error::protocol("the request carries no analysis; tdl sends one with every request")
        })?;
        let model: &Model = &request.model;
        if analysis.decls.len() != model.decls.len() || analysis.types.len() != model.types.len() {
            return Err(Error::protocol(
                "the request's analysis does not describe its model",
            ));
        }

        let mut s = Session {
            request,
            analysis,
            lang: lang.into(),
            decls: HashMap::new(),
            members: HashMap::new(),
            constraints: HashMap::new(),
            roundtrip: false,
            readback: false,
            diagnostics: Vec::new(),
        };
        for (i, (d, view)) in model.decls.iter().zip(&analysis.decls).enumerate() {
            s.decls.insert(d, i);
            if let Some(Node::Newtype(n)) = &d.node {
                s.index_constraints(&n.value_constraints, &view.constraints);
            }
            s.index_fields(fields(d), &view.fields);
            for (v, vv) in variants(d).iter().zip(&view.variants) {
                s.members.insert(ptr(v), vv);
                s.index_fields(&v.fields, &vv.fields);
            }
        }
        Ok(s)
    }

    fn index_fields(&mut self, fields: &'a [Field], views: &'a [MemberView]) {
        for (f, view) in fields.iter().zip(views) {
            self.members.insert(ptr(f), view);
            self.index_constraints(&f.constraints, &view.constraints);
        }
    }

    fn index_constraints(&mut self, cs: &'a [Constraint], views: &'a [ConstraintView]) {
        for (c, view) in cs.iter().zip(views) {
            self.constraints.insert(c, view);
        }
    }

    /// The request being served.
    pub fn request(&self) -> &'a Request {
        self.request
    }

    /// The model, with the prelude's declarations merged into it.
    pub fn model(&self) -> &'a Model {
        &self.request.model
    }

    /// The target block being served.
    pub fn target(&self) -> &'a str {
        &self.request.target
    }

    /// The target language as a message names it.
    pub fn lang(&self) -> &str {
        &self.lang
    }

    /// What tdl derived from the model for this target.
    pub fn analysis(&self) -> &'a Analysis {
        self.analysis
    }

    /// The response holding files and every diagnostic collected.
    pub fn response(self, files: Vec<File>) -> Response {
        Response {
            files,
            diagnostics: self.diagnostics,
            ..Default::default()
        }
    }

    /// The analysis of a declaration in the model, or `None` for one not in
    /// it.
    pub fn decl_view(&self, d: &Decl) -> Option<&'a DeclView> {
        self.decls
            .get(&(d as *const Decl))
            .map(|&i| &self.analysis.decls[i])
    }

    /// The analysis of a field of a declaration in the model, or `None`.
    pub fn field_view(&self, f: &Field) -> Option<&'a MemberView> {
        self.members.get(&ptr(f)).copied()
    }

    /// The analysis of an enum's variant in the model, or `None`.
    pub fn variant_view(&self, v: &Variant) -> Option<&'a MemberView> {
        self.members.get(&ptr(v)).copied()
    }

    /// The analysis of a constraint on a newtype or a field in the model, or
    /// `None`.
    pub fn constraint_view(&self, c: &Constraint) -> Option<&'a ConstraintView> {
        self.constraints.get(&(c as *const Constraint)).copied()
    }

    /// A constraint as TDL source writes it, as `min(0)`.
    pub fn source<'c>(&self, c: &'c Constraint) -> &'c str
    where
        'a: 'c,
    {
        self.constraint_view(c).map_or(&c.name, |v| &v.source)
    }

    /// The analysis of a type reference, or `None` when it does not resolve.
    pub fn type_view(&self, id: &ID) -> Option<&'a TypeView> {
        usize::try_from(id.index)
            .ok()
            .and_then(|i| self.analysis.types.get(i))
    }

    /// The declaration an ID names, or `None` when it does not resolve.
    pub fn decl_at(&self, id: &ID) -> Option<&'a Decl> {
        usize::try_from(id.index)
            .ok()
            .and_then(|i| self.model().decls.get(i))
    }

    /// The declarations that did not come from the embedded prelude.
    pub fn own(&self) -> Vec<&'a Decl> {
        self.model()
            .decls
            .iter()
            .zip(&self.analysis.decls)
            .filter(|(_, v)| v.own)
            .map(|(d, _)| d)
            .collect()
    }

    /// The session target's directive with this name carrying at least one
    /// argument, or `None`.
    pub fn find<'d>(&self, all: &'d [Directive], name: &str) -> Option<&'d Directive> {
        directives_for(self.target(), all).find(|d| d.name == name && !d.args.is_empty())
    }

    /// The first argument of a directive on a node, as written, or `None`.
    pub fn text<'d>(&self, all: &'d [Directive], name: &str) -> Option<&'d str> {
        self.find(all, name).map(|d| d.args[0].text.as_str())
    }

    /// A directive written on the target block itself, returned whole so a
    /// diagnostic about its value has its position.
    pub fn block(&self, name: &str) -> Option<&'a Directive> {
        self.model()
            .targets
            .iter()
            .filter(|b| b.meta.name == self.target())
            .find_map(|b| self.find(&b.directives, name))
    }

    /// Whether the session target carries a directive with this name among
    /// `all`, with or without arguments.
    pub fn tagged(&self, all: &[Directive], name: &str) -> bool {
        directives_for(self.target(), all).any(|d| d.name == name)
    }

    /// Whether this target's block carries a directive with this name, such
    /// as a bare `readback`.
    pub fn bare(&self, name: &str) -> bool {
        self.model()
            .targets
            .iter()
            .any(|b| b.meta.name == self.target() && self.tagged(&b.directives, name))
    }

    /// A declaration's name in the target language: a `name` directive's
    /// argument when there is one, otherwise its words joined by style.
    pub fn decl_name(&self, d: &Decl, style: Style) -> String {
        let words = self.decl_view(d).map(|v| v.words.as_slice());
        self.name(
            &d.directives,
            words,
            case::last_segment(&d.meta.name),
            style,
        )
    }

    /// A field's name in the target language, by the rule
    /// [`decl_name`](Session::decl_name) follows.
    pub fn field_name(&self, f: &Field, style: Style) -> String {
        let words = self.field_view(f).map(|v| v.words.as_slice());
        self.name(&f.directives, words, &f.meta.name, style)
    }

    /// A variant's name in the target language, by the rule
    /// [`decl_name`](Session::decl_name) follows.
    pub fn variant_name(&self, v: &Variant, style: Style) -> String {
        let words = self.variant_view(v).map(|view| view.words.as_slice());
        self.name(&v.directives, words, &v.meta.name, style)
    }

    // The rule the three names share: a name directive, else the shipped
    // words, else the words of the name as written, for a node the request
    // does not hold.
    fn name(
        &self,
        directives: &[Directive],
        words: Option<&[String]>,
        written: &str,
        style: Style,
    ) -> String {
        if let Some(name) = self.text(directives, "name") {
            return name.to_string();
        }
        match words {
            Some(words) => style(words),
            None => style(&case::words(written)),
        }
    }

    /// Reports a problem: an [`Unsupported`](Problem::Unsupported) as a
    /// warning with its loss code, and an [`Invalid`](Problem::Invalid) as
    /// an error.
    pub fn warn(&mut self, problem: Problem) {
        let d = match problem {
            Problem::Unsupported {
                message,
                position,
                code,
            } => diagnostic(Severity::Warning, message, position, code),
            Problem::Invalid { message, position } => {
                diagnostic(Severity::Error, message, position, "")
            }
        };
        self.diagnostics.push(d);
    }

    /// Reports a problem that makes the output unusable. The host writes
    /// nothing when a response carries one.
    pub fn error(&mut self, position: Option<&Position>, message: impl Into<String>) {
        self.diagnostics.push(diagnostic(
            Severity::Error,
            message.into(),
            position.cloned(),
            "",
        ));
    }

    /// Warns that a fact is lost, with the loss code that names it. It
    /// reports output loss: what the output itself drops or weakens, which
    /// annotations do not restore, so it warns under roundtrip too.
    /// [`lose`](Session::lose) reports read-back loss.
    pub fn lossy(
        &mut self,
        code: &'static str,
        position: Option<&Position>,
        message: impl Into<String>,
    ) {
        self.diagnostics.push(diagnostic(
            Severity::Warning,
            message.into(),
            position.cloned(),
            code,
        ));
    }

    /// Warns that a fact the output keeps is one reading it back would not
    /// recover. It reports only under [`readback`](Session::readback), and
    /// never under [`roundtrip`](Session::roundtrip), where an annotation
    /// carries the fact. [`lossy`](Session::lossy) reports output loss.
    pub fn lose(
        &mut self,
        code: &'static str,
        position: Option<&Position>,
        message: impl Into<String>,
    ) {
        if self.readback && !self.roundtrip {
            self.lossy(code, position, message);
        }
    }

    /// Reports a declaration that is not generated. It is output loss, so it
    /// warns under roundtrip too, where the declaration is carried as TDL but
    /// the output still lacks it.
    pub fn skip(&mut self, problem: Problem) {
        self.warn(problem);
    }

    /// Warns that a declaration's constraints are not enforced: a newtype's
    /// where block and each field's, including enum variants'. `enforced`
    /// reports the constraints the target's types enforce themselves, which
    /// do not warn.
    pub fn warn_constraints(&mut self, d: &Decl, enforced: impl Fn(&Constraint) -> bool) {
        let name = &d.meta.name;
        let unenforced = |cs: &[Constraint]| cs.iter().filter(|c| !enforced(c)).count();
        if let Some(Node::Newtype(n)) = &d.node {
            let n = unenforced(&n.value_constraints);
            if n > 0 {
                self.lossy(
                    loss::CONSTRAINT,
                    d.meta.position.as_option(),
                    format!("{name} carries {n} where constraint(s), and validation is not generated yet"),
                );
            }
        }

        let fields = fields(d)
            .iter()
            .chain(variants(d).iter().flat_map(|v| &v.fields));
        for f in fields {
            let m = unenforced(&f.constraints);
            if m > 0 {
                self.lossy(
                    loss::CONSTRAINT,
                    f.meta.position.as_option(),
                    format!(
                        "{}.{} carries {m} constraint(s), and validation is not generated yet",
                        case::last_segment(name),
                        f.meta.name
                    ),
                );
            }
        }
    }
}

fn diagnostic(
    severity: Severity,
    message: String,
    position: Option<Position>,
    code: &str,
) -> Diagnostic {
    Diagnostic {
        severity: severity.into(),
        message,
        position: position.into(),
        code: code.to_string(),
        ..Default::default()
    }
}

fn ptr<T>(t: &T) -> *const () {
    (t as *const T).cast()
}

/// A struct declaration's fields; an enum's belong to its variants.
pub fn fields(d: &Decl) -> &[Field] {
    match &d.node {
        Some(Node::Structure(s)) => &s.fields,
        _ => &[],
    }
}

/// An enum declaration's variants.
pub fn variants(d: &Decl) -> &[Variant] {
    match &d.node {
        Some(Node::Enumeration(e)) => &e.variants,
        _ => &[],
    }
}

/// The type parameters of a declaration.
pub fn params(d: &Decl) -> &[Param] {
    match &d.node {
        Some(Node::Alias(a)) => &a.params,
        Some(Node::Newtype(n)) => &n.params,
        Some(Node::Structure(s)) => &s.params,
        Some(Node::Enumeration(e)) => &e.params,
        Some(Node::Class(c)) => &c.params,
        _ => &[],
    }
}

/// Whether a schema backend writes anything for a declaration, or the
/// problem that keeps it from generating one: an alias is expanded where it
/// is used and a primitive names an opaque type, so neither declares
/// anything, while classes, units, and type parameters are not generated
/// yet.
pub fn declares(d: &Decl) -> Result<bool, Problem> {
    let pos = d.meta.position.as_option();
    let name = &d.meta.name;
    match &d.node {
        Some(Node::Class(_)) => Err(Problem::unsupported(
            format!("{name} is a class, and classes are not generated yet"),
            pos,
        )),
        Some(Node::Unit(_)) => Err(Problem::unsupported(
            format!("{name} is a unit, and units are not generated yet"),
            pos,
        )),
        Some(Node::Structure(_) | Node::Enumeration(_) | Node::Newtype(_)) => {
            if params(d).is_empty() {
                Ok(true)
            } else {
                Err(Problem::unsupported(
                    format!("{name} is parameterized, and generics are not generated yet"),
                    pos,
                ))
            }
        }
        _ => Ok(false),
    }
}

/// A node's documentation, one line per entry, with trailing whitespace
/// removed and indentation kept.
pub fn doc(m: &Meta) -> Vec<&str> {
    m.doc
        .iter()
        .map(|l| l.trim_end_matches([' ', '\t']))
        .collect()
}

/// Whether a node is deprecated, with the reason, which may be empty.
pub fn deprecated(m: &Meta) -> Option<&str> {
    m.deprecated.as_option().map(|d| d.reason.as_str())
}
