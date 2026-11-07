package gen

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// A backend that declares itself nondeterministic may answer one request
// with different files each time, so tdl keeps what it wrote while the
// request is unchanged (docs/adr/0002-host-reuses-nondeterministic-output.md).
// The marker records, per target, the request's hash and a hash of each file
// it wrote, as comment lines that [Owned] skips:
//
//	#@ yolo request <sha256>
//	#@ yolo file <sha256> lib/order.ex

// keptPrefix starts a marker line recording a kept target.
const keptPrefix = "#@ "

// Kept is what the marker records about one nondeterministic target's last
// generation: the hash of the request, and each file it wrote with the hash
// of its content, by path joined to out.
type Kept struct {
	Request string
	Files   map[string]string
}

// RequestHash is the hash a request is kept under. It covers everything the
// backend reads but what varies with where tdl runs: positions, the output
// path, and whether this is a dry run.
func RequestHash(req *plugin.Request) (string, error) {
	req = proto.CloneOf(req)
	req.Out = ""
	req.DryRun = false
	ir.ClearPositions(req)
	b, err := proto.MarshalOptions{Deterministic: true}.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("hashing the request: %w", err)
	}
	return digest(b), nil
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Keep records in out's marker that target's request hashed to request and
// wrote written, replacing what it recorded before.
func Keep(out, target, request string, written []string) error {
	owned, err := Owned(out)
	if err != nil {
		return err
	}
	kept, err := readKept(out)
	if err != nil {
		return err
	}
	files := map[string]string{}
	for _, path := range written {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		files[path] = digest(content)
		owned[path] = true
	}
	kept[target] = Kept{Request: request, Files: files}
	return writeMarker(out, owned, kept)
}

// Recorded returns what out's marker records about target, and whether it
// records anything.
func Recorded(out, target string) (Kept, bool, error) {
	kept, err := readKept(out)
	if err != nil {
		return Kept{}, false, err
	}
	k, ok := kept[target]
	return k, ok, nil
}

// Check compares the files a kept generation wrote with what is on disk.
func (k Kept) Check() ([]Stale, error) {
	var stale []Stale
	for _, path := range slices.Sorted(maps.Keys(k.Files)) {
		switch content, err := os.ReadFile(path); {
		case errors.Is(err, os.ErrNotExist):
			stale = append(stale, Stale{Path: path, Reason: "missing"})
		case err != nil:
			return nil, fmt.Errorf("reading %s: %w", path, err)
		case digest(content) != k.Files[path]:
			stale = append(stale, Stale{Path: path, Reason: "differs"})
		}
	}
	return stale, nil
}

// Paths is every file the kept generation wrote.
func (k Kept) Paths() []string { return slices.Sorted(maps.Keys(k.Files)) }

func readKept(out string) (map[string]Kept, error) {
	content, err := os.ReadFile(filepath.Join(out, MarkerName))
	switch {
	case errors.Is(err, os.ErrNotExist):
		return map[string]Kept{}, nil
	case err != nil:
		return nil, fmt.Errorf("reading %s: %w", filepath.Join(out, MarkerName), err)
	}

	kept := map[string]Kept{}
	for line := range strings.Lines(string(content)) {
		rest, ok := strings.CutPrefix(strings.TrimRight(line, "\r\n"), keptPrefix)
		if !ok {
			continue
		}
		target, rest, _ := strings.Cut(rest, " ")
		kind, rest, _ := strings.Cut(rest, " ")
		k := kept[target]
		switch kind {
		case "request":
			k.Request = rest
		case "file":
			sum, rel, _ := strings.Cut(rest, " ")
			if k.Files == nil {
				k.Files = map[string]string{}
			}
			k.Files[filepath.Join(out, filepath.FromSlash(rel))] = sum
		default:
			continue
		}
		kept[target] = k
	}
	return kept, nil
}

// writeKept appends each target's record to a marker, leaving out a file
// the marker no longer lists, and a target with no request.
func writeKept(b *strings.Builder, out string, owned map[string]bool, kept map[string]Kept) {
	for _, target := range slices.Sorted(maps.Keys(kept)) {
		k := kept[target]
		if k.Request == "" {
			continue
		}
		fmt.Fprintf(b, "%s%s request %s\n", keptPrefix, target, k.Request)
		for _, path := range k.Paths() {
			rel, err := filepath.Rel(out, path)
			if err != nil || !owned[path] {
				continue
			}
			fmt.Fprintf(b, "%s%s file %s %s\n", keptPrefix, target, k.Files[path], filepath.ToSlash(rel))
		}
	}
}
