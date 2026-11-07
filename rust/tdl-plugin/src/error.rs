use std::fmt;
use std::io;

/// A failure of the protocol itself, or of a request the SDK cannot serve,
/// rather than a problem with the model, which belongs in a response's
/// diagnostics.
#[derive(Debug)]
pub enum Error {
    /// Reading or writing the connection failed. A stream that ends part
    /// way through a message is [`io::ErrorKind::UnexpectedEof`].
    Io(io::Error),
    /// A message could not be decoded.
    Decode(buffa::DecodeError),
    /// The other end broke the protocol, or a request cannot be served.
    Protocol(String),
}

impl Error {
    pub(crate) fn protocol(message: impl Into<String>) -> Self {
        Error::Protocol(message.into())
    }
}

impl fmt::Display for Error {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Error::Io(e) => e.fmt(f),
            Error::Decode(e) => write!(f, "unmarshalling: {e}"),
            Error::Protocol(m) => f.write_str(m),
        }
    }
}

impl std::error::Error for Error {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Error::Io(e) => Some(e),
            Error::Decode(e) => Some(e),
            Error::Protocol(_) => None,
        }
    }
}

impl From<io::Error> for Error {
    fn from(e: io::Error) -> Self {
        Error::Io(e)
    }
}

impl From<buffa::DecodeError> for Error {
    fn from(e: buffa::DecodeError) -> Self {
        Error::Decode(e)
    }
}
