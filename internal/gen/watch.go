package gen

import (
	"os"
	"time"
)

// WatchInterval is how often a watched file is polled.
const WatchInterval = time.Second

// Watch calls onChange whenever path's contents change, until done is
// closed. It compares contents rather than modification time, so a
// rewrite with no change does not trigger. A poll landing mid-save may
// see a truncated file; the next poll corrects it.
func Watch(done <-chan struct{}, path string, onChange func()) {
	last, _ := os.ReadFile(path)

	ticker := time.NewTicker(WatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			current, err := os.ReadFile(path)
			if err != nil {
				// A missing file may be mid-save.
				continue
			}
			if string(current) == string(last) {
				continue
			}
			last = current
			onChange()
		}
	}
}
