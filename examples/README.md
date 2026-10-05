# Examples

Files to experiment with.
They are outside the conformance corpus, so edit them freely.
Each is stored in canonical form, comments included.

| File | Shows |
| --- | --- |
| `flat.tdl` | One entity, parallel lists, optionality doing the modelling work |
| `nested.tdl` | The same domain split into entities, values, and enums |
| `collections.tdl` | Nested collections, every optionality form, qualified references |
| `targets.tdl` | `nested.tdl` plus a class and a backend mapping kept out of the model |

## Running them

```shell
tdl play examples/nested.tdl --views all   # re-render on every save
tdl ast examples/nested.tdl                # parse tree
tdl tokens examples/nested.tdl             # token stream
tdl fmt examples/flat.tdl                  # canonical formatting
```

`tdl play` with no argument uses `./scratch.tdl`, creating it from a template if missing.

## Things to try

- Drop `: Entity` from a type. Does the thing still make sense without identity?
- Delete a `?` and watch the `optional` count in the `stats` view.
- Replace `customer: Customer` with the five inlined fields from `flat.tdl` and compare `stats`.
- Break a line. The error points at the column, and parsing continues at the next declaration.
- Put a comma between two fields. Commas do not separate block members, so it is a syntax error.
- Name a field `type` or `unit`. A reserved word followed by `:` is a field name.
- Invent a constraint: `where { between(0, 100) }`. The set is open, so the parser accepts any name.
