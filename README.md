# UnstoppableMango's Type Description Language

Language for generating code and mapping types between languages.

## Usage

Nothing currently works, but this is how it works in my head.

## Converting

### From UML

```yaml
# myTypes.uml
# It looks like yaml now, it might not later. idk
name: MyTypes
types:
  someType:
    type: object
    fields:
      someField:
        type: string
```

```shell
$ um to proto myTypes.uml
Wrote 69B to ./myTypes.proto
```

```proto
// myTypes.proto
message SomeType {
  someField: string;
}
```

### To UML

```proto
// myTypes.proto
message MyType {
  string some_field = 1;
}
```

```shell
$ um from myTypes.proto
Wrote 420B to ./myTypes.uml
```

```yaml
# myTypes.uml
types:
  myType:
    type: object
    fields:
      someField:
        type: string
```

## Generating

```yaml
# myTypes.uml
# It looks like yaml now, it might not later. idk
name: MyTypes
types:
  someType:
    type: object
    fields:
      someField:
        type: string
```

```shell
$ um gen ts myTypes.uml
Wrote 69B to ./myTypes.ts
```

```ts
// myTypes.ts
export interface SomeType {
  someField: string;
}
```

## Development

`make` kinda builds everything, or at least its supposed to.
If you run `make` or `make build` everything should build, hopefully.
If it doesn't, make sure you have the stuff listed below.

### Prerequisites

Probably put some links here to install docs.

- `buf`
- `bun`
- `docker`
- `dotnet`
- `go`
- `make`
- `dprint`

Probably good to have but not needed

- `goreleaser`
- `node`
- `nvm`

### Setup

Run `make work` to configure a local `go.work`, if you want it.

### Building

Run `make gen` to generate code in `/gen`. Actually I lied. This doesn't do that yet.

Run `make build` to build everything.

Run `make docker` to build all docker images.

### Testing

Run `make test` to run all test suites.

### Workflow

Run `make lint` to lint everything.

Run `make clean` to remove local artifacts like `/.make` targets and `/node_modules`.

### Repository Structure

| Directory | Description |
|----------:|:------------|
|`/.config` | Just `dotnet` tools at the moment |
|`/.github` | GitHub configuration files |
|`/.github/actions` | GitHub actions |
|`/.github/workflows` | GitHub workflows |
|`/.idea` | JetBrains IDE configuration (yes some of this gets checked in, fight me) |
|`/.make` | Local `make` sentinel target files |
|`/.vscode` | VSCode configuration |
|`/cli` | Go CLI applications |
|`/docker` | Dockerfiles |
|`/gen` | Generated code |
|`/packages` | Node-ish ecosystem packages and applications |
|`/pkg` | Go packages |
|`/proto` | Protobuf definitions |
|`/src` | .NET ecosystem libraries and applications |

## Architecture

I've got a dozen conflicting ideas but the current path I'm working towards is a primary CLI `um` calling a "runner" CLI `um2something` and communicating between stdin and stdout.

The reason for the binary separation is so that the conversion/generation logic can be written as close to the ecosystem as possible (i.e. we write the typescript converter in TS/JS so we have easy programatic access to the `typescript` package).

I'm aware of gRPC being used for IPC on unix sockets so I thought it could be fun and at least semi-correct to have the two processes communicate this way.
As fun as that might be, I'm worried I might be pushing the limits of "how over-engineered does this _really_ need to be".

CLI tools should be really snappy so the overhead of setting up a gRPC server might be ridiculous. If it's not though... I might do that. It sounds really cool "the user CLI communicates with the runner CLI via gRPC on a unix domain socket".

If we want to REALLY over-engineer everything I was thinking we could have a little broker do-dad that hangs out in the background and loads up plugins that the user CLI can call to convert things. It would be fun and ridiculous.

## Notes to self

As I'm typing out these examples I'm realizing there is a lot of overlap between the commands I've hallucinated.
Intuitively these two commands should behave the same

```shell
um to ts
um gen ts
```

There really isn't a need to differentiate between a specification like protobuf and a language like typescript, they won't ever overlap.

In the CLI tools I can just alias `to` and `gen`.
However, I naively started implementing a number of abstractions for generators and converters that will take a bit of refactoring.
This is what I get for premature optimization and over-engineering shit.
