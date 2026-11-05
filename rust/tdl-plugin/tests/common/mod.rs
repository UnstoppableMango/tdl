//! Finds the repository's testdata and docs above the crate, wherever the
//! build put it.

#![allow(dead_code)]

use std::path::PathBuf;

use tdl_plugin::buffa::Message;
use tdl_plugin::protocol::Request;

pub fn root() -> PathBuf {
    let mut dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    loop {
        if dir.join("testdata").join("plugin").is_dir() {
            return dir;
        }
        assert!(
            dir.pop(),
            "no testdata/plugin above {}",
            env!("CARGO_MANIFEST_DIR")
        );
    }
}

/// A request recorded in testdata/plugin.
pub fn request(exchange: &str) -> Request {
    let path = root()
        .join("testdata/plugin")
        .join(format!("{exchange}.request.binpb"));
    Request::decode_from_slice(&std::fs::read(path).unwrap()).unwrap()
}
