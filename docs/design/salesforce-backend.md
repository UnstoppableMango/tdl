# The Salesforce backend

Design document.

The Salesforce backend turns a resolved model into Salesforce DX source: custom object metadata for the entities, and Apex for everything else.
It is called `salesforce` in a target block, ships compiled into `tdl` and as `tdl-gen-salesforce`, and is implemented in `backend/salesforce` on the shared pieces [schema-backends.md](schema-backends.md) describes.

## Two halves

An entity is identified and stored, so it becomes a `CustomObject`.
A value, a mixin, and an enum are shapes code passes around, so each becomes an Apex type.
Apex names an entity by its SObject type, `Order__c`, so an entity gets no Apex class.

| TDL | Output |
| --- | --- |
| entity | a `CustomObject` named `<Name>__c`, with a `CustomField` per field it can store |
| value, mixin | an Apex `public class` with a public member per field |
| fieldless enum | an Apex `public enum`, and a restricted picklist wherever an object field holds it |
| fielded enum | an Apex class holding a `Kind` enum, a `kind` member, and an inner class and member per variant with fields |
| newtype, alias | expanded to its base in both halves |

Generics, classes, units, and externs warn and skip the declaration reaching them; constraints warn and the declaration is still emitted.

Apex has no union, so a fielded enum has the shape of a protobuf `oneof`: `kind` says which variant a value is, and the member for each other variant is null.
A variant's class is an inner class, such as `Payment.Card`, since an org has one namespace for top-level classes.

## Objects

| TDL | Field type | Apex |
| --- | --- | --- |
| `string` | `Text`, length 255 | `String` |
| `int` | `Number`, precision 18, scale 0 | `Long` |
| `int32` | `Number`, precision 18, scale 0 | `Integer` |
| `int64` | `Number`, precision 18, scale 0 | `Long` |
| `uint32` | `Number`, precision 18, scale 0 | `Long` |
| `uint64` | `Number`, precision 18, scale 0 | `Decimal` |
| `float32` | `Number`, precision 18, scale 6 | `Double` |
| `float64` | `Number`, precision 18, scale 6 | `Double` |
| `bool` | `Checkbox`, default false | `Boolean` |
| `bytes` | warn | `Blob` |
| `decimal` | `Number`, precision 18, scale 6 | `Decimal` |
| `uuid` | `Text`, length 36 | `String` |
| `instant` | `DateTime` | `Datetime` |
| `date` | `Date` | `Date` |
| `duration` | `Text`, length 64 | `String` |

A `Number` holds 18 digits, the most an org stores; a duration is ISO 8601 text; an org has no binary field, so `bytes` has no column.

A field naming another entity is a `Lookup` to its object.
A field holding a fieldless enum is a restricted `Picklist`, and one holding a `Set` of one is a restricted `MultiselectPicklist`.

A field that is not optional is required, except a `Checkbox`, which is never empty, and a `MultiselectPicklist`, where an empty set is a value.
A required `Lookup` refuses the delete of what it points at, and an optional one clears itself.

Every other shape has no column: a `List`, a `Map`, a set of anything but a fieldless enum, a value, and a fielded enum.
Such a field warns and the object is written without it, where a schema backend would skip the declaration, because an object missing a column still deploys.

An object's name field is an auto number, `<Name>-{0000}`, since a model says nothing about what names a row.
Its sharing model is `ReadWrite` and its deployment status is deployed.

A `key` directive, as the `go` target reads it, makes the named field a unique external ID.
An external ID is one `Text` or `Number` field, so a key of several fields or of any other type warns and the object is written without one.

## Apex

A collection is `List<T>`, `Set<T>`, or `Map<K, V>`.
Every Apex variable may be null, so an optional type is the type it wraps.

A member keeps its TDL name, and an enum value keeps its variant's name, so `JSON.serialize` writes the shape the other wire backends describe and an Apex enum value matches its picklist value.

Every class declares the API version in its `.cls-meta.xml`.

## Names

An object or a field is named in Pascal case with `__c` after it, and labelled with its words capitalized and separated by spaces.
An Apex type is Pascal case, and a `prefix` directive on the target block prepends to every one, since an org has one namespace for classes.

| Directive | On | Does |
| --- | --- | --- |
| `name("X")` | a declaration, a field, a variant | the API name, before any `__c` |
| `label("X")` | an entity, a field | the label the org shows |
| `plural("X")` | an entity | the plural label, which is otherwise the label and an `s` |
| `key(field)` | an entity | the field that becomes a unique external ID |
| `length(n)` | a `Text` field | its length, from 1 to 255 |
| `scale(n)` | a `decimal` field | its decimal places |
| `prefix("X")` | the target block | prepended to every Apex type's name |
| `apiVersion("X")` | the target block | the API version each class declares, `66.0` by default |

Each of these warns and skips the declaration reaching it:

- a name that is not a letter followed by letters, digits, and single underscores, or that ends in an underscore
- a name longer than 40 characters, not counting `__c`
- an Apex name that is a reserved word, which includes `type`, `date`, and `number`
- an Apex type named like a standard type an org already declares, such as `Account`, `Contact`, `Order`, or `User`, which `name` or `prefix` resolves
- two names differing only in case, since Salesforce compares names without case

## Output

The backend writes Salesforce DX source format, one file per component.
The target's `out` is a package directory such as `force-app/main/default`.

```text
objects/<Object>__c/<Object>__c.object-meta.xml
objects/<Object>__c/fields/<Field>__c.field-meta.xml
classes/<Class>.cls
classes/<Class>.cls-meta.xml
```

Elements are in the order an org retrieve writes them, so generated and retrieved metadata diff cleanly.
Every file starts with `Code generated by tdl. DO NOT EDIT.` in its comment syntax.
A doc comment is an object's or a field's `description` and an Apex type's ApexDoc.
A deprecation is appended to the description, and is an `@deprecated` ApexDoc tag rather than the `@Deprecated` annotation, which Apex accepts only in a managed package.

## Checking the output

The tests check that every XML file is well formed and assert on the elements, and `checks.gen-salesforce` runs `xmllint` over the metadata generated from `testdata/gen/smoke`.
Neither checks Apex, which has no compiler outside an org.
`sf project deploy validate` against an org checks both, and CI has no org.
