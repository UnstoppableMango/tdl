package gen

import (
	"os"
	"time"
)

// WatchInterval is how often a watched file is checked.
//
// Polling rather than an OS notification API keeps this dependency-free
// and behaves the same everywhere; a second is fast enough for a person
// saving a file and slow enough to be free.
const WatchInterval = time.Second

// Watch calls onChange whenever path's contents change, until ctx is done.
//
// It compares contents rather than modification time, so an editor that
// rewrites a file without changing it does not trigger a regeneration.
//
// A poll can land while a file is being written, and an editor that
// truncates before writing will briefly present an empty file. That reads
// as a change, and the regeneration it triggers fails to parse. The next
// poll sees the finished file and succeeds, so it corrects itself; an
// editor that saves by renaming never shows the intermediate state at
// all.
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
				// A file that has gone missing may be mid-save. Wait for it
				// rather than reporting a change that has not happened.
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
