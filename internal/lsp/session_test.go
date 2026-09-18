package lsp_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"

	"github.com/unstoppablemango/tdl/internal/lsp"
)

// publishWait is how long a test waits for a notification.
//
// A notification has no reply to synchronize on, so a test that wants to
// see what a didOpen produced has to wait for it. The value is generous
// because it is a failure timeout rather than a delay: a passing test
// never spends it.
const publishWait = 10 * time.Second

// session is a client and a server talking over a pipe.
//
// The test speaks real JSON-RPC to the real handler rather than calling
// its methods, which is what internal/gen/subprocess_test.go does for the
// plugin protocol: a surface only the in-process caller reaches is one
// nothing keeps honest.
type session struct {
	t      *testing.T
	server protocol.Server
	client *testClient
}

func newSession(t *testing.T) *session {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = lsp.Serve(ctx, serverConn)
	}()

	client := &testClient{published: map[string][]protocol.Diagnostic{}}
	_, conn, server := protocol.NewClient(ctx, client, jsonrpc2.NewStream(clientConn), zap.NewNop())

	t.Cleanup(func() {
		_ = conn.Close()
		cancel()
		<-done
	})

	if _, err := server.Initialize(ctx, &protocol.InitializeParams{}); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if err := server.Initialized(ctx, &protocol.InitializedParams{}); err != nil {
		t.Fatalf("initialized: %v", err)
	}

	return &session{t: t, server: server, client: client}
}

// open sends a didOpen and returns the diagnostics published for that file.
func (s *session) open(path, text string) []protocol.Diagnostic {
	s.t.Helper()

	u := uri.File(path)
	s.client.expect(u)
	err := s.server.DidOpen(context.Background(), &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        u,
			LanguageID: "tdl",
			Version:    1,
			Text:       text,
		},
	})
	if err != nil {
		s.t.Fatalf("didOpen: %v", err)
	}
	return s.client.wait(s.t, u)
}

// change sends a didChange with the whole new text and returns the
// diagnostics published for that file.
func (s *session) change(path, text string, version int32) []protocol.Diagnostic {
	s.t.Helper()

	u := uri.File(path)
	s.client.expect(u)
	err := s.server.DidChange(context.Background(), &protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: u},
			Version:                version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{{Text: text}},
	})
	if err != nil {
		s.t.Fatalf("didChange: %v", err)
	}
	return s.client.wait(s.t, u)
}

// definition sends a textDocument/definition with the cursor on the first
// occurrence of needle in text.
//
// The cursor is computed from the text the test wrote rather than passed
// as a line and a column, so a case says which name it is asking about and
// not where that name happens to sit.
func (s *session) definition(path, text, needle string) []protocol.Location {
	s.t.Helper()

	locs, err := s.server.Definition(context.Background(), &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri.File(path)},
			Position:     cursor(s.t, text, needle),
		},
	})
	if err != nil {
		s.t.Fatalf("definition: %v", err)
	}
	return locs
}

// cursor is the protocol position of the first occurrence of needle.
//
// Every caller writes ASCII, where a byte and a UTF-16 code unit are the
// same thing; TestDiagnosticsUseUTF16Columns is what covers the conversion
// itself.
func cursor(t *testing.T, text, needle string) protocol.Position {
	t.Helper()

	off := strings.Index(text, needle)
	if off < 0 {
		t.Fatalf("%q is not in the source", needle)
	}

	line := strings.Count(text[:off], "\n")
	col := off - (strings.LastIndex(text[:off], "\n") + 1)
	return protocol.Position{Line: uint32(line), Character: uint32(col)}
}

// testClient records what the server publishes.
//
// Only PublishDiagnostics is interesting; the rest of the interface exists
// because protocol.Client declares it.
type testClient struct {
	mu        sync.Mutex
	published map[string][]protocol.Diagnostic
	waiting   map[string]chan []protocol.Diagnostic
}

// expect arms a wait for the next publish against u, before the request
// that causes it is sent, so a publish that arrives immediately is not
// missed.
func (c *testClient) expect(u uri.URI) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.waiting == nil {
		c.waiting = map[string]chan []protocol.Diagnostic{}
	}
	c.waiting[string(u)] = make(chan []protocol.Diagnostic, 1)
}

// wait returns the diagnostics published against u.
func (c *testClient) wait(t *testing.T, u uri.URI) []protocol.Diagnostic {
	t.Helper()

	c.mu.Lock()
	ch := c.waiting[string(u)]
	c.mu.Unlock()

	select {
	case diags := <-ch:
		return diags
	case <-time.After(publishWait):
		t.Fatalf("timed out waiting for diagnostics on %s", u)
		return nil
	}
}

func (c *testClient) PublishDiagnostics(_ context.Context, params *protocol.PublishDiagnosticsParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := string(params.URI)
	c.published[key] = params.Diagnostics
	if ch, ok := c.waiting[key]; ok {
		select {
		case ch <- params.Diagnostics:
		default:
		}
	}
	return nil
}

func (c *testClient) Progress(context.Context, *protocol.ProgressParams) error { return nil }

func (c *testClient) WorkDoneProgressCreate(context.Context, *protocol.WorkDoneProgressCreateParams) error {
	return nil
}

func (c *testClient) LogMessage(context.Context, *protocol.LogMessageParams) error { return nil }

func (c *testClient) ShowMessage(context.Context, *protocol.ShowMessageParams) error { return nil }

func (c *testClient) ShowMessageRequest(context.Context, *protocol.ShowMessageRequestParams) (*protocol.MessageActionItem, error) {
	return nil, nil
}

func (c *testClient) Telemetry(context.Context, interface{}) error { return nil }

func (c *testClient) RegisterCapability(context.Context, *protocol.RegistrationParams) error {
	return nil
}

func (c *testClient) UnregisterCapability(context.Context, *protocol.UnregistrationParams) error {
	return nil
}

func (c *testClient) ApplyEdit(context.Context, *protocol.ApplyWorkspaceEditParams) (bool, error) {
	return false, nil
}

func (c *testClient) Configuration(context.Context, *protocol.ConfigurationParams) ([]interface{}, error) {
	return nil, nil
}

func (c *testClient) WorkspaceFolders(context.Context) ([]protocol.WorkspaceFolder, error) {
	return nil, nil
}

var _ protocol.Client = (*testClient)(nil)

// corpusFile reads a case's source.tdl, skipping a pending one the way the
// parser's conformance test does.
func corpusFile(t *testing.T, dir string) (string, string) {
	t.Helper()

	if reason, err := os.ReadFile(filepath.Join(dir, "pending")); err == nil {
		t.Skip(strings.TrimSpace(string(reason)))
	}

	path := filepath.Join(dir, "source.tdl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading source.tdl: %v", err)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolving %s: %v", path, err)
	}
	return abs, string(data)
}

// subdirs is every case directory under root.
func subdirs(t *testing.T, root string) []string {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs
}
