package host

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/unstoppablemango/tdl/internal/config"
	"github.com/unstoppablemango/tdl/ir"
)

// Command is one post or check command.
type Command struct {
	// Name is what the directive and its tdl.toml table call it.
	Name string

	// Args is the command line split into words, with placeholders left in.
	Args []string

	// Line is the command line as the table spells it.
	Line string
}

// Commands reads the names a target block's directive gives, across
// repeated entries, through the manifest table of the same name. It
// returns nil when the block has none, and an error naming the target for
// a name the table does not declare, including when there is no tdl.toml.
func Commands(block *ir.TargetBlock, cfg *config.Config, directive string, table map[string]string) ([]Command, error) {
	target := block.GetMeta().GetName()
	var names []string
	for _, d := range block.GetDirectives() {
		if d.GetName() == directive {
			for _, a := range d.GetArgs() {
				names = append(names, a.GetText())
			}
		}
	}

	var cmds []Command
	for _, name := range names {
		if cfg.Path == "" {
			return nil, fmt.Errorf("target %s: %s names %q, but no %s was found above the model; add one with a [%s] table declaring %s = \"<command>\"",
				target, directive, name, config.FileName, directive, name)
		}
		line, ok := table[name]
		if !ok {
			return nil, fmt.Errorf("target %s: %s names %q, which [%s] in %s does not declare; add %s = \"<command>\" there",
				target, directive, name, directive, cfg.Path, name)
		}
		args, err := splitWords(line)
		if err != nil {
			return nil, fmt.Errorf("target %s: %s.%s in %s: %w", target, directive, name, cfg.Path, err)
		}
		cmds = append(cmds, Command{Name: name, Args: args, Line: line})
	}
	return cmds, nil
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
