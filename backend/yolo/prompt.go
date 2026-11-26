package yolo

import (
	"fmt"
	"strings"

	"github.com/unstoppablemango/tdl/backend/internal/emit"
	"github.com/unstoppablemango/tdl/docs"
	"github.com/unstoppablemango/tdl/plugin"
)

// lossMeanings says what each loss code names, as
// docs/architecture/reverse.md's table does, so the model picks the code
// for what it dropped. TestPrompt keeps it complete.
var lossMeanings = map[string]string{
	emit.LossPrimitive:   "a primitive's width or kind, such as uuid written as a string",
	emit.LossNewtype:     "a newtype written as its base type, with no type of its own",
	emit.LossCollection:  "a set written as a list",
	emit.LossOptional:    "the difference between an optional field (T?) and a nullable one (T | null)",
	emit.LossStructKind:  "whether a struct is an entity, a value, or a mixin",
	emit.LossInclude:     "a mixin include, flattened into the including struct",
	emit.LossAlias:       "an alias expanded at each use",
	emit.LossConstraint:  "a where constraint the output does not enforce",
	emit.LossOwned:       "the owned modifier",
	emit.LossDefault:     "a field default",
	emit.LossKey:         "a key directive",
	emit.LossName:        "a name the language's conventions forced you to change",
	emit.LossNumber:      "a pinned number",
	emit.LossOrder:       "declaration order",
	emit.LossDoc:         "doc comment text",
	emit.LossGeneric:     "type parameters",
	emit.LossClass:       "a class, or a type's conformance to one",
	emit.LossUnit:        "a unit of measure",
	emit.LossUnsupported: "anything else the output cannot express, including a declaration left out entirely",
}

// system is the system prompt: the same for every model, so a provider
// can cache it.
func system() string {
	var b strings.Builder
	b.WriteString(`You are a code generator for TDL, the Type Description Language. TDL describes domain models: entities, values, enums, newtypes, classes, and collections, with no expressions or runtime. Its specification is at the end of this prompt.

The user gives you a model written in TDL and the language to write it in. Write the model's declarations as idiomatic, compiling source in that language, following any instructions the user adds. Write every declaration the model declares; the prelude's types (string, int, List, Option, and so on) are TDL's own and map to the language's. A declaration you leave out takes with it every declaration naming it, so the files still compile; report each one left out.

Write each file with the write_file tool, one call per file. A path is relative to the output directory, uses forward slashes, and may not be absolute or contain "..". Writing a path again replaces the file.

Whatever the files cannot carry from the model, report with the report_loss tool: name the declaration ("Order"), or a field or variant ("Order.total"), the loss code naming what was lost, and a sentence saying what happened. Report each loss once. The loss codes:

`)
	for _, code := range emit.LossCodes {
		fmt.Fprintf(&b, "- %s: %s\n", code, lossMeanings[code])
	}
	b.WriteString(`
When the output reads or writes the model as JSON, follow the JSON wire convention every TDL target shares: a property is named exactly as the model spells the field, whatever the language calls it; a fielded enum's variants are tagged internally by a discriminant property, "kind" unless a discriminant directive names another, whose value is the variant's name in the model; a newtype is its base value; an empty list is written [] and an empty map {}, at every depth, never null.

When every file is written and every loss reported, end your turn without calling a tool.

The TDL specification:

`)
	b.WriteString(docs.Spec)
	return b.String()
}

// userPrompt is the first message: the language, the user's instructions,
// the model, and what it needs from the packages it imports.
func userPrompt(language, instructions string, src *plugin.Source, deps string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Write this model in %s.\n", language)
	if instructions != "" {
		fmt.Fprintf(&b, "\n%s\n", instructions)
	}
	b.WriteString("\nThe model:\n\n```tdl\n")
	b.WriteString(modelText(src))
	b.WriteString("```\n")
	b.WriteString(deps)
	return b.String()
}

// dependencies is what the prompt says about the packages the model
// imports, which are generated separately: each import's language and
// output directory, the declarations the model uses from them, their
// foreign mappings, and the files they generated. It is "" for a model
// importing nothing.
func (g *generator) dependencies(src *plugin.Source) string {
	imports := g.Model.GetImports()
	if len(imports) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\nThe model imports these packages, which are generated separately. Refer to what they declare; never write their declarations yourself.\n\n")
	for _, imp := range imports {
		switch alias := imp.GetAlias(); alias {
		case "_":
			fmt.Fprintf(&b, "- %s, whose names the model uses unqualified", imp.GetPackage())
		default:
			fmt.Fprintf(&b, "- %s, imported as %s", imp.GetPackage(), alias)
		}
		if lang, ok := g.Text(imp.GetDirectives(), "language"); ok {
			fmt.Fprintf(&b, ", written in %s", lang)
		}
		if dir, ok := g.depDir(imp); ok {
			fmt.Fprintf(&b, ", into %s relative to the output directory", dir)
		}
		b.WriteString("\n")
	}

	var decls []string
	for i, ext := range g.Model.GetExterns() {
		if i < len(src.GetExterns()) && src.GetExterns()[i].GetSource() != "" {
			decls = append(decls, fmt.Sprintf("// in package %s\n%s", ext.GetPackage(), src.GetExterns()[i].GetSource()))
		}
	}
	if len(decls) > 0 {
		b.WriteString("\nThe declarations the model uses from them, each declared in another package and generated separately:\n\n```tdl\n")
		b.WriteString(strings.Join(decls, "\n\n"))
		b.WriteString("\n```\n")
	}

	var mapped []string
	for _, ext := range g.Model.GetExterns() {
		if module, name, ok := g.foreign(ext); ok {
			mapped = append(mapped, fmt.Sprintf("- %s.%s is %s from %s; use that and do not write it.\n", ext.GetPackage(), ext.GetName(), name, module))
		}
	}
	if len(mapped) > 0 {
		b.WriteString("\nThese are mapped to existing code:\n\n")
		b.WriteString(strings.Join(mapped, ""))
	}

	if files := g.deps; len(files) > 0 {
		b.WriteString("\nThe imported packages generated these files, each under its path relative to the output directory. Import these paths and the names they declare rather than inventing your own, and do not write these files.\n")
		for _, f := range files {
			fmt.Fprintf(&b, "\n%s, from %s:\n\n````\n%s", f.GetPath(), f.GetPackage(), f.GetContent())
			if !strings.HasSuffix(string(f.GetContent()), "\n") {
				b.WriteString("\n")
			}
			b.WriteString("````\n")
		}
	}
	return b.String()
}

// modelText is the model as one TDL file: its imports, then its items.
func modelText(src *plugin.Source) string {
	var parts []string
	if imps := src.GetImports(); len(imps) > 0 {
		parts = append(parts, strings.Join(imps, "\n"))
	}
	for _, it := range src.GetItems() {
		parts = append(parts, it.GetSource())
	}
	return strings.Join(parts, "\n\n") + "\n"
}

// tools are what the model answers through.
var tools = []Tool{
	{
		Name:        writeFile,
		Description: "Write one file of the generated output.",
		Schema: []byte(`{
  "type": "object",
  "properties": {
    "path": {"type": "string", "description": "The file's path, relative to the output directory, with forward slashes."},
    "content": {"type": "string", "description": "The file's whole content."}
  },
  "required": ["path", "content"]
}`),
	},
	{
		Name:        reportLoss,
		Description: "Report a fact of the model the output does not carry.",
		Schema: []byte(`{
  "type": "object",
  "properties": {
    "decl": {"type": "string", "description": "The declaration (Order), or field or variant (Order.total), the loss is about; empty for the model as a whole."},
    "code": {"type": "string", "description": "The loss code naming what was lost."},
    "message": {"type": "string", "description": "One sentence saying what was lost and why."}
  },
  "required": ["decl", "code", "message"]
}`),
	},
}

const (
	writeFile  = "write_file"
	reportLoss = "report_loss"
)
