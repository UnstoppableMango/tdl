package gen_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/unstoppablemango/tdl/internal/gen"
)

func TestWatchNoticesAChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched.tdl")
	if err := os.WriteFile(path, []byte("before"), 0o644); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	defer close(done)

	var mu sync.Mutex
	changes := 0
	go gen.Watch(done, path, func() {
		mu.Lock()
		changes++
		mu.Unlock()
	})

	// Watch reads the file once before it starts polling. Writing before
	// that read would make the new contents the baseline, and nothing would
	// ever look like a change.
	time.Sleep(100 * time.Millisecond)

	if err := os.WriteFile(path, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}

	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		seen := changes
		mu.Unlock()
		if seen > 0 {
			return
		}
		select {
		case <-deadline:
			t.Fatal("the change was never noticed")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// Contents are compared rather than timestamps, so an editor that
// rewrites a file without changing it does not trigger a regeneration.
func TestWatchIgnoresAnIdenticalWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched.tdl")
	if err := os.WriteFile(path, []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	defer close(done)

	var mu sync.Mutex
	changes := 0
	go gen.Watch(done, path, func() {
		mu.Lock()
		changes++
		mu.Unlock()
	})
	time.Sleep(100 * time.Millisecond)

	// Saved by rename, the way an editor that never shows a half-written
	// file does, so this measures the content comparison rather than a
	// race with truncation.
	for i := 0; i < 3; i++ {
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, []byte("same"), 0o644); err != nil {
			t.Fatal(err)
		}
		future := time.Now().Add(time.Duration(i+1) * time.Second)
		if err := os.Chtimes(tmp, future, future); err != nil {
			t.Fatal(err)
		}
		if err := os.Rename(tmp, path); err != nil {
			t.Fatal(err)
		}
		time.Sleep(gen.WatchInterval + 200*time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if changes != 0 {
		t.Errorf("an identical write triggered %d regeneration(s)", changes)
	}
}

func TestWatchStops(t *testing.T) {
	path := filepath.Join(t.TempDir(), "watched.tdl")
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		gen.Watch(done, path, func() {})
		close(stopped)
	}()

	close(done)
	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("the watch did not stop")
	}
}
