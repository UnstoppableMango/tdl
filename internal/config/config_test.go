package config_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/config"
)

func write(t *testing.T, path, src string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The manifest is found from a directory below it, and a target's codes
// follow those allowed for every target.
func TestFindWalksUp(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, config.FileName), `
[project]
name = "shop"

[lossy]
allow = ["lossy.order"]

[lossy.protobuf]
allow = ["lossy.collection", "lossy.newtype"]
`)
	dir := filepath.Join(root, "models", "shop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	c, err := config.Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Path != filepath.Join(root, config.FileName) {
		t.Errorf("path = %q", c.Path)
	}
	if got, want := c.Allowed("protobuf"), []string{"lossy.order", "lossy.collection", "lossy.newtype"}; !slices.Equal(got, want) {
		t.Errorf("protobuf allows %v, want %v", got, want)
	}
	if got, want := c.Allowed("thrift"), []string{"lossy.order"}; !slices.Equal(got, want) {
		t.Errorf("thrift allows %v, want %v", got, want)
	}
}

// The nearest manifest wins; one further up is not merged in.
func TestFindStopsAtTheNearest(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, config.FileName), "[lossy]\nallow = [\"lossy.order\"]\n")
	write(t, filepath.Join(root, "sub", config.FileName), "[lossy]\nallow = [\"lossy.doc\"]\n")

	c, err := config.Find(filepath.Join(root, "sub"))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Allowed("go"); !slices.Equal(got, []string{"lossy.doc"}) {
		t.Errorf("allowed %v, want only the nearest manifest's", got)
	}
}

func TestFindWithoutAManifest(t *testing.T) {
	c, err := config.Find(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if c.Path != "" || len(c.Allowed("go")) != 0 {
		t.Errorf("got %+v, want the zero config", c)
	}
}

// A misspelled key in [lossy] is an error rather than a silent no-op.
func TestLoadRejectsUnknownLossyKeys(t *testing.T) {
	cases := map[string]string{
		"lossy.alow":          "[lossy]\nalow = [\"lossy.order\"]\n",
		"lossy.protobuf.alow": "[lossy.protobuf]\nalow = [\"lossy.order\"]\n",
	}
	for want, src := range cases {
		t.Run(want, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), config.FileName)
			write(t, path, src)
			_, err := config.Load(path)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("err = %v, want one naming %s", err, want)
			}
		})
	}
}

func TestLoadRejectsAWrongType(t *testing.T) {
	path := filepath.Join(t.TempDir(), config.FileName)
	write(t, path, "[lossy]\nallow = \"lossy.order\"\n")
	if _, err := config.Load(path); err == nil {
		t.Error("a string where a list belongs was accepted")
	}
}
