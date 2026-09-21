# The Salesforce backend

Design document.

The Salesforce backend turns a resolved model into Salesforce DX source: custom object metadata for the entities, and Apex for everything else.
It is called `salesforce` in a target block, ships compiled into `tdl`, and ships again as `tdl-gen-salesforce` on `PATH`.
It is implemented in `backend/salesforce`, on the shared pieces [schema-backends.md](schema-backends.md) describes, and where it has no reason to differ from those backends it does not.

## Two halves

An org stores rows and runs Apex, and a model describes both.
An entity is something identified and stored, so it becomes a `CustomObject`.
A value, a mixin, and an enum are shapes code passes around, so each becomes an Apex type.
Apex code names an entity by its SObject type, `Order__c`, so an entity gets no Apex class of its own.

| TDL | Output |
| --- | --- |
| entity | a `CustomObject` named `<Name>__c`, with a `CustomField` per field it can store |
| value, mixin | an Apex `public class` with a public member per field |
| fieldless enum | an Apex `public enum`, and a restricted picklist wherever an object field holds it |
| fielded enum | an Apex class holding a `Kind` enum, a `kind` member, and an inner class and member per variant with fields |
| newtype, alias | expanded to its base in both halves |

Generics, classes, units, and externs are a positioned warning and the declaration reaching them is skipped, as in every schema backend.
Constraints are a warning and the declaration is still emitted.

A fielded enum has the shape of a protobuf `oneof`, because Apex has no union.
`kind` says which variant a value is, and the member for each variant not chosen is null.
A variant's class is an inner class named after the variant, so `Payment.Card`, since Apex allows one level of nesting and an org has one namespace for top-level classes.

## Objects

| TDL | Field type | Apex |
| --- | --- | --- |
| `string` | `Text`, length 255 | `String` |
| `int` | `Number`, precision 18, scale 0 | `Long` |
| `bool` | `Checkbox`, default false | `Boolean` |
| `bytes` | warn | `Blob` |
| `decimal` | `Number`, precision 18, scale 6 | `Decimal` |
| `uuid` | `Text`, length 36 | `String` |
| `instant` | `DateTime` | `Datetime` |
| `date` | `Date` | `Date` |
| `duration` | `Text`, length 64 | `String` |

A `Number` holds 18 digits, which is as many as an org stores.
An org has no duration, so one is ISO 8601 text, and no binary field, so `bytes` has no column.

A field that names another entity is a `Lookup` to that entity's object.
A field holding a fieldless enum is a `Picklist`, and one holding a `Set` of a fieldless enum is a `MultiselectPicklist`.
Both are restricted, so the org refuses a value the enum does not name.

A field that is not optional is required.
A `Checkbox` is never required, since it is never empty, and neither is a `MultiselectPicklist`, since an empty set is a value.
A required `Lookup` refuses the delete of what it points at, and an optional one clears itself.

Every other shape has no column: a `List`, a `Map`, a set of anything but a fieldless enum, a value, and a fielded enum.
Such a field is a warning and the object is written without it.
This is where the backend departs from the schema backends, which skip the whole declaration: a struct missing a field is a different type, and an object missing a column still deploys.

An object's name field is an auto number, `<Name>-{0000}`, since a model says nothing about what names a row.
Its sharing model is `ReadWrite` and it is deployed.

A `key` directive, the same one the `go` target reads, names the field that identifies an entity, and that field becomes a unique external ID.
An external ID is one `Text` or `Number` field, so a key of several fields, or of any other type, is a warning and the object is written without one.

## Apex

A collection is `List<T>`, `Set<T>`, or `Map<K, V>`.
Every Apex variable may be null, so an optional type is the type it wraps.

A member keeps its TDL name, and an enum value keeps its variant's name.
That is what lets `JSON.serialize` and `JSON.deserialize` write the shape the other wire backends describe, and it makes an Apex enum value and a picklist value the same string.

Every class declares the API version in its `.cls-meta.xml`.

## Names

An object or a field is named in Pascal case with `__c` after it, and labelled with its words capitalized and separated by spaces.
An Apex type is Pascal case, and a `prefix` directive on the target block prepends to every one, since an org has one namespace for classes and a package's names are otherwise everyone's.

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

Salesforce compares names without case, so two names differing only in case collide.
Each of these is a warning, and the declaration reaching it is skipped:

- a name that is not a letter followed by letters, digits, and single underscores, or that ends in an underscore
- a name longer than 40 characters, not counting `__c`
- an Apex name that is a reserved word, which includes `type`, `date`, and `number`
- an Apex type named like a standard type an org already declares, such as `Account`, `Contact`, `Order`, or `User`, which `name` or `prefix` resolves

## Output

The backend writes Salesforce DX source format, which is one file per component rather than one per model.
The target's `out` is meant to be a package directory such as `force-app/main/default`.

```text
objects/<Object>__c/<Object>__c.object-meta.xml
objects/<Object>__c/fields/<Field>__c.field-meta.xml
classes/<Class>.cls
classes/<Class>.cls-meta.xml
```

Elements are in the order a retrieve from an org writes them, so generated and retrieved metadata diff cleanly.
Every file starts with `Code generated by tdl. DO NOT EDIT.` in its comment syntax.
A doc comment is an object's or a field's `description` and an Apex type's ApexDoc.
A deprecation is appended to the description, and is an `@deprecated` ApexDoc tag rather than the `@Deprecated` annotation, which Apex accepts only in a managed package.

## Checking the output

The tests check that every XML file is well formed and assert on the elements, and `checks.gen-salesforce` runs `xmllint` over the metadata generated from `testdata/gen/smoke`.
Neither checks Apex, which has no compiler outside an org.
Deploying is the check that settles both, with `sf project deploy validate` against an org, and nothing in CI has one.
