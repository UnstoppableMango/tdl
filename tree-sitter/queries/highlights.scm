; Syntax highlighting for TDL.
;
; Hand-written, unlike grammar.js. What should be colored is a judgment
; rather than a fact about the grammar, so tools/treesitter does not emit
; this file; docs/design/treesitter.md says so.
;
; Capture names are the nvim-treesitter set, which every consumer knows.
;
; Two things hold it to the language. `tree-sitter query` runs it over the
; corpus, so a node renamed in docs/grammar.ebnf breaks the build rather
; than quietly uncoloring something. And TestHighlightsCoverKeywords in
; internal/treesitter checks every spelling in lex.Keywords() appears
; below, because a keyword is an anonymous token and a missing one is
; invisible otherwise.

; ---- keywords -------------------------------------------------------------
;
; Every spelling in lex.Keywords(). `where` and `requires` introduce blocks
; rather than declarations, and are keywords all the same.

[
  "alias"
  "as"
  "class"
  "entity"
  "enum"
  "for"
  "import"
  "include"
  "instance"
  "mixin"
  "package"
  "primitive"
  "requires"
  "target"
  "type"
  "unit"
  "value"
  "where"
] @keyword

[
  "false"
  "null"
  "true"
] @constant.builtin

; ---- modifiers ------------------------------------------------------------
;
; Contextual rather than reserved: each is usable as a field name, and each
; reads as an annotation on the declaration it precedes.

[
  "key"
  "owned"
] @attribute

(deprecated) @attribute

; ---- names ----------------------------------------------------------------

(alias_decl (identifier) @type)
(class_decl (identifier) @type)
(entity_decl (identifier) @type)
(enum_decl (identifier) @type)
(mixin_decl (identifier) @type)
(primitive_decl (identifier) @type)
(type_decl (identifier) @type)
(unit_decl (identifier) @type)
(value_decl (identifier) @type)

(named_type (dotted_ident (identifier) @type))
(class_ref (dotted_ident (identifier) @type))

(variant (identifier) @constructor)

(type_param (identifier) @variable.parameter)

(field (name) @property)

(package_decl (dotted_ident (identifier) @module))
(import_decl (identifier) @module)

; A target path is the backend's namespace, and a directive is the call it
; makes there. Both admit reserved words, which is why they are `name`.
(path (name) @module)

(constraint (identifier) @function.call)
(directive (name) @function.call)

; ---- literals -------------------------------------------------------------

(string_lit) @string
(regex_lit) @string.regex
(bool_lit) @boolean

[
  (int_lit)
  (float_lit)
] @number

; ---- comments -------------------------------------------------------------

(line_comment) @comment
(doc_comment) @comment.documentation

; ---- operators and punctuation --------------------------------------------
;
; Every spelling in lex.Punctuation(), split by how it reads rather than by
; what the lexer calls it. `*`, `/`, and `^` are unit arithmetic; `->` and
; `=>` are kinds and functional dependencies; `..` is a range.

[
  "*"
  "/"
  "^"
  "="
  "->"
  "=>"
  ".."
  "?"
  "|"
] @operator

[
  ","
  "."
  ":"
] @punctuation.delimiter

[
  "("
  ")"
  "["
  "]"
  "{"
  "}"
  "<"
  ">"
] @punctuation.bracket
