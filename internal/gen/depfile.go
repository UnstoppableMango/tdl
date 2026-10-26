package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DepFile renders a make dependency file in the style of gcc -MD -MP: one
// rule naming target and every prerequisite, then an empty rule for each
// prerequisite, so deleting one makes make rebuild rather than fail.
func DepFile(target string, prereqs []string) []byte {
	prereqs = slices.Compact(slices.Sorted(slices.Values(prereqs)))

	var b strings.Builder
	b.WriteString(makeEscape(target) + ":")
	for _, p := range prereqs {
		b.WriteString(" " + makeEscape(p))
	}
	b.WriteString("\n")
	for _, p := range prereqs {
		b.WriteString(makeEscape(p) + ":\n")
	}
	return []byte(b.String())
}

// makeEscape escapes what GNU make would read as the end of a name or the
// start of a comment or a reference, as gcc does.
func makeEscape(name string) string {
	return strings.NewReplacer(" ", `\ `, "#", `\#`, "$", "$$").Replace(name)
}

// WriteDepFile writes the dependency file DepFile renders at target. It
// writes a temporary file and renames it over target, so an interrupted run
// never leaves a partial rule, and the mtime make reads is always new.
func WriteDepFile(target string, prereqs []string) error {
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".*")
	if err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(DepFile(target, prereqs)); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", target, err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", target, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return fmt.Errorf("writing %s: %w", target, err)
	}
	return nil
}
