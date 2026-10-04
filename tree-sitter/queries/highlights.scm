; Syntax highlighting for TDL.
;
; Hand-written; tools/treesitter does not emit this file. Capture names are
; the nvim-treesitter set.
;
; tree-sitter/corpus.sh compiles it, so a node renamed in docs/grammar.ebnf
; fails the build. TestHighlightsCoverKeywords in internal/treesitter checks
; that every spelling in lex.Keywords() appears below.

; ---- keywords -------------------------------------------------------------
;
; Every spelling in lex.Keywords().

[
  "alias"
  "as"
  "class"
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
  "where"
] @keyword

[
  "false"
  "null"
  "true"
] @constant.builtin

; ---- modifiers ------------------------------------------------------------
;
; Contextual rather than reserved: each is usable as a field name.

"owned" @attribute

(deprecated) @attribute

; ---- names ----------------------------------------------------------------

(alias_decl (identifier) @type)
(class_decl (identifier) @type)
(enum_decl (identifier) @type)
(mixin_decl (identifier) @type)
(primitive_decl (identifier) @type)
(type_decl (identifier) @type)
(unit_decl (identifier) @type)

(named_type (dotted_ident (identifier) @type))
(class_ref (dotted_ident (identifier) @type))

(variant (identifier) @constructor)

(type_param (identifier) @variable.parameter)

(field (name) @property)

(package_decl (name) @module)
(target_decl (name) @module)
(import_decl (identifier) @module)

; Target paths and directive names admit reserved words, so they are `name`.
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
; Every spelling in lex.Punctuation(), grouped by how it reads.

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
