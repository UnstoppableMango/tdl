//! The debug port answers each request the Go host recorded in
//! testdata/plugin with the bytes the Go debug backend answered it with.

use std::io;
use std::path::PathBuf;

use tdl_plugin::buffa::Message;
use tdl_plugin::protocol::{Handshake, Request, Response};
use tdl_plugin::{Backend, Conn, FRAMING_VERSION, IR_VERSION, serve};
use tdl_plugin_debug::Debug;

fn plugin_data() -> PathBuf {
    let mut dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    loop {
        let candidate = dir.join("testdata").join("plugin");
        if candidate.is_dir() {
            return candidate;
        }
        assert!(
            dir.pop(),
            "no testdata/plugin above {}",
            env!("CARGO_MANIFEST_DIR")
        );
    }
}

// The base names of the recorded exchanges.
fn exchanges() -> Vec<String> {
    let mut names: Vec<String> = std::fs::read_dir(plugin_data())
        .unwrap()
        .filter_map(|e| {
            let name = e.unwrap().file_name().into_string().unwrap();
            name.strip_suffix(".request.binpb").map(str::to_string)
        })
        .collect();
    names.sort();
    names
}

fn read(name: &str) -> Vec<u8> {
    std::fs::read(plugin_data().join(name)).unwrap()
}

fn request(exchange: &str) -> Request {
    Request::decode_from_slice(&read(&format!("{exchange}.request.binpb"))).unwrap()
}

fn response(exchange: &str) -> Vec<u8> {
    read(&format!("{exchange}.response.binpb"))
}

// A writer of one message to a connection.
type Sender<'a> = &'a dyn Fn(&mut Conn<io::Empty, &mut Vec<u8>>);

fn framed(messages: &[Sender]) -> Vec<u8> {
    let mut out = Vec::new();
    let mut conn = Conn::new(io::empty(), &mut out);
    for m in messages {
        m(&mut conn);
    }
    out
}

#[test]
fn there_are_exchanges() {
    assert!(!exchanges().is_empty());
}

#[test]
fn answers_byte_for_byte() {
    for exchange in exchanges() {
        let resp = Debug.generate(&request(&exchange)).unwrap();

        assert!(resp.encode_to_vec() == response(&exchange), "{exchange}");
    }
}

#[test]
fn answers_over_a_connection() {
    for exchange in exchanges() {
        let hello = Handshake {
            framing_version: FRAMING_VERSION,
            ir_version: IR_VERSION.into(),
            ..Default::default()
        };
        let req = request(&exchange);
        let input = framed(&[&|c| c.send(&hello).unwrap(), &|c| c.send(&req).unwrap()]);
        let mut output = Vec::new();

        serve(&Debug, Conn::new(input.as_slice(), &mut output)).unwrap();

        let want_resp = Response::decode_from_slice(&response(&exchange)).unwrap();
        let reply = Debug.describe().reply();
        let want = framed(&[&|c| c.send(&reply).unwrap(), &|c| {
            c.send(&want_resp).unwrap()
        }]);
        assert!(output == want, "{exchange}");
    }
}
