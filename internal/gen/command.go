package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/internal/host"
)

// Command is a command line tdl.toml declares, under the name a target
// block's directive gives it.
type Command struct {
	Name string
	Line string
}

// Configure reads the commands a target's post directive names from the
// manifest. A name the manifest does not declare, or no manifest at all,
// is an error, since skipping the command would make what is written
// depend on the machine.
func (t *Target) Configure(cfg *config.Config) error {
	t.Root = cfg.Dir()
	var err error
	t.Post, err = commands(t.Name, host.Names(t.Block, host.PostDirective), host.PostDirective, cfg.Post, cfg.Path)
	return err
}

// commands looks each name up in the table a manifest declares for a
// directive.
func commands(target string, names []string, table string, lines map[string]string, manifest string) ([]Command, error) {
	cmds := make([]Command, 0, len(names))
	for _, name := range names {
		line, ok := lines[name]
		switch {
		case manifest == "":
			return nil, fmt.Errorf("target %s: %s names %q, and no %s was found above the model to declare it; add one holding [%s] %s = \"<command line>\"",
				target, table, name, config.FileName, table, name)
		case !ok:
			return nil, fmt.Errorf("target %s: %s names %q, which [%s] in %s does not declare; add %s = \"<command line>\" there",
				target, table, name, table, manifest, name)
		}
		cmds = append(cmds, Command{Name: name, Line: line})
	}
	return cmds, nil
}

// run runs a command line in dir, replacing each placeholder in its words,
// with stdin as its input, and returns its stdout. A program named by a
// relative path, such as ./scripts/fmt, is relative to root, the directory
// holding tdl.toml, wherever tdl runs. A command that is not found or that
// exits non-zero is an error carrying what it wrote to stderr.
func (c Command) run(ctx context.Context, root, dir string, placeholders map[string]string, stdin []byte) ([]byte, []byte, error) {
	words, err := splitWords(c.Line)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", c.Line, err)
	}
	for i, w := range words {
		for k, v := range placeholders {
			w = strings.ReplaceAll(w, k, v)
		}
		words[i] = w
	}
	program := words[0]
	if strings.ContainsRune(program, '/') && !filepath.IsAbs(program) {
		program = filepath.Join(root, program)
	}
	path, err := exec.LookPath(program)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.CommandContext(ctx, path, words[1:]...)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s: %w", c.Line, err)
	}
	return stdout.Bytes(), stderr.Bytes(), nil
}

// withOutput appends a command's output to its error, when it wrote any.
func withOutput(err error, output []byte) error {
	if out := strings.TrimRight(string(output), "\n"); out != "" {
		return fmt.Errorf("%w\n%s", err, out)
	}
	return err
}

// splitWords splits a command line into words as a POSIX shell would, but
// for expansion: whitespace separates words, single quotes keep everything
// literally, and inside double quotes or outside quotes a backslash keeps
// the next character. Nothing is expanded and no shell runs.
func splitWords(line string) ([]string, error) {
	var words []string
	var word strings.Builder
	inWord := false
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ', '\t', '\n':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		case '\'':
			end := strings.IndexByte(line[i+1:], '\'')
			if end < 0 {
				return nil, errors.New("unterminated single quote")
			}
			word.WriteString(line[i+1 : i+1+end])
			i += end + 1
			inWord = true
		case '"':
			i++
			for ; i < len(line) && line[i] != '"'; i++ {
				if line[i] == '\\' && i+1 < len(line) && strings.IndexByte(`"\$`+"`", line[i+1]) >= 0 {
					i++
				}
				word.WriteByte(line[i])
			}
			if i >= len(line) {
				return nil, errors.New("unterminated double quote")
			}
			inWord = true
		case '\\':
			if i+1 >= len(line) {
				return nil, errors.New("a trailing backslash escapes nothing")
			}
			i++
			word.WriteByte(line[i])
			inWord = true
		default:
			word.WriteByte(line[i])
			inWord = true
		}
	}
	if inWord {
		words = append(words, word.String())
	}
	if len(words) == 0 {
		return nil, errors.New("the command line is empty")
	}
	return words, nil
}
