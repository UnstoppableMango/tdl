package gen_test

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/internal/gen"
	"github.com/unstoppablemango/tdl/internal/gen/echo"
	"github.com/unstoppablemango/tdl/plugin"
)

// importRow is one backend [TestImportHostsAgree] runs, with the files it
// reads.
type importRow struct {
	backend plugin.Backend
	files   []*plugin.File
	needs   func() error
}

// importers are every shipped row with a reverse column, and the echo test
// backend.
func importers(t *testing.T) []importRow {
	t.Helper()

	data, err := protojson.Marshal(orderModel())
	if err != nil {
		t.Fatal(err)
	}
	rows := []importRow{{echo.Backend{}, []*plugin.File{{Path: echo.File, Content: data}}, nil}}
	for _, row := range shipped {
		if row.reverse != nil {
			rows = append(rows, importRow{row.backend, row.reverse(), row.needs})
		}
	}
	return rows
}

// A compiled-in importer and the same backend as a subprocess return the
// same model and diagnostics.
func TestImportHostsAgree(t *testing.T) {
	onPath(t, pluginDir(t))

	for _, row := range importers(t) {
		name := row.backend.Describe().Name
		t.Run(name, func(t *testing.T) {
			if row.needs != nil {
				if err := row.needs(); err != nil {
					t.Skip(err)
				}
			}
			importer, ok := row.backend.(plugin.Importer)
			if !ok {
				t.Fatalf("%s has a reverse column and does not implement plugin.Importer", name)
			}
			sub, err := gen.Find(name)
			if err != nil {
				t.Fatalf("find: %v", err)
			}
			if !sub.Describe().Reverse {
				t.Fatal("the plugin's handshake does not declare reverse")
			}

			req := &plugin.ImportRequest{Target: name, Files: row.files}
			inProcess, err := importer.Import(context.Background(), req)
			if err != nil {
				t.Fatalf("in process: %v", err)
			}
			viaPipe, err := sub.Import(context.Background(), req)
			if err != nil {
				t.Fatalf("subprocess: %v", err)
			}

			if inProcess.GetModel() == nil {
				t.Fatalf("the files imported no model: %v", inProcess.GetDiagnostics())
			}
			if !proto.Equal(inProcess, viaPipe) {
				t.Errorf("the hosts disagree\nin process: %v\nsubprocess: %v", inProcess, viaPipe)
			}
		})
	}
}

// The echo backend imports exactly what it generated.
func TestEchoRoundTrips(t *testing.T) {
	resp, err := echo.Backend{}.Generate(context.Background(), &plugin.Request{Model: orderModel()})
	if err != nil {
		t.Fatal(err)
	}
	got, err := echo.Backend{}.Import(context.Background(), &plugin.ImportRequest{Files: resp.GetFiles()})
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(got.GetModel(), orderModel()) {
		t.Errorf("got %v, want %v", got.GetModel(), orderModel())
	}
}

// A backend declares reverse exactly when it implements plugin.Importer,
// since the host reads the declaration and calls the method.
func TestReverseIsDeclared(t *testing.T) {
	for _, row := range shipped {
		name := row.backend.Describe().Name
		_, imports := row.backend.(plugin.Importer)
		if declared := row.backend.Describe().Reverse; declared != imports {
			t.Errorf("%s: declares reverse %t, implements Importer %t", name, declared, imports)
		}
		if imports != (row.reverse != nil) {
			t.Errorf("%s: implements Importer %t, has a reverse column %t", name, imports, row.reverse != nil)
		}
	}
}

// A plugin that only generates is never sent an import request.
func TestImportRefusedByAGenerator(t *testing.T) {
	onPath(t, pluginDir(t))

	sub, err := gen.Find("debug")
	if err != nil {
		t.Fatal(err)
	}
	_, err = sub.Import(context.Background(), &plugin.ImportRequest{Target: "debug"})
	if err == nil || !strings.Contains(err.Error(), "does not import") {
		t.Errorf("err = %v, want a refusal saying debug does not import", err)
	}
}

// A warning with an allowed code is dropped; an error and a warning with
// no code never are.
func TestSilence(t *testing.T) {
	diags := []*plugin.Diagnostic{
		{Severity: plugin.Severity_SEVERITY_WARNING, Message: "order", Code: "lossy.order"},
		{Severity: plugin.Severity_SEVERITY_WARNING, Message: "set", Code: "lossy.collection"},
		{Severity: plugin.Severity_SEVERITY_ERROR, Message: "broken", Code: "lossy.order"},
		{Severity: plugin.Severity_SEVERITY_WARNING, Message: "plain"},
	}
	got := gen.Silence(diags, []string{"lossy.order", ""})

	var msgs []string
	for _, d := range got {
		msgs = append(msgs, d.GetMessage())
	}
	if want := "set broken plain"; strings.Join(msgs, " ") != want {
		t.Errorf("kept %v, want %s", msgs, want)
	}
	if len(diags) != 4 {
		t.Error("Silence changed its argument")
	}
}
