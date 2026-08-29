# Examples

Starting points for playing with the language.
They are not part of the conformance corpus, so edit them freely.

| File | What it shows |
| --- | --- |
| `flat.tdl` | One record, parallel lists, optionality doing the modelling work |
| `nested.tdl` | The same domain split into records and an enum |
| `annotated.tdl` | `nested.tdl` plus backend mapping annotations |
| `collections.tdl` | Nested `list`/`map`, qualified references, every literal kind |

## Twisting them

```shell
tdl play examples/nested.tdl --views all   # watch: re-renders on every save
tdl ast examples/nested.tdl                # parse tree, one node per line
tdl tokens examples/nested.tdl             # token stream
tdl fmt examples/flat.tdl                  # canonical formatting
```

`tdl play` with no argument uses `scratch.tdl` in the current directory and creates it from a starter template if it is missing.

Things to try:

- Delete a `?` and watch the `optional` count in the `stats` view move.
- Replace `customer: Customer` with the five inlined fields from `flat.tdl` and compare `stats`.
- Break a line on purpose. The `errors` view points a caret at the column, and parsing continues past it to the next declaration.
- Rename an annotation argument to a keyword (`enum:`, `type:`). M1 rejects keywords there.
- Write `union Foo { ... }` or `type Box<T> { ... }`. Both are reserved in the grammar and unimplemented, so both are syntax errors.
