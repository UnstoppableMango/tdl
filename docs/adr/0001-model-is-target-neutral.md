# The model is target-neutral, not generator-free

The spec's second design commitment said that everything a code generator needs lives in a `target` block, never in the model.
It came from the idea that a target block would approximate the shape of the language it maps to, which did not hold up: the spec already kept defaults, `deprecated`, doc comments, and constraints in the model because every backend reads them, while facts about the domain, such as which fields identify an entity, were pushed into target blocks and copied once per target (`Order => key(id)` in two blocks of the smoke model, an identity band on 94 kinds in unmango/apis).
We replaced the rule with a test on what a fact is about: a **model fact** is true of the domain with no target in view and belongs in the model; a **target fact** says how one target spells, lays out, numbers, or places the model and belongs in that target's block; and a fact several targets must agree on only because they share an encoding or an API style belongs to a **convention**, written once for every target that follows it.
The model never names a target, which is what still keeps `option("(google.api.field_behavior)", ...)` out of it as written.

## Consequences

- Model facts by this test, each needing its own syntax spec: which fields identify an entity (today's `key` directive), a field's lifecycle (output-only, immutable, assigned by the system), and what a field holding another entity's name refers to (`resource_reference`).
- Conventions, designed under #1425 with the `target json` idea: the wire convention's facts (`discriminant`, a property's JSON name, `closed`) and Google's resource names, whose name pattern should derive from identity and ownership in the model, leaving the service domain and the path spelling to the convention (#994).
- Target facts, unchanged: field numbers and reserved numbers, output paths, imports, and foreign type mappings.
