//! A port of the Go debug backend, which describes the model it was given.
//! It exercises the SDK and the protocol without generating code, and
//! answers a request with the bytes the Go backend answers it with.

use std::fmt::Write as _;

use tdl_plugin::ir::LiteralKind;
use tdl_plugin::protocol::{DirectiveSpec, File, Request, Response};
use tdl_plugin::{Backend, Description, Error, Problem, Session, directives_for, fields};

/// What this backend is called, in a target block and as
/// `tdl-gen-rust_debug` on PATH. The Go backend holds the name `debug`, and
/// tdl never looks a built-in name up on PATH.
pub const NAME: &str = "rust_debug";

/// The debug backend.
pub struct Debug;

impl Backend for Debug {
    fn describe(&self) -> Description {
        Description {
            // Requests share no state.
            reuse: true,
            directives: vec![
                // note("...") exists to exercise directive plumbing.
                DirectiveSpec {
                    name: "note".into(),
                    min_args: 1,
                    max_args: 1,
                    arg_kinds: vec![LiteralKind::LITERAL_KIND_STRING.into()],
                    ..Default::default()
                },
            ],
            ..Description::new(NAME, "0.1.0")
        }
    }

    /// Writes one file describing the model.
    fn generate(&self, request: &Request) -> Result<Response, Error> {
        let mut s = Session::new(request, "debug")?;
        let model = s.model();

        let mut b = String::new();
        let own = s.own();
        // Writing to a String cannot fail.
        let _ = write!(
            b,
            "package {}\ntarget {}\n\n\
             declarations: {} own, {} from the prelude\n\
             types: {}\nexterns: {}\ninstances: {}\n",
            model.package,
            request.target,
            own.len(),
            model.decls.len() - own.len(),
            model.types.len(),
            model.externs.len(),
            model.instances.len(),
        );

        for d in &own {
            let _ = write!(b, "\n{}\n", d.meta.name);
            for dir in directives_for(&request.target, &d.directives) {
                let _ = writeln!(b, "  directive {}", dir.name);
            }
            for f in fields(d) {
                let _ = writeln!(b, "  field {}: {}", f.meta.name, f.r#type.name);
            }
        }

        // Each fieldless declaration warns, to exercise the diagnostic path.
        for d in &own {
            if !fields(d).is_empty()
                || matches!(d.node, Some(tdl_plugin::ir::decl::Node::Enumeration(_)))
            {
                continue;
            }
            s.warn(Problem::unsupported(
                format!(
                    "{} has no fields, so there is nothing to generate for it",
                    d.meta.name
                ),
                d.meta.position.as_option(),
            ));
        }

        Ok(s.response(vec![File {
            path: "model.txt".into(),
            content: b.into_bytes(),
            ..Default::default()
        }]))
    }
}
