package treesitter

import (
	"fmt"
	"strings"
	"unicode"
)

// ruleName is the snake_case tree-sitter spelling of an EBNF name. A name
// already in snake_case passes through.
func ruleName(name string) string {
	var out strings.Builder
	runes := []rune(name)

	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			next := ' '
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			// IRNode reads ir_node, not i_r_node.
			if !unicode.IsUpper(prev) || unicode.IsLower(next) {
				out.WriteByte('_')
			}
		}
		out.WriteRune(unicode.ToLower(r))
	}

	return out.String()
}

// buildNames maps every EBNF name to its rule name. A hidden production
// gains a leading underscore.
func (e *emitter) buildNames() error {
	e.rules = make(map[string]string, len(e.order)+len(e.file.Annotations.Extras))
	from := map[string]string{}

	claim := func(name, rule string) error {
		if other, taken := from[rule]; taken {
			return fmt.Errorf("%s and %s are both %s", other, name, rule)
		}
		from[rule], e.rules[name] = name, rule
		return nil
	}

	for _, prod := range e.order {
		rule := ruleName(prod.Name.String)
		if e.file.Annotations.Prods[prod.Name.String].Hidden {
			rule = "_" + rule
		}
		if err := claim(prod.Name.String, rule); err != nil {
			return err
		}
	}

	// An extra with no production is never hidden.
	for _, extra := range e.file.Annotations.Extras {
		if _, ok := e.rules[extra]; ok {
			continue
		}
		if err := claim(extra, ruleName(extra)); err != nil {
			return err
		}
	}

	return nil
}

// ref returns the `$.rule` reference for an EBNF name.
func (e *emitter) ref(name string) (string, error) {
	rule, ok := e.rules[name]
	if !ok {
		return "", fmt.Errorf("%s is not a production", name)
	}
	return "$." + rule, nil
}
