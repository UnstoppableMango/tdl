package gen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/unstoppablemango/tdl/plugin"
)

// startScript runs body as a plugin and sends handshakes until a write
// fails, which it does once the plugin closes its stdin.
func startUntilBrokenPipe(t *testing.T, ctx context.Context, body string) (*session, error) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the plugin is a shell script")
	}
	path := filepath.Join(t.TempDir(), CommandPrefix+"closes")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := (&Subprocess{Name: "closes", Path: path}).start(ctx)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := s.conn.Send(&plugin.Handshake{}); err != nil {
			return s, err
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("writing to a plugin that closed its stdin never failed")
	return nil, nil
}

// A plugin that stops reading before it writes its stderr still has that
// stderr in the error (#1545).
func TestWrapWaitsAfterABrokenPipe(t *testing.T) {
	s, err := startUntilBrokenPipe(t, context.Background(),
		"exec 0<&-\nsleep 0.3\necho 'the backend fell over' >&2\nexit 3\n")
	defer s.close()
	if !errors.Is(err, syscall.EPIPE) {
		t.Fatalf("the write failed with %v, not a broken pipe", err)
	}

	got := s.wrap(err).Error()
	if !strings.Contains(got, "the backend fell over") {
		t.Errorf("the plugin's stderr was not reported: %s", got)
	}
	if !strings.Contains(got, "closes stopped before answering") {
		t.Errorf("the error does not say the plugin stopped: %s", got)
	}
}

// A plugin that closes its stdin but keeps running holds the error only
// until the deadline, when CommandContext kills it.
func TestWrapWaitsOnlyUntilTheDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	s, err := startUntilBrokenPipe(t, ctx, "exec 0<&-\necho 'still here' >&2\nexec sleep 60\n")
	defer s.close()

	got := s.wrap(err).Error()
	if ctx.Err() == nil {
		t.Errorf("wrap returned before the deadline: %s", got)
	}
	if !strings.Contains(got, "still here") {
		t.Errorf("the stderr written so far was not reported: %s", got)
	}
}
