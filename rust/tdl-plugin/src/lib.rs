//! The TDL plugin SDK: what a backend written in Rust builds on.
//!
//! It holds the wire framing, the handshake, a [`Session`] over a request
//! and the analysis tdl sends with it, positioned diagnostics with loss
//! codes, the cascade, and case styles. Like the .NET SDK, it computes no
//! fact the analysis carries, and refuses a request without one.
//!
//! A plugin's `main` is [`run`] over its [`Backend`]:
//!
//! ```no_run
//! # struct MyBackend;
//! # impl tdl_plugin::Backend for MyBackend {
//! #     fn describe(&self) -> tdl_plugin::Description { tdl_plugin::Description::new("mine", "0.1.0") }
//! #     fn generate(&self, _: &tdl_plugin::protocol::Request) -> Result<tdl_plugin::protocol::Response, tdl_plugin::Error> { todo!() }
//! # }
//! fn main() -> std::process::ExitCode {
//!     tdl_plugin::run(&MyBackend)
//! }
//! ```

mod backend;
mod cascade;
mod case;
mod conn;
mod error;
mod problem;
mod server;
mod session;

pub use backend::{Backend, Description, ON_MEMBERS, directives_for};
pub use cascade::Skipped;
pub use case::{Style, camel, from_screaming, last_segment, pascal, screaming_snake, snake, words};
pub use conn::{Conn, FRAMING_VERSION, IR_VERSION, MAX_MESSAGE_SIZE};
pub use error::Error;
pub use problem::{Problem, loss};
pub use server::{run, serve};
pub use session::{Session, declares, deprecated, doc, fields, params, variants};

/// The protobuf types, generated from `proto/` by buffa (`make generate`).
/// [`ir`] and [`protocol`] name the two packages.
#[allow(clippy::all, missing_docs, rustdoc::all)]
pub mod tdl {
    /// The `tdl.ir` package.
    pub mod ir {
        /// `tdl.ir.v1`: the resolved model.
        pub mod v1 {
            include!("generated/tdl.ir.v1.rs");
        }
    }

    /// The `tdl.plugin` package.
    pub mod plugin {
        /// `tdl.plugin.v1`: the plugin protocol.
        pub mod v1 {
            include!("generated/tdl.plugin.v1.rs");
        }
    }
}

pub use tdl::ir::v1 as ir;
pub use tdl::plugin::v1 as protocol;

/// The buffa runtime the generated types are built on, for its `Message`
/// trait and field wrappers.
pub use buffa;
