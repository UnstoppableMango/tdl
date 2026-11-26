package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// Serve runs backends over stdin and stdout, reading requests until the
// stream ends. A plugin's main is this and nothing else. A plugin serving
// several backends lists them all, and each connection is served by the one
// its handshake names (see [ServeConn]).
func Serve(bs ...Backend) error {
	return ServeConn(context.Background(), NewConn(os.Stdin, os.Stdout), bs...)
}

// ServeConn is [Serve] over a connection the caller supplies.
//
// The backend is chosen once, at the handshake: the one whose
// [Description.Name] is the handshake's target; with no target, from a
// host built before the field, the only backend, or the one named by the
// executable, tdl-gen-<name>. Any other target is refused, naming the
// targets served. The rest of the connection is that backend's alone.
func ServeConn(ctx context.Context, conn *Conn, bs ...Backend) error {
	if err := distinct(bs); err != nil {
		return err
	}

	var hello Handshake
	if err := conn.Recv(&hello); err != nil {
		return fmt.Errorf("plugin: reading the handshake: %w", err)
	}

	reason, ok := compatible(&hello)
	var b Backend
	if ok {
		b, reason, ok = choose(bs, hello.GetTarget(), os.Args[0])
	}
	var desc Description
	if ok {
		desc = b.Describe()
	}
	importer, imports := b.(Importer)
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

// distinct refuses a plugin serving no backend, or two under one name,
// since the handshake could never choose between them.
func distinct(bs []Backend) error {
	if len(bs) == 0 {
		return errors.New("plugin: no backend to serve")
	}
	seen := map[string]bool{}
	for _, b := range bs {
		name := b.Describe().Name
		if seen[name] {
			return fmt.Errorf("plugin: two backends are named %s", name)
		}
		seen[name] = true
	}
	return nil
}

// choose picks the backend a connection serves: the one named target, or
// with no target the only one, or the one the executable is named for. It
// says which targets it serves when none fits.
func choose(bs []Backend, target, executable string) (Backend, string, bool) {
	want := target
	if want == "" {
		if len(bs) == 1 {
			return bs[0], "", true
		}
		base := filepath.Base(executable)
		if runtime.GOOS == "windows" {
			base = strings.TrimSuffix(base, filepath.Ext(base))
		}
		want = strings.TrimPrefix(base, "tdl-gen-")
	}

	names := make([]string, 0, len(bs))
	for _, b := range bs {
		name := b.Describe().Name
		if name == want {
			return b, "", true
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return nil, fmt.Sprintf("serves %s; asked for %s", strings.Join(names, ", "), want), false
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
