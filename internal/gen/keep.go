package gen

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// A backend that declares itself nondeterministic may answer one request
// with different files each time, so tdl keeps what it wrote while the
// request is unchanged (docs/adr/0002-host-reuses-nondeterministic-output.md).
// The marker records, per target, the request's hash with how many files it
// wrote, a hash of each, and the diagnostics the backend reported, as
// comment lines that [Owned] skips:
//
//	#@ yolo request <sha256> 1
//	#@ yolo file <sha256> lib/order.ex
//	#@ yolo diagnostics <base64 of a Response holding only diagnostics>

// keptPrefix starts a marker line recording a kept target.
const keptPrefix = "#@ "

// Kept is what the marker records about one nondeterministic target's last
// generation: the hash of the request, each file it wrote with the hash of
// its content, by path joined to out, and the diagnostics it reported, which
// a run keeping the files reports again.
type Kept struct {
	Request     string
	Files       map[string]string
	Diagnostics []*plugin.Diagnostic

	count int // how many files the request line says were written
}

// Complete says the record lists as many files as its request line says,
// so a marker that lost lines is not read as an output that wrote fewer.
func (k Kept) Complete() bool { return k.count == len(k.Files) }

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

// Keep records in out's marker that target's request hashed to request,
// wrote written, and reported diags, replacing what it recorded before.
func Keep(out, target, request string, written []string, diags []*plugin.Diagnostic) error {
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
	kept[target] = Kept{Request: request, Files: files, Diagnostics: diags, count: len(files)}
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
			hash, count, _ := strings.Cut(rest, " ")
			k.Request = hash
			k.count, _ = strconv.Atoi(count)
		case "file":
			sum, rel, _ := strings.Cut(rest, " ")
			if k.Files == nil {
				k.Files = map[string]string{}
			}
			k.Files[filepath.Join(out, filepath.FromSlash(rel))] = sum
		case "diagnostics":
			var resp plugin.Response
			if b, err := base64.StdEncoding.DecodeString(rest); err == nil && proto.Unmarshal(b, &resp) == nil {
				k.Diagnostics = resp.GetDiagnostics()
			}
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
		var files []string
		for _, path := range k.Paths() {
			rel, err := filepath.Rel(out, path)
			if err != nil || !owned[path] {
				continue
			}
			files = append(files, fmt.Sprintf("%s%s file %s %s\n", keptPrefix, target, k.Files[path], filepath.ToSlash(rel)))
		}
		// A file no longer listed shortens the record, so the count stays
		// what the request wrote and the record reads as incomplete.
		fmt.Fprintf(b, "%s%s request %s %d\n", keptPrefix, target, k.Request, k.count)
		for _, line := range files {
			b.WriteString(line)
		}
		if len(k.Diagnostics) > 0 {
			if enc, err := (proto.MarshalOptions{Deterministic: true}).Marshal(&plugin.Response{Diagnostics: k.Diagnostics}); err == nil {
				fmt.Fprintf(b, "%s%s diagnostics %s\n", keptPrefix, target, base64.StdEncoding.EncodeToString(enc))
			}
		}
	}
}
