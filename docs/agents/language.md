# Language conventions

Whitespace is insignificant and there are no separators: an item ends where the next begins.
Commas are required inside `<...>`, conformance lists, and list literals, and not permitted inside `{ }`.

`where` introduces a constraint block; `requires` introduces class constraints on parameters.

Declaration keywords are reserved.
Modifiers and constraint names (`owned`, `deprecated`, `min`, `max`, `length`, `matches`, `oneOf`, `unique`) are contextual and usable as field names.
A reserved word or contextual modifier followed by `:` is a field name: `include Foo` is an include, `include: Foo` a field.

Identity is conformance to the prelude's `Entity` class, known to the compiler by name.
`ir.StructKind` is computed from that conformance, and the recursion rules read it.
Which fields identify an entity is a target directive.

A `<...>` argument is a type or a unit.
A bare name is recorded as a type reference and resolved by kind; only an operator (`*`, `/`, `^`) or parentheses makes it a unit.

In a target block, directive names and path segments may be reserved words.
A package path segment may be a reserved word after `package` or a target block's `for`, so `package google.type` works.
`Name` in the grammar is that production; every other name is `identifier`.

Directive and constraint arguments are parenthesized and comma separated, since the parser knows no name's arity.

The parser calls `lex.RescanRegexAt` when it wants a regex; nothing else in the lexer takes context.

Comments survive formatting.
A `///` doc comment is a token attached to the next declaration, in `DeclHead.Doc` with each line's position in `DeclHead.DocP`.
A `//` comment is collected on the side into `ast.File.Comments` in source order.
`ast.Fprint` places each by position, on its own line or at the end of the line it was on; a block holding one does not collapse to a line.
Doc and ordinary comments are merged by offset, so they keep their order.
