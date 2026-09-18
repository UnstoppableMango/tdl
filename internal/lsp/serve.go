package lsp

import (
	"context"
	"io"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"

	"github.com/unstoppablemango/tdl/internal/sema"
)

// Serve runs a server over one connection and returns when it ends.
//
// The connection is built by protocol.NewServer rather than by hand,
// because the codec it installs is what decodes a union-typed parameter
// and is not exported to be installed separately. It puts the client in
// the context every request is answered under, which is where this server
// reads the one it publishes to.
//
// Nothing is logged. Anything written to stdout on a stdio server corrupts
// the stream, and a library that logs by default is a library that will.
func Serve(ctx context.Context, rwc io.ReadWriteCloser, opts ...sema.Option) error {
	_, conn, _ := protocol.NewServer(ctx, NewServer(opts...), jsonrpc2.NewStream(rwc))

	<-conn.Done()
	return conn.Err()
}
