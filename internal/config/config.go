// Package config reads tdl.toml, the project manifest. It reads only the
// [lossy], [post], and [check] tables so far; issue #1307 describes the
// rest.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// FileName is the manifest's name.
const FileName = "tdl.toml"

// Config is what a manifest says. The zero value is a project with no
// manifest.
type Config struct {
	// Path is the manifest read, or empty when none was found.
	Path string

	// Allow holds the loss codes [lossy] allows for every target.
	Allow []string

	// Targets holds, per target, the codes [lossy.<target>] allows.
	Targets map[string][]string

	// Post maps each name a target block's post directive may use to the
	// command line [post] gives it.
	Post map[string]string

	// Check maps each name a target block's check directive may use to the
	// command line [check] gives it.
	Check map[string]string
}

// Dir is the directory holding the manifest, or "" when none was found.
func (c *Config) Dir() string {
	if c.Path == "" {
		return ""
	}
	return filepath.Dir(c.Path)
}

// Find reads the manifest in dir or the nearest directory above it. It
// returns the zero Config when there is none.
func Find(dir string) (*Config, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	for {
		path := filepath.Join(dir, FileName)
		if _, err := os.Stat(path); err == nil {
			return Load(path)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return &Config{}, nil
		}
		dir = parent
	}
}

// Load reads one manifest. A key it does not know inside [lossy] is an
// error, so a misspelled `allow` is not silently ignored, and every entry
// of [post] and [check] must be a string; other tables are left to the
// commands that read them.
func Load(path string) (*Config, error) {
	var raw struct {
		Lossy map[string]toml.Primitive `toml:"lossy"`
		Post  map[string]string         `toml:"post"`
		Check map[string]string         `toml:"check"`
	}
	md, err := toml.DecodeFile(path, &raw)
	if err != nil {
		return nil, err
	}

	c := &Config{Path: path, Targets: map[string][]string{}, Post: raw.Post, Check: raw.Check}
	for key, prim := range raw.Lossy {
		if key == "allow" {
			if err := md.PrimitiveDecode(prim, &c.Allow); err != nil {
				return nil, fmt.Errorf("%s: lossy.allow: %w", path, err)
			}
			continue
		}

		var target struct {
			Allow []string `toml:"allow"`
		}
		if err := md.PrimitiveDecode(prim, &target); err != nil {
			return nil, fmt.Errorf("%s: lossy.%s: %w", path, key, err)
		}
		c.Targets[key] = target.Allow
	}

	for _, k := range md.Undecoded() {
		if k[0] == "lossy" {
			return nil, fmt.Errorf("%s: unknown key %s", path, strings.Join(k, "."))
		}
	}
	return c, nil
}

// Allowed returns the loss codes allowed for one target: those [lossy]
// allows for every target, then those [lossy.<target>] adds.
func (c *Config) Allowed(target string) []string {
	return slices.Concat(c.Allow, c.Targets[target])
}
