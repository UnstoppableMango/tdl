use crate::ir::Position;

/// Something a backend reports about the model, at a position.
#[derive(Clone, Debug, PartialEq)]
pub enum Problem {
    /// A shape the backend cannot express. It reaches the user as a
    /// warning, carrying a code from [`loss`] when its omission is one the
    /// user may choose to allow, or an empty code otherwise.
    Unsupported {
        /// What the output lacks.
        message: String,
        /// Where the shape is written.
        position: Option<Position>,
        /// A loss code, or empty.
        code: &'static str,
    },
    /// A directive the user got wrong, such as two fields pinned to one
    /// number. It reaches the user as an error, so the host writes nothing.
    Invalid {
        /// What is wrong.
        message: String,
        /// Where it is written.
        position: Option<Position>,
    },
}

impl Problem {
    /// An [`Unsupported`](Problem::Unsupported) with no loss code.
    pub fn unsupported(message: impl Into<String>, position: Option<&Position>) -> Self {
        Problem::Unsupported {
            message: message.into(),
            position: position.cloned(),
            code: "",
        }
    }

    /// An [`Unsupported`](Problem::Unsupported) carrying a loss code.
    pub fn lossy(
        code: &'static str,
        message: impl Into<String>,
        position: Option<&Position>,
    ) -> Self {
        Problem::Unsupported {
            message: message.into(),
            position: position.cloned(),
            code,
        }
    }

    /// An [`Invalid`](Problem::Invalid).
    pub fn invalid(message: impl Into<String>, position: Option<&Position>) -> Self {
        Problem::Invalid {
            message: message.into(),
            position: position.cloned(),
        }
    }
}

/// Loss codes name a fact a conversion cannot carry, in either direction. A
/// warning carrying one can be silenced in `tdl.toml` or with
/// `--allow-lossy`. `docs/architecture/reverse.md` describes each.
pub mod loss {
    #![allow(missing_docs)] // Each is described in reverse.md.

    pub const PRIMITIVE: &str = "lossy.primitive";
    pub const NEWTYPE: &str = "lossy.newtype";
    pub const COLLECTION: &str = "lossy.collection";
    pub const OPTIONAL: &str = "lossy.optional";
    pub const STRUCT_KIND: &str = "lossy.struct-kind";
    pub const INCLUDE: &str = "lossy.include";
    pub const ALIAS: &str = "lossy.alias";
    pub const CONSTRAINT: &str = "lossy.constraint";
    pub const OWNED: &str = "lossy.owned";
    pub const DEFAULT: &str = "lossy.default";
    pub const KEY: &str = "lossy.key";
    pub const NAME: &str = "lossy.name";
    pub const NUMBER: &str = "lossy.number";
    pub const ORDER: &str = "lossy.order";
    pub const DOC: &str = "lossy.doc";
    pub const GENERIC: &str = "lossy.generic";
    pub const CLASS: &str = "lossy.class";
    pub const UNIT: &str = "lossy.unit";
    pub const UNSUPPORTED: &str = "lossy.unsupported";

    /// Every loss code, in the order the design lists them.
    pub const ALL: [&str; 19] = [
        PRIMITIVE,
        NEWTYPE,
        COLLECTION,
        OPTIONAL,
        STRUCT_KIND,
        INCLUDE,
        ALIAS,
        CONSTRAINT,
        OWNED,
        DEFAULT,
        KEY,
        NAME,
        NUMBER,
        ORDER,
        DOC,
        GENERIC,
        CLASS,
        UNIT,
        UNSUPPORTED,
    ];
}
