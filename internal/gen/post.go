package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/host"
	"github.com/unstoppablemango/tdl/ir"
	"github.com/unstoppablemango/tdl/plugin"
)

// filePlaceholder is replaced, in each word of a post command line, with
// the destination path of the file being processed.
const filePlaceholder = "{file}"

// Post is the commands a target block's post directive names, read through
// tdl.toml's [post] table. Each is a filter over one file: tdl writes the
// content to its standard input and takes its standard output in its
// place.
type Post struct {
	// Dir is the directory holding tdl.toml. Every command runs there, and
	// {file} is relative to it.
	Dir string

	// Commands run in the order the directive names them, each one's output
	// feeding the next.
	Commands []Command
}

// Command is one post command.
type Command struct {
	// Name is what the directive and [post] call it.
	Name string

	// Args is the command line split into words, with {file} left in.
	Args []string
}

// ResolvePost reads a target block's post directive through cfg, the
// tdl.toml nearest the model. It returns nil when the block has none, and
// an error naming the target for a name cfg does not declare, including
// when there is no tdl.toml.
func ResolvePost(block *ir.TargetBlock, cfg *config.Config) (*Post, error) {
	target := block.GetMeta().GetName()
	var names []string
	for _, d := range block.GetDirectives() {
		if d.GetName() == host.PostDirective {
			for _, a := range d.GetArgs() {
				names = append(names, a.GetText())
			}
		}
	}
	if len(names) == 0 {
		return nil, nil
	}

	post := &Post{Dir: cfg.Dir()}
	for _, name := range names {
		if cfg.Path == "" {
			return nil, fmt.Errorf("target %s: post names %q, but no %s was found above the model; add one with a [post] table declaring %s = \"<command>\"",
				target, name, config.FileName, name)
		}
		line, ok := cfg.Post[name]
		if !ok {
			return nil, fmt.Errorf("target %s: post names %q, which [post] in %s does not declare; add %s = \"<command>\" there",
				target, name, cfg.Path, name)
		}
		args, err := splitWords(line)
		if err != nil {
			return nil, fmt.Errorf("target %s: post.%s in %s: %w", target, name, cfg.Path, err)
		}
		post.Commands = append(post.Commands, Command{Name: name, Args: args})
	}
	return post, nil
}

// Apply runs every command over every file, returning the files with
// their processed content. out is the target's output directory, which a
// file's path is relative to. A nil Post returns the files unchanged.
func (p *Post) Apply(ctx context.Context, out string, files []*plugin.File) ([]*plugin.File, error) {
	if p == nil || len(p.Commands) == 0 {
		return files, nil
	}
	processed := make([]*plugin.File, len(files))
	for i, f := range files {
		// A path the write would refuse is refused before any command
		// sees it.
		dest, err := resolve(out, f.GetPath())
		if err != nil {
			return nil, err
		}
		if dest, err = filepath.Abs(dest); err != nil {
			return nil, err
		}
		rel, err := filepath.Rel(p.Dir, dest)
		if err != nil {
			return nil, err
		}

		content := f.GetContent()
		for _, c := range p.Commands {
			if content, err = c.run(ctx, p.Dir, rel, content); err != nil {
				return nil, err
			}
		}
		processed[i] = &plugin.File{Path: f.GetPath(), Content: content}
	}
	return processed, nil
}

// run filters one file's content through the command.
func (c Command) run(ctx context.Context, dir, file string, content []byte) ([]byte, error) {
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = strings.ReplaceAll(a, filePlaceholder, file)
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(content)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return stdout.Bytes(), nil
	case errors.Is(err, exec.ErrNotFound), errors.Is(err, os.ErrNotExist):
		return nil, fmt.Errorf("post command %s on %s: %s was not found", c.Name, file, args[0])
	case errors.As(err, &exit):
		msg := fmt.Sprintf("post command %s failed on %s: %v", c.Name, file, err)
		if s := strings.TrimSpace(stderr.String()); s != "" {
			msg += "\n" + s
		}
		return nil, errors.New(msg)
	}
	return nil, fmt.Errorf("post command %s on %s: %w", c.Name, file, err)
}

// splitWords splits a command line into words as a shell would, without
// expanding anything: whitespace separates words, single quotes keep
// everything inside them, a backslash outside quotes keeps the character
// after it, and one inside double quotes does so only before $, `, ", \,
// or a newline, and is otherwise kept itself.
func splitWords(line string) ([]string, error) {
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune
	escaped := false

	for _, r := range line {
		switch {
		case escaped:
			if quote == '"' && !strings.ContainsRune("$`\"\\\n", r) {
				word.WriteRune('\\')
			}
			word.WriteRune(r)
			escaped = false
		case quote == '\'':
			if r == '\'' {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '\\':
			escaped, inWord = true, true
		case quote == '"':
			if r == '"' {
				quote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote, inWord = r, true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	switch {
	case escaped:
		return nil, fmt.Errorf("the command line ends in a backslash")
	case quote != 0:
		return nil, fmt.Errorf("the command line has an unterminated %s quote", strconv.QuoteRune(quote))
	}
	if inWord {
		words = append(words, word.String())
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("the command line is empty")
	}
	return words, nil
}
