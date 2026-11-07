use crate::Error;
use crate::ir::Directive;
use crate::protocol::{
    DirectiveSpec, Features, HandshakeReply, ImportRequest, ImportResponse, Numbering, Placement,
    Request, Response,
};

/// A backend turns a resolved model into files.
pub trait Backend {
    /// Reports what this backend is and what directives it understands. tdl
    /// checks a target block against them before generating.
    fn describe(&self) -> Description;

    /// Returns the files for one request. It fails only when it cannot
    /// produce a response at all; a problem with the model belongs in the
    /// response's diagnostics.
    fn generate(&self, request: &Request) -> Result<Response, Error>;

    /// Returns the model the request's files describe, or TDL source the
    /// host lowers, for `tdl import`. A backend that imports overrides it
    /// and sets [`Description::reverse`]; [`serve`](crate::serve) refuses
    /// an import handshake to one that does not.
    fn import(&self, request: &ImportRequest) -> Result<ImportResponse, Error> {
        let _ = request;
        Err(Error::protocol(format!(
            "{} does not import; it generates only",
            self.describe().name
        )))
    }
}

/// What a backend says about itself, answered in the handshake.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Description {
    /// What the backend is called, in a target block and as
    /// `tdl-gen-<name>` on PATH.
    pub name: String,
    /// The backend's own version.
    pub version: String,
    /// The directives the backend understands.
    pub directives: Vec<DirectiveSpec>,
    /// Says the backend can serve more than one request on a connection.
    pub reuse: bool,
    /// Says the backend overrides [`Backend::import`]. Set without it, the
    /// handshake accepts an import whose first request then fails.
    pub reverse: bool,
    /// How the backend numbers fields and variants, or `None` for a backend
    /// that does not.
    pub numbering: Option<Numbering>,
    /// Says the same request may not produce the same files, so the host
    /// keeps what it wrote while the request is unchanged.
    pub nondeterministic: bool,
}

impl Description {
    /// A description with a name and a version and nothing else.
    pub fn new(name: impl Into<String>, version: impl Into<String>) -> Self {
        Description {
            name: name.into(),
            version: version.into(),
            ..Default::default()
        }
    }

    /// The handshake reply carrying this description.
    pub fn reply(&self) -> HandshakeReply {
        HandshakeReply {
            accepted: true,
            name: self.name.clone(),
            version: self.version.clone(),
            directives: self.directives.clone(),
            features: Features {
                reuse: self.reuse,
                reverse: self.reverse,
                nondeterministic: self.nondeterministic,
                ..Default::default()
            }
            .into(),
            numbering: self.numbering.clone().into(),
            ..Default::default()
        }
    }
}

/// The directives in `all` that belong to `target`. A request from tdl
/// carries only its target's directives, so this matters for a model built
/// some other way.
pub fn directives_for<'t, 'd>(
    target: &'t str,
    all: &'d [Directive],
) -> impl Iterator<Item = &'d Directive> + use<'t, 'd> {
    all.iter().filter(move |d| d.target == target)
}

/// Where a name for a declaration, a field, or a variant goes.
pub const ON_MEMBERS: [Placement; 3] = [
    Placement::PLACEMENT_DECL,
    Placement::PLACEMENT_FIELD,
    Placement::PLACEMENT_VARIANT,
];

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn a_reply_carries_the_description() {
        let desc = Description {
            reuse: true,
            nondeterministic: true,
            directives: vec![DirectiveSpec {
                name: "note".into(),
                ..Default::default()
            }],
            ..Description::new("mine", "1.2.3")
        };

        let reply = desc.reply();

        assert!(reply.accepted);
        assert_eq!(reply.name, "mine");
        assert_eq!(reply.version, "1.2.3");
        assert_eq!(reply.directives, desc.directives);
        assert!(reply.features.reuse);
        assert!(!reply.features.reverse);
        assert!(reply.features.nondeterministic);
        assert!(
            !Description::new("mine", "1.2.3")
                .reply()
                .features
                .nondeterministic
        );
        assert!(reply.numbering.is_unset());
    }

    #[test]
    fn directives_for_keeps_the_target_s() {
        let all = [
            Directive {
                name: "a".into(),
                target: "go".into(),
                ..Default::default()
            },
            Directive {
                name: "b".into(),
                target: "rust".into(),
                ..Default::default()
            },
        ];

        let names: Vec<_> = directives_for("rust", &all)
            .map(|d| d.name.as_str())
            .collect();

        assert_eq!(names, ["b"]);
    }
}
