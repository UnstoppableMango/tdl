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
// protocol.NewServer builds the connection because only its unexported
// codec decodes union-typed parameters. Nothing is logged, since stdout is
// the stream on a stdio server.
func Serve(ctx context.Context, rwc io.ReadWriteCloser, opts ...sema.Option) error {
	_, conn, _ := protocol.NewServer(ctx, NewServer(opts...), jsonrpc2.NewStream(rwc))

	<-conn.Done()
	return conn.Err()
}
