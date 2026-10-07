package plugin_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// reader is an importer that names its model after the first file.
type reader struct{ shout }

func (reader) Describe() plugin.Description {
	return plugin.Description{Name: "reader", Reverse: true}
}

func (reader) Import(_ context.Context, req *plugin.ImportRequest) (*plugin.ImportResponse, error) {
	return &plugin.ImportResponse{
		Model: &ir.Model{Package: req.GetFiles()[0].GetPath()},
		Diagnostics: []*plugin.Diagnostic{{
			Severity: plugin.Severity_SEVERITY_WARNING,
			Message:  "allowed: " + strings.Join(req.GetAllowLossy(), ","),
			Code:     "lossy.order",
		}},
	}, nil
}

// serve runs ServeConn over the messages given and returns what the
// plugin wrote.
func serve(t *testing.T, b plugin.Backend, msgs ...proto.Message) (*plugin.Conn, error) {
	t.Helper()
	var in, out bytes.Buffer
	send := plugin.NewConn(nil, &in)
	for _, m := range msgs {
		if err := send.Send(m); err != nil {
			t.Fatal(err)
		}
	}
	err := plugin.ServeConn(context.Background(), b, plugin.NewConn(&in, &out))
	return plugin.NewConn(&out, nil), err
}

func hello(mode plugin.Mode) *plugin.Handshake {
	return &plugin.Handshake{FramingVersion: plugin.FramingVersion, IrVersion: plugin.IRVersion, Mode: mode}
}

func TestServeImports(t *testing.T) {
	req := &plugin.ImportRequest{
		Target:     "reader",
		Files:      []*plugin.File{{Path: "shop.reader"}},
		AllowLossy: []string{"lossy.order"},
	}
	got, err := serve(t, reader{}, hello(plugin.Mode_MODE_IMPORT), req)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}

	var reply plugin.HandshakeReply
	if err := got.Recv(&reply); err != nil {
		t.Fatal(err)
	}
	if !reply.GetAccepted() || !reply.GetFeatures().GetReverse() {
		t.Fatalf("reply = %v, want accepted with reverse", &reply)
	}

	var resp plugin.ImportResponse
	if err := got.Recv(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.GetModel().GetPackage() != "shop.reader" {
		t.Errorf("model = %v", resp.GetModel())
	}
	if d := resp.GetDiagnostics(); len(d) != 1 || d[0].GetCode() != "lossy.order" || d[0].GetMessage() != "allowed: lossy.order" {
		t.Errorf("diagnostics = %v", d)
	}
}

// A backend that only generates refuses import mode in the handshake, so
// it never reads an ImportRequest as a Request.
func TestServeRefusesImportWithoutReverse(t *testing.T) {
	got, err := serve(t, shout{}, hello(plugin.Mode_MODE_IMPORT))
	if err == nil {
		t.Fatal("a generate-only backend accepted import mode")
	}

	var reply plugin.HandshakeReply
	if err := got.Recv(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.GetAccepted() || !strings.Contains(reply.GetRefusal(), "does not import") {
		t.Errorf("reply = %v, want a refusal saying it does not import", &reply)
	}
}

// An importer still generates when the handshake does not ask to import.
func TestServeGeneratesByDefault(t *testing.T) {
	got, err := serve(t, reader{}, hello(plugin.Mode_MODE_UNSPECIFIED), &plugin.Request{Model: &ir.Model{}})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	var reply plugin.HandshakeReply
	if err := got.Recv(&reply); err != nil {
		t.Fatal(err)
	}
	var resp plugin.Response
	if err := got.Recv(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.GetFiles()) == 0 {
		t.Error("generating returned no files")
	}
}
