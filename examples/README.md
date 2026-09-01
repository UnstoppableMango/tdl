# Examples

Starting points for playing with the language.
They are not part of the conformance corpus, so edit them freely.

All four parse.
They are written for reading rather than in canonical form: `tdl fmt` groups them differently and drops the `//` comments, which is a known gap.

| File | What it shows |
| --- | --- |
| `flat.tdl` | One entity, parallel lists, optionality doing the modelling work |
| `nested.tdl` | The same domain split into entities, values, and enums |
| `collections.tdl` | Nested collections, every optionality form, qualified references |
| `targets.tdl` | `nested.tdl` plus a class and the backend mapping that stays out of the model |

## Twisting them

```shell
tdl play examples/nested.tdl --views all   # watch: re-renders on every save
tdl ast examples/nested.tdl                # parse tree, one node per line
tdl tokens examples/nested.tdl             # token stream
tdl fmt examples/flat.tdl                  # canonical formatting
```

`tdl play` with no argument uses `scratch.tdl` in the current directory and creates it from a starter template if it is missing.

Things to try:

- Change an `entity` to a `value` and ask whether the thing it describes still makes sense without identity.
- Delete a `?` and watch the `optional` count in the `stats` view move.
- Replace `customer: Customer` with the five inlined fields from `flat.tdl` and compare `stats`.
- Break a line on purpose. The `errors` view points a caret at the column, and parsing continues past it to the next declaration.
- Put a comma between two fields. Whitespace is insignificant and commas are not block separators, so it is a syntax error.
- Name a field `value` or `type`. A reserved word followed by `:` is a field name.
- Invent a constraint: `where { between(0, 100) }`. The set is open, so the parser takes any name.
