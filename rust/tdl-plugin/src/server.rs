use std::io::{self, Read, Write};
use std::process::ExitCode;

use crate::conn::{FRAMING_VERSION, IR_VERSION};
use crate::protocol::{Handshake, HandshakeReply, ImportRequest, Mode, Request};
use crate::{Backend, Conn, Error};

/// Runs a backend over standard input and output, reading requests until
/// the stream ends, and returns the process's exit status. A plugin's
/// `main` is this and nothing else.
pub fn run(backend: &impl Backend) -> ExitCode {
    let conn = Conn::new(io::stdin().lock(), io::stdout().lock());
    match serve(backend, conn) {
        Ok(()) => ExitCode::SUCCESS,
        Err(e) => {
            // A backend that fails cannot answer; tdl reports what it wrote.
            eprintln!("plugin: {e}");
            ExitCode::FAILURE
        }
    }
}

/// Serves one connection: the handshake, then requests until the stream
/// ends. A refused handshake is answered and then returned as an error.
pub fn serve<R: Read, W: Write>(backend: &impl Backend, mut conn: Conn<R, W>) -> Result<(), Error> {
    let hello: Handshake = conn.recv()?.ok_or_else(|| {
        Error::Io(io::Error::new(
            io::ErrorKind::UnexpectedEof,
            "the stream ended before the handshake",
        ))
    })?;

    let desc = backend.describe();
    let import = hello.mode == Mode::MODE_IMPORT;
    let mut refusal = incompatible(&hello);
    if refusal.is_none() && import && !desc.reverse {
        refusal = Some(format!("{} does not import; it generates only", desc.name));
    }
    if let Some(refusal) = refusal {
        conn.send(&HandshakeReply {
            accepted: false,
            refusal: refusal.clone(),
            ..Default::default()
        })?;
        return Err(Error::Protocol(refusal));
    }

    conn.send(&desc.reply())?;

    if import {
        while let Some(req) = conn.recv::<ImportRequest>()? {
            conn.send(&backend.import(&req)?)?;
        }
        return Ok(());
    }

    // Several requests arrive only in a watch session with a backend that
    // declared reuse.
    while let Some(req) = conn.recv::<Request>()? {
        conn.send(&backend.generate(&req)?)?;
    }
    Ok(())
}

// Says what a plugin needed when it cannot read what the host is about to
// send, or returns None when it can.
fn incompatible(h: &Handshake) -> Option<String> {
    if h.framing_version != FRAMING_VERSION {
        return Some(format!(
            "framing version {} is not supported; this plugin speaks {FRAMING_VERSION}",
            h.framing_version
        ));
    }
    if h.ir_version != IR_VERSION {
        return Some(format!(
            "model version \"{}\" is not supported; this plugin reads \"{IR_VERSION}\"",
            h.ir_version
        ));
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::Description;
    use crate::protocol::{File, Response};

    struct Echo {
        reverse: bool,
    }

    impl Backend for Echo {
        fn describe(&self) -> Description {
            Description {
                reverse: self.reverse,
                ..Description::new("echo", "0.0.1")
            }
        }

        fn generate(&self, request: &Request) -> Result<Response, Error> {
            Ok(Response {
                files: vec![File {
                    path: request.target.clone(),
                    ..Default::default()
                }],
                ..Default::default()
            })
        }
    }

    fn hello() -> Handshake {
        Handshake {
            framing_version: FRAMING_VERSION,
            ir_version: IR_VERSION.into(),
            ..Default::default()
        }
    }

    fn framed(write: impl FnOnce(&mut Conn<io::Empty, &mut Vec<u8>>)) -> Vec<u8> {
        let mut out = Vec::new();
        write(&mut Conn::new(io::empty(), &mut out));
        out
    }

    fn exchange(backend: &Echo, input: &[u8]) -> (Result<(), Error>, Vec<u8>) {
        let mut out = Vec::new();
        let result = serve(backend, Conn::new(input, &mut out));
        (result, out)
    }

    fn reply(out: &[u8]) -> HandshakeReply {
        Conn::new(out, io::sink()).recv().unwrap().unwrap()
    }

    #[test]
    fn answers_each_request_after_the_handshake() {
        let input = framed(|c| {
            c.send(&hello()).unwrap();
            for target in ["a", "b"] {
                c.send(&Request {
                    target: target.into(),
                    ..Default::default()
                })
                .unwrap();
            }
        });

        let (result, out) = exchange(&Echo { reverse: false }, &input);

        result.unwrap();
        let mut conn = Conn::new(out.as_slice(), io::sink());
        let got: HandshakeReply = conn.recv().unwrap().unwrap();
        assert_eq!(got, Echo { reverse: false }.describe().reply());
        for target in ["a", "b"] {
            let resp: Response = conn.recv().unwrap().unwrap();
            assert_eq!(resp.files[0].path, target);
        }
        assert_eq!(conn.recv::<Response>().unwrap(), None);
    }

    #[test]
    fn refuses_another_framing_version() {
        let input = framed(|c| {
            c.send(&Handshake {
                framing_version: 2,
                ..hello()
            })
            .unwrap()
        });

        let (result, out) = exchange(&Echo { reverse: false }, &input);

        let got = reply(&out);
        assert!(!got.accepted);
        assert!(got.refusal.contains("framing version 2"), "{}", got.refusal);
        assert!(matches!(result, Err(Error::Protocol(m)) if m == got.refusal));
    }

    #[test]
    fn refuses_another_model_version() {
        let input = framed(|c| {
            c.send(&Handshake {
                ir_version: "tdl.ir.v2".into(),
                ..hello()
            })
            .unwrap()
        });

        let (result, out) = exchange(&Echo { reverse: false }, &input);

        assert!(reply(&out).refusal.contains("\"tdl.ir.v2\""));
        assert!(result.is_err());
    }

    #[test]
    fn refuses_import_to_a_backend_that_only_generates() {
        let input = framed(|c| {
            c.send(&Handshake {
                mode: Mode::MODE_IMPORT.into(),
                ..hello()
            })
            .unwrap()
        });

        let (result, out) = exchange(&Echo { reverse: false }, &input);

        assert_eq!(
            reply(&out).refusal,
            "echo does not import; it generates only"
        );
        assert!(result.is_err());
    }

    #[test]
    fn an_empty_stream_is_an_unexpected_eof() {
        let (result, out) = exchange(&Echo { reverse: false }, &[]);

        assert!(out.is_empty());
        assert!(matches!(result, Err(Error::Io(e)) if e.kind() == io::ErrorKind::UnexpectedEof));
    }
}
