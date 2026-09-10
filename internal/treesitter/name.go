package treesitter

import (
	"fmt"
	"strings"
	"unicode"
)

// ruleName is the tree-sitter spelling of an EBNF name.
//
// The two notations disagree about case: the grammar writes a nonterminal
// PackageDecl and a lexical name package_decl, and every published
// tree-sitter grammar writes both the second way. A name already spelled
// that way passes through, so the lexical productions are unchanged.
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
			// A boundary is a capital after a lower-case letter or a
			// digit, or the last capital of a run before a lower-case
			// letter, so IRNode reads ir_node rather than i_r_node.
			if !unicode.IsUpper(prev) || unicode.IsLower(next) {
				out.WriteByte('_')
			}
		}
		out.WriteRune(unicode.ToLower(r))
	}

	return out.String()
}

// names maps every EBNF name to the rule that stands for it. A hidden
// production gains the leading underscore tree-sitter reads as "structure
// the tree without appearing in it".
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

	// An extra is a name with no production, so it has no annotations of
	// its own and is never hidden.
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

// ref is how an expression names a rule.
func (e *emitter) ref(name string) (string, error) {
	rule, ok := e.rules[name]
	if !ok {
		return "", fmt.Errorf("%s is not a production", name)
	}
	return "$." + rule, nil
}
