package plugin_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/plugin"
)

func TestRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	conn := plugin.NewConn(&buf, &buf)

	sent := &plugin.Handshake{
		FramingVersion: plugin.FramingVersion,
		IrVersion:      plugin.IRVersion,
		Watch:          true,
	}
	if err := conn.Send(sent); err != nil {
		t.Fatalf("send: %v", err)
	}

	var got plugin.Handshake
	if err := conn.Recv(&got); err != nil {
		t.Fatalf("recv: %v", err)
	}
	if !proto.Equal(sent, &got) {
		t.Errorf("got %+v, want %+v", &got, sent)
	}
}

// A connection carries several messages, which is what lets the handshake
// and the request share one stream.
func TestSeveralMessages(t *testing.T) {
	var buf bytes.Buffer
	conn := plugin.NewConn(&buf, &buf)

	for _, name := range []string{"first", "second", "third"} {
		if err := conn.Send(&plugin.HandshakeReply{Name: name}); err != nil {
			t.Fatalf("send %s: %v", name, err)
		}
	}
	for _, want := range []string{"first", "second", "third"} {
		var got plugin.HandshakeReply
		if err := conn.Recv(&got); err != nil {
			t.Fatalf("recv %s: %v", want, err)
		}
		if got.GetName() != want {
			t.Errorf("got %q, want %q", got.GetName(), want)
		}
	}
}

// A stream that ends between messages is an ending, not a failure.
func TestEndOfStream(t *testing.T) {
	conn := plugin.NewConn(strings.NewReader(""), io.Discard)
	if err := conn.Recv(&plugin.Handshake{}); !errors.Is(err, io.EOF) {
		t.Errorf("err = %v, want io.EOF", err)
	}
}

// A stream that ends part way through a message is a truncation, and the
// error says so rather than the read blocking forever.
func TestTruncatedMessage(t *testing.T) {
	var full bytes.Buffer
	if err := plugin.NewConn(nil, &full).Send(&plugin.HandshakeReply{Name: "cut short"}); err != nil {
		t.Fatalf("send: %v", err)
	}

	whole := full.Bytes()
	for _, cut := range []int{1, len(whole) / 2, len(whole) - 1} {
		conn := plugin.NewConn(bytes.NewReader(whole[:cut]), io.Discard)
		err := conn.Recv(&plugin.HandshakeReply{})
		if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			t.Errorf("cut at %d: err = %v, want an unexpected EOF", cut, err)
		}
		if cut > 1 && errors.Is(err, io.EOF) {
			t.Errorf("cut at %d: a truncated message reported a clean ending", cut)
		}
	}
}

// A length prefix is the first thing read from a stream that may be
// anything at all, so it is checked before it is trusted.
func TestOversizedPrefixIsRefused(t *testing.T) {
	var prefix [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(prefix[:], uint64(plugin.MaxMessageSize)+1)

	conn := plugin.NewConn(bytes.NewReader(prefix[:n]), io.Discard)
	if err := conn.Recv(&plugin.Handshake{}); !errors.Is(err, plugin.ErrTooLarge) {
		t.Errorf("err = %v, want ErrTooLarge", err)
	}
}

func TestOversizedSendIsRefused(t *testing.T) {
	big := &plugin.File{Content: make([]byte, plugin.MaxMessageSize+1)}
	if err := plugin.NewConn(nil, io.Discard).Send(big); !errors.Is(err, plugin.ErrTooLarge) {
		t.Errorf("err = %v, want ErrTooLarge", err)
	}
}

func TestGarbageBody(t *testing.T) {
	var buf bytes.Buffer
	var prefix [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(prefix[:], 4)
	buf.Write(prefix[:n])
	buf.Write([]byte{0xff, 0xff, 0xff, 0xff})

	conn := plugin.NewConn(&buf, io.Discard)
	if err := conn.Recv(&plugin.Handshake{}); err == nil {
		t.Error("a body that is not a message decoded without complaint")
	}
}
