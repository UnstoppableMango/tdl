// Package plugin is the wire protocol a TDL backend speaks, and the SDK
// for writing one.
//
// The messages are generated from proto/tdl/plugin/v1/plugin.proto. A
// backend author imports this package; it is a compatibility surface like
// ir, and internal/gen is the compiler side that changes freely.
//
// See docs/design/plugins.md.
package plugin

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"
)

// FramingVersion is the wire framing this package implements: a varint
// byte count followed by the encoded message, in both directions, over one
// connection that may carry several messages.
//
// It is not protoc's model of one request in, one response out and a
// closed stdin. A stream lets the handshake and the request be ordinary
// messages, lets a plugin be reused across regenerations, and leaves room
// to offer something else later without a second mechanism.
const FramingVersion = 1

// IRVersion is the protobuf package the model is encoded with.
const IRVersion = "tdl.ir.v1"

// MaxMessageSize bounds a single message.
//
// A length prefix is the first thing read from a stream that may be
// anything at all, so it is checked before it is trusted: without a bound,
// a corrupt or hostile prefix asks for an allocation of whatever it says.
const MaxMessageSize = 64 << 20

// ErrTooLarge is returned when a length prefix exceeds [MaxMessageSize].
var ErrTooLarge = errors.New("plugin: message exceeds the maximum size")

// Conn is one end of a plugin connection.
//
// It is not safe for concurrent use. The protocol is a sequence of
// exchanges on one connection, so there is nothing to interleave.
type Conn struct {
	r *bufio.Reader
	w io.Writer
}

// NewConn returns a Conn reading from r and writing to w.
func NewConn(r io.Reader, w io.Writer) *Conn {
	return &Conn{r: bufio.NewReader(r), w: w}
}

// Send encodes m and writes it with its length prefix.
func (c *Conn) Send(m proto.Message) error {
	body, err := proto.Marshal(m)
	if err != nil {
		return fmt.Errorf("plugin: marshalling: %w", err)
	}
	if len(body) > MaxMessageSize {
		return fmt.Errorf("%w: %d bytes", ErrTooLarge, len(body))
	}

	var prefix [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(prefix[:], uint64(len(body)))

	if _, err := c.w.Write(prefix[:n]); err != nil {
		return fmt.Errorf("plugin: writing length: %w", err)
	}
	if _, err := c.w.Write(body); err != nil {
		return fmt.Errorf("plugin: writing message: %w", err)
	}
	return nil
}

// Recv reads one message into m.
//
// A stream that ends between messages returns [io.EOF], which is how a
// reader knows the other end is done. A stream that ends part way through
// one returns [io.ErrUnexpectedEOF], because that is a truncated message
// rather than an ending.
func (c *Conn) Recv(m proto.Message) error {
	size, err := binary.ReadUvarint(c.r)
	switch {
	case errors.Is(err, io.EOF):
		return io.EOF
	case err != nil:
		return fmt.Errorf("plugin: reading length: %w", err)
	case size > MaxMessageSize:
		return fmt.Errorf("%w: %d bytes", ErrTooLarge, size)
	}

	body := make([]byte, size)
	if _, err := io.ReadFull(c.r, body); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return fmt.Errorf("plugin: reading message: %w", err)
	}

	if err := proto.Unmarshal(body, m); err != nil {
		return fmt.Errorf("plugin: unmarshalling: %w", err)
	}
	return nil
}
