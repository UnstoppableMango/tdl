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
| entity marked `standard("Account")` | a `CustomField` per field it can store on the standard object, and no object file |
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

## Standard objects

Hand-written Salesforce source adds custom fields to the objects an org already declares, such as `Account`, and looks up to them.
A `standard("<ApiName>")` directive on an entity maps it to that standard object, under whatever name the model gives it:

```tdl
target salesforce for shop {
  Customer => standard("Account")
  Customer.name => standard("Name")
}
```

The org declares the object, so no object file is written: each field the entity can store is a custom field in `objects/Account/fields/`, written as on a custom object.
A lookup to the entity names `Account`, and so does an Apex member holding it, since `Account` is its SObject type.
`label` and `plural` belong to the object, which keeps its own, so on a standard entity they warn and do nothing.

`standard("<FieldApiName>")` on a field of any entity marks a field the object already has, such as `Name` or `CreatedDate`: nothing is written for it.
A `key` naming such a field warns, since only a custom field is made an external ID.
Two fields standing for one standard field warn.

The directive names a standard object or field, so a name holding `__`, a custom object's `Tier__c` or a managed package's `acme__Thing`, warns: on an entity it skips the entity, and on a field it drops the field.
The backend holds no list of standard objects or their fields, so it trusts the name.
`standard` on a declaration that is not an entity, or on its fields, warns and does nothing.
Import reading standard objects back waits on #1458, and is the rest of #1463.

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
| `standard("X")` | an entity, an entity's field | the standard object or field it stands for (see [Standard objects](#standard-objects)) |
| `prefix("X")` | the target block | prepended to every Apex type's name |
| `apiVersion("X")` | the target block | the API version each class declares, `66.0` by default |

Each of these warns and skips the declaration reaching it:

- a name that is not a letter followed by letters, digits, and single underscores, or that ends in an underscore
- a name longer than 40 characters, not counting `__c`
- an Apex name that is a reserved word, which includes `type`, `date`, and `number`
- an Apex type named like a standard type an org already declares, such as `Account`, `Contact`, `Order`, or `User`, which `name` or `prefix` resolves; an entity has no Apex type, and one marked `standard` names the org's own
- a `standard` directive naming a custom or managed-package name, one holding `__`
- two names differing only in case, since Salesforce compares names without case

## Output

The backend writes Salesforce DX source format, one file per component.
The target's `out` is a package directory such as `force-app/main/default`.

```text
objects/<Object>__c/<Object>__c.object-meta.xml
objects/<Object>__c/fields/<Field>__c.field-meta.xml
objects/<Standard>/fields/<Field>__c.field-meta.xml
classes/<Class>.cls
classes/<Class>.cls-meta.xml
```

Elements are in the order an org retrieve writes them, so generated and retrieved metadata diff cleanly.
Every file starts with `Code generated by tdl. DO NOT EDIT.` in its comment syntax.
A doc comment is an object's or a field's `description` and an Apex type's ApexDoc.
A deprecation is appended to the description, and is an `@deprecated` ApexDoc tag rather than the `@Deprecated` annotation, which Apex accepts only in a managed package.

## Import

`tdl import --from salesforce` reads the object and field metadata among its files, `objects/<Object>/<Object>.object-meta.xml` and `objects/<Object>/fields/*.field-meta.xml`, into package `salesforce`, since the source names none.
A field's object is the `objects/<Object>/` segment of its path, and a field with no object file beside it warns and is not read.
Apex is not read yet (#1459), so each class warns once, without a loss code, and `roundtrip` writes nothing yet (#1461).

A custom object is an entity, and its fields read back as generating writes them:

| Field type | TDL |
| --- | --- |
| `Text` | `string`, with `length` when it is not 255 |
| `Number`, scale 0 | `int64` |
| `Number`, scale above 0 | `decimal`, with `scale` when it is not 6 |
| `Checkbox` | `bool` |
| `Date`, `DateTime` | `date`, `instant` |
| `Lookup` | the entity its object reads as |
| restricted `Picklist` | an enum named for the field, or for its entity and the field when another enum takes the name |
| restricted `MultiselectPicklist` | a set of that enum |

A field that is not required is optional, but for a `Checkbox` and a `MultiselectPicklist`; a required one of those is read-back loss.
An external ID is the entity's `key`; a second warns.
A description is a doc comment, and its last line a deprecation when it starts with `Deprecated.`.
Names come back by the convention: `__c` stripped, a declaration in Pascal case and a field in camel case, with `name`, `label`, and `plural` written only where the convention would not regenerate them.

What has no TDL form warns and is not read:

- a field type generating never writes, such as `Time`, `Formula`, or `Email`, and a `Lookup` to a standard object or one not imported, drop the field under `lossy.unsupported`
- a field or object on a standard object, `objects/Account/fields/...`, or in a managed package's namespace, `acme__Tier__c`, drops under `lossy.unsupported`
- a validation rule, in the object file or under `validationRules/`, warns `lossy.constraint`, and a `unique` field that is not an external ID does too
- a checkbox default other than false, and a picklist value marked default, warn `lossy.default`
- every other element and folder generating does not write, or writes differently, such as a `sharingModel` other than `ReadWrite` or a `listViews` folder, is named in one `lossy.unsupported` warning per object
- a `Number` whose precision is not 18 is read and warns `lossy.unsupported`, until a `precision` directive exists (#1458)
- a `Text` length outside 1 to 255, or a scale above 18, is read without its `length` or `scale` and warns `lossy.unsupported`
- an external ID on a field other than `Text` or `Number` is not read as the key and warns `lossy.unsupported`
- an object or field whose name is another's, by case or by the convention, `Line_Item__c` beside `LineItem__c`, or a prelude declaration's, drops under `lossy.unsupported`, and a picklist's enum takes its entity's name first

Each object and field file's normal form for the round-trip corpus is the file parsed and written again without those elements.

## Checking the output

The tests check that every XML file is well formed and assert on the elements, and `checks.gen-salesforce` runs `xmllint` over the metadata generated from `testdata/gen/smoke`.
Neither checks Apex, which has no compiler outside an org.
`sf project deploy validate` against an org checks both, and CI has no org.
