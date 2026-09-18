package lsp

import (
	"context"
	"io"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"

	"github.com/unstoppablemango/tdl/internal/sema"
)

// Serve runs a server over one connection and returns when it ends.
//
// The connection is built here rather than with protocol.NewServer,
// because that constructor wants the server before it has made the client,
// and this server publishes diagnostics, so it needs the client to exist
// first.
//
// The logger is silent. Anything written to stdout on a stdio server
// corrupts the stream, and a library that logs by default is a library
// that will.
func Serve(ctx context.Context, rwc io.ReadWriteCloser, opts ...sema.Option) error {
	conn := jsonrpc2.NewConn(jsonrpc2.NewStream(rwc))
	client := protocol.ClientDispatcher(conn, zap.NewNop())

	ctx = protocol.WithClient(ctx, client)
	conn.Go(ctx, protocol.Handlers(
		protocol.ServerHandler(NewServer(client, opts...), jsonrpc2.MethodNotFoundHandler),
	))

	<-conn.Done()
	return conn.Err()
}
