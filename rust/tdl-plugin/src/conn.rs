use std::io::{self, Read, Write};

use buffa::Message;

use crate::Error;

/// The wire framing [`Conn`] implements.
pub const FRAMING_VERSION: i32 = 1;

/// The protobuf package the model is encoded with.
pub const IR_VERSION: &str = "tdl.ir.v1";

/// Bounds a single message. A length prefix over it is refused before
/// anything is allocated.
pub const MAX_MESSAGE_SIZE: u64 = 64 << 20;

// A varint holding a 64-bit length is at most ten bytes.
const MAX_VARINT_LENGTH: usize = 10;

/// One end of a plugin connection. Each message is framed as a varint byte
/// count followed by the encoded message.
pub struct Conn<R, W> {
    input: R,
    output: W,
}

impl<R: Read, W: Write> Conn<R, W> {
    /// Returns a connection reading from input and writing to output.
    pub fn new(input: R, output: W) -> Self {
        Conn { input, output }
    }

    /// Returns the two streams.
    pub fn into_inner(self) -> (R, W) {
        (self.input, self.output)
    }

    /// Encodes a message and writes it with its length prefix.
    pub fn send(&mut self, message: &impl Message) -> Result<(), Error> {
        let body = message.encode_to_vec();
        if body.len() as u64 > MAX_MESSAGE_SIZE {
            return Err(Error::protocol(format!(
                "message exceeds the maximum size: {} bytes",
                body.len()
            )));
        }

        let mut prefix = [0u8; MAX_VARINT_LENGTH];
        let mut n = 0;
        let mut size = body.len() as u64;
        loop {
            if size < 0x80 {
                prefix[n] = size as u8;
                n += 1;
                break;
            }
            prefix[n] = (size as u8) | 0x80;
            n += 1;
            size >>= 7;
        }

        self.output.write_all(&prefix[..n])?;
        self.output.write_all(&body)?;
        self.output.flush()?;
        Ok(())
    }

    /// Reads one message, or returns `None` when the stream ends between
    /// messages. A stream that ends part way through one is an
    /// [`io::ErrorKind::UnexpectedEof`] error.
    pub fn recv<M: Message>(&mut self) -> Result<Option<M>, Error> {
        let Some(size) = self.read_length()? else {
            return Ok(None);
        };
        if size > MAX_MESSAGE_SIZE {
            return Err(Error::protocol(format!(
                "message exceeds the maximum size: {size} bytes"
            )));
        }

        let mut body = vec![0u8; size as usize];
        self.input.read_exact(&mut body)?;
        Ok(Some(M::decode_from_slice(&body)?))
    }

    // Reads a varint, or returns None at the end of the stream before its
    // first byte.
    fn read_length(&mut self) -> Result<Option<u64>, Error> {
        let mut size = 0u64;
        for i in 0..MAX_VARINT_LENGTH {
            let mut b = [0u8; 1];
            if let Err(e) = self.input.read_exact(&mut b) {
                if e.kind() != io::ErrorKind::UnexpectedEof {
                    return Err(e.into());
                }
                if i == 0 {
                    return Ok(None);
                }
                return Err(io::Error::new(
                    io::ErrorKind::UnexpectedEof,
                    "the stream ended inside a length prefix",
                )
                .into());
            }
            let b = b[0];
            if i == MAX_VARINT_LENGTH - 1 && b > 1 {
                break;
            }
            size |= u64::from(b & 0x7f) << (7 * i);
            if b < 0x80 {
                return Ok(Some(size));
            }
        }
        Err(Error::protocol(
            "reading length: the varint overflows 64 bits",
        ))
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::protocol::{Handshake, Response};

    fn hello() -> Handshake {
        Handshake {
            framing_version: FRAMING_VERSION,
            ir_version: IR_VERSION.into(),
            ..Default::default()
        }
    }

    #[test]
    fn round_trips_messages() {
        let mut out = Vec::new();
        let mut conn = Conn::new(io::empty(), &mut out);
        conn.send(&hello()).unwrap();
        conn.send(&Response::default()).unwrap();

        let mut conn = Conn::new(out.as_slice(), io::sink());
        assert_eq!(conn.recv::<Handshake>().unwrap(), Some(hello()));
        assert_eq!(conn.recv::<Response>().unwrap(), Some(Response::default()));
        assert_eq!(conn.recv::<Response>().unwrap(), None);
    }

    #[test]
    fn prefixes_the_length_as_a_varint() {
        let big = Response {
            post: vec!["x".repeat(200)],
            ..Default::default()
        };
        let body = big.encode_to_vec();
        let mut out = Vec::new();
        Conn::new(io::empty(), &mut out).send(&big).unwrap();

        let len = body.len();
        assert_eq!(out[0], (len as u8) | 0x80);
        assert_eq!(out[1], (len >> 7) as u8);
        assert_eq!(&out[2..], body.as_slice());
    }

    #[test]
    fn a_stream_ending_inside_a_prefix_is_an_unexpected_eof() {
        let mut conn = Conn::new([0x80u8].as_slice(), io::sink());
        match conn.recv::<Response>() {
            Err(Error::Io(e)) => assert_eq!(e.kind(), io::ErrorKind::UnexpectedEof),
            other => panic!("got {other:?}"),
        }
    }

    #[test]
    fn a_stream_ending_inside_a_body_is_an_unexpected_eof() {
        let mut conn = Conn::new([5u8, 1].as_slice(), io::sink());
        match conn.recv::<Response>() {
            Err(Error::Io(e)) => assert_eq!(e.kind(), io::ErrorKind::UnexpectedEof),
            other => panic!("got {other:?}"),
        }
    }

    #[test]
    fn refuses_a_length_over_the_maximum() {
        // 64 MiB + 1 as a varint.
        let mut prefix = Vec::new();
        let mut n = MAX_MESSAGE_SIZE + 1;
        while n >= 0x80 {
            prefix.push((n as u8) | 0x80);
            n >>= 7;
        }
        prefix.push(n as u8);

        let mut conn = Conn::new(prefix.as_slice(), io::sink());
        match conn.recv::<Response>() {
            Err(Error::Protocol(m)) => assert!(m.contains("maximum size"), "{m}"),
            other => panic!("got {other:?}"),
        }
    }

    #[test]
    fn refuses_a_varint_over_64_bits() {
        let prefix = [0xffu8; 10];
        let mut conn = Conn::new(prefix.as_slice(), io::sink());
        match conn.recv::<Response>() {
            Err(Error::Protocol(m)) => assert!(m.contains("overflows"), "{m}"),
            other => panic!("got {other:?}"),
        }
    }
}
