package host_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/unstoppablemango/tdl/internal/host"
)

// A protobuf tree yields every .proto under it, sorted, without
// dot-directories, each spelled under the directory as given.
func TestImportFilesWalksATree(t *testing.T) {
	dir := t.TempDir()
	for _, path := range []string{"shop/v1/order.proto", "shop/billing.proto", "a.proto", ".git/x.proto", "shop/README.md", "shop/.cache/y.proto"} {
		full := filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := host.ImportFiles("protobuf", dir, []string{"*.proto"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(dir, "a.proto"),
		filepath.Join(dir, "shop", "billing.proto"),
		filepath.Join(dir, "shop", "v1", "order.proto"),
	}
	if !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}

	// A dot-directory named directly is walked.
	got, err = host.ImportFiles("protobuf", filepath.Join(dir, ".git"), []string{"*.proto"})
	if err != nil || len(got) != 1 {
		t.Errorf("files = %v, %v, want the one .proto under .git", got, err)
	}

	_, err = host.ImportFiles("protobuf", filepath.Join(dir, "shop"), []string{"*.thrift"})
	if err == nil || !strings.Contains(err.Error(), "holds no file protobuf reads (*.thrift)") {
		t.Errorf("err = %v, want one naming the empty match", err)
	}
	_, err = host.ImportFiles("debug", dir, nil)
	if err == nil || !strings.Contains(err.Error(), "debug names no files") {
		t.Errorf("err = %v, want a refusal naming the backend", err)
	}
}
