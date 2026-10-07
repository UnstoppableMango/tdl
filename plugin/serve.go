package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

// Serve runs a backend over stdin and stdout, reading requests until the
// stream ends. A plugin's main is this and nothing else.
func Serve(b Backend) error {
	return ServeConn(context.Background(), b, NewConn(os.Stdin, os.Stdout))
}

// ServeConn is [Serve] over a connection the caller supplies.
func ServeConn(ctx context.Context, b Backend, conn *Conn) error {
	var hello Handshake
	if err := conn.Recv(&hello); err != nil {
		return fmt.Errorf("plugin: reading the handshake: %w", err)
	}

	desc := b.Describe()
	importer, imports := b.(Importer)
	reason, ok := compatible(&hello)
	if ok && hello.GetMode() == Mode_MODE_IMPORT && (!imports || !desc.Reverse) {
		reason, ok = desc.Name+" does not import; it generates only", false
	}
	if !ok {
		if err := conn.Send(&HandshakeReply{Accepted: false, Refusal: reason}); err != nil {
			return fmt.Errorf("plugin: refusing: %w", err)
		}
		return errors.New("plugin: " + reason)
	}

	if err := conn.Send(desc.Reply()); err != nil {
		return fmt.Errorf("plugin: accepting: %w", err)
	}

	if hello.GetMode() == Mode_MODE_IMPORT {
		return serveImports(ctx, importer, conn)
	}

	// Several requests arrive only in a watch session with a backend that
	// declared reuse.
	for {
		var req Request
		switch err := conn.Recv(&req); {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return fmt.Errorf("plugin: reading a request: %w", err)
		}

		resp, err := b.Generate(ctx, &req)
		if err != nil {
			return fmt.Errorf("plugin: generating: %w", err)
		}
		if err := conn.Send(resp); err != nil {
			return fmt.Errorf("plugin: answering: %w", err)
		}
	}
}

// serveImports answers import requests until the stream ends.
func serveImports(ctx context.Context, b Importer, conn *Conn) error {
	for {
		var req ImportRequest
		switch err := conn.Recv(&req); {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return fmt.Errorf("plugin: reading an import request: %w", err)
		}

		resp, err := b.Import(ctx, &req)
		if err != nil {
			return fmt.Errorf("plugin: importing: %w", err)
		}
		if err := conn.Send(resp); err != nil {
			return fmt.Errorf("plugin: answering: %w", err)
		}
	}
}

// compatible reports whether a plugin can read what the host is about to
// send, and says what it needed when it cannot.
func compatible(h *Handshake) (string, bool) {
	if v := h.GetFramingVersion(); v != FramingVersion {
		return fmt.Sprintf("framing version %d is not supported; this plugin speaks %d", v, FramingVersion), false
	}
	if v := h.GetIrVersion(); v != IRVersion {
		return fmt.Sprintf("model version %q is not supported; this plugin reads %q", v, IRVersion), false
	}
	return "", true
}
